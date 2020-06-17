// Copyright © 2019 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cmd/pipelineresource"
	"github.com/tektoncd/cli/pkg/cmd/taskrun"
	"github.com/tektoncd/cli/pkg/file"
	"github.com/tektoncd/cli/pkg/flags"
	"github.com/tektoncd/cli/pkg/labels"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/params"
	"github.com/tektoncd/cli/pkg/task"
	traction "github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/cli/pkg/workspaces"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/yaml"
)

var (
	errNoTask      = errors.New("missing Task name")
	errInvalidTask = "Task name %s does not exist in namespace %s"
)

const invalidResource = "invalid input format for resource parameter: "

type startOptions struct {
	cliparams          cli.Params
	stream             *cli.Stream
	Params             []string
	InputResources     []string
	OutputResources    []string
	ServiceAccountName string
	Last               bool
	Labels             []string
	ShowLog            bool
	Filename           string
	TimeOut            string
	DryRun             bool
	Output             string
	UseTaskRun         string
	PrefixName         string
	Workspaces         []string
	task               *v1beta1.Task
	askOpts            survey.AskOpt
	TektonOptions      flags.TektonOptions
}

// NameArg validates that the first argument is a valid task name
func NameArg(args []string, p cli.Params, opt *startOptions) error {
	if len(args) == 0 {
		return errNoTask
	}

	c, err := p.Clients()
	if err != nil {
		return err
	}

	name, ns := args[0], p.Namespace()
	t, err := task.Get(c, name, metav1.GetOptions{}, ns)
	if err != nil {
		return fmt.Errorf(errInvalidTask, name, ns)
	}
	opt.task = t
	if t.Spec.Params != nil {
		params.FilterParamsByType(t.Spec.Params)
	}

	return nil
}

func startCommand(p cli.Params) *cobra.Command {
	opt := startOptions{
		cliparams: p,
		askOpts: func(opt *survey.AskOptions) error {
			opt.Stdio = terminal.Stdio{
				In:  os.Stdin,
				Out: os.Stdout,
				Err: os.Stderr,
			}
			return nil
		},
	}

	c := &cobra.Command{
		Use:   "start [RESOURCES...] [PARAMS...] [SERVICEACCOUNT]",
		Short: "Start Tasks",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Example: `Start Task foo by creating a TaskRun named "foo-run-xyz123" from namespace 'bar':

    tkn task start foo -s ServiceAccountName -n bar

The Task can either be specified by reference in a cluster using the positional argument
or in a file using the --filename argument.

For params values, if you want to provide multiple values, provide them comma separated
like cat,foo,bar
`,
		SilenceUsage: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := flags.InitParams(p, cmd); err != nil {
				return err
			}
			if opt.DryRun {
				format := strings.ToLower(opt.Output)
				if format != "" && format != "json" && format != "yaml" {
					return fmt.Errorf("output format specified is %s but must be yaml or json", opt.Output)
				}
			}
			if len(args) != 0 {
				return NameArg(args, p, &opt)
			}
			if opt.Filename == "" {
				return errors.New("either a Task name or a --filename argument must be supplied")
			}
			if opt.Filename != "" && opt.Last {
				return errors.New("cannot use --last option with --filename option")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opt.stream = &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if opt.Last && opt.UseTaskRun != "" {
				return fmt.Errorf("using --last and --use-taskrun are not compatible")
			}

			opt.TektonOptions = flags.GetTektonOptions(cmd)
			return startTask(opt, args)
		},
	}

	c.Flags().StringSliceVarP(&opt.InputResources, "inputresource", "i", []string{}, "pass the input resource name and ref as name=ref")
	c.Flags().StringSliceVarP(&opt.OutputResources, "outputresource", "o", []string{}, "pass the output resource name and ref as name=ref")
	c.Flags().StringArrayVarP(&opt.Params, "param", "p", []string{}, "pass the param as key=value for string type, or key=value1,value2,... for array type")
	c.Flags().StringVarP(&opt.ServiceAccountName, "serviceaccount", "s", "", "pass the serviceaccount name")
	flags.AddShellCompletion(c.Flags().Lookup("serviceaccount"), "__kubectl_get_serviceaccount")
	c.Flags().BoolVarP(&opt.Last, "last", "L", false, "re-run the Task using last TaskRun values")
	c.Flags().StringVarP(&opt.UseTaskRun, "use-taskrun", "", "", "specify a TaskRun name to use its values to re-run the TaskRun")
	flags.AddShellCompletion(c.Flags().Lookup("use-taskrun"), "__tkn_get_taskrun")
	c.Flags().StringSliceVarP(&opt.Labels, "labels", "l", []string{}, "pass labels as label=value.")
	c.Flags().StringArrayVarP(&opt.Workspaces, "workspace", "w", []string{}, "pass the workspace.")
	c.Flags().BoolVarP(&opt.ShowLog, "showlog", "", false, "show logs right after starting the Task")
	c.Flags().StringVarP(&opt.Filename, "filename", "f", "", "local or remote file name containing a Task definition to start a TaskRun")
	c.Flags().StringVarP(&opt.TimeOut, "timeout", "", "", "timeout for TaskRun")
	c.Flags().BoolVarP(&opt.DryRun, "dry-run", "", false, "preview TaskRun without running it")
	c.Flags().StringVarP(&opt.Output, "output", "", "", "format of TaskRun dry-run (yaml or json)")
	c.Flags().StringVarP(&opt.PrefixName, "prefix-name", "", "", "specify a prefix for the TaskRun name (must be lowercase alphanumeric characters)")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_task")

	return c
}

func parseTask(taskLocation string, p cli.Params) (*v1beta1.Task, error) {
	b, err := file.LoadFileContent(p, taskLocation, file.IsYamlFile(), fmt.Errorf("invalid file format for %s: .yaml or .yml file extension and format required", taskLocation))
	if err != nil {
		return nil, err
	}

	m := map[string]interface{}{}
	err = yaml.UnmarshalStrict(b, &m)
	if err != nil {
		return nil, err
	}
	task := v1beta1.Task{}
	if m["apiVersion"] == "tekton.dev/v1alpha1" {
		v1alpha1Task := v1alpha1.Task{}
		if err := yaml.UnmarshalStrict(b, &v1alpha1Task); err != nil {
			return nil, err
		}
		if err := v1alpha1Task.ConvertTo(context.Background(), &task); err != nil {
			return nil, err
		}
		task.TypeMeta.APIVersion = "tekton.dev/v1alpha1"
		task.TypeMeta.APIVersion = "Task"
		return &task, nil
	}

	if err := yaml.UnmarshalStrict(b, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func useTaskRunFrom(opt startOptions, tr *v1beta1.TaskRun, cs *cli.Clients, tname string) error {
	var (
		trUsed *v1beta1.TaskRun
		err    error
	)
	if opt.Last {
		trUsed, err = task.LastRun(cs, tname, opt.cliparams.Namespace(), "Task")
		if err != nil {
			return err
		}
	} else if opt.UseTaskRun != "" {
		trUsed, err = traction.Get(cs, opt.UseTaskRun, metav1.GetOptions{}, opt.cliparams.Namespace())
		if err != nil {
			return err
		}
	}

	// if --prefix-name is specified by user, allow name to be used for taskrun
	if len(trUsed.ObjectMeta.GenerateName) > 0 && opt.PrefixName == "" {
		tr.ObjectMeta.GenerateName = trUsed.ObjectMeta.GenerateName
	} else if opt.PrefixName == "" {
		tr.ObjectMeta.GenerateName = trUsed.ObjectMeta.Name + "-"
	}

	tr.Spec.Resources = trUsed.Spec.Resources
	tr.Spec.Params = trUsed.Spec.Params
	tr.Spec.ServiceAccountName = trUsed.Spec.ServiceAccountName
	tr.Spec.Workspaces = trUsed.Spec.Workspaces
	tr.Spec.Timeout = trUsed.Spec.Timeout

	return nil
}

func startTask(opt startOptions, args []string) error {
	cs, err := opt.cliparams.Clients()
	if err != nil {
		return err
	}

	tr := &v1beta1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "TaskRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: opt.cliparams.Namespace(),
		},
	}

	var tname string
	if len(args) > 0 {
		tname = args[0]
		tr.Spec = v1beta1.TaskRunSpec{
			TaskRef: &v1beta1.TaskRef{Name: tname},
		}
	} else {
		task, err := parseTask(opt.Filename, opt.cliparams)
		if err != nil {
			return err
		}
		opt.task = task
		tname = task.ObjectMeta.Name

		if task.Spec.Params != nil {
			params.FilterParamsByType(task.Spec.Params)
		}

		tr.Spec = v1beta1.TaskRunSpec{
			TaskSpec: &task.Spec,
		}
	}

	if err := opt.getInputs(); err != nil {
		return err
	}

	if opt.TimeOut != "" {
		timeoutDuration, err := time.ParseDuration(opt.TimeOut)
		if err != nil {
			return err
		}
		tr.Spec.Timeout = &metav1.Duration{Duration: timeoutDuration}
	}

	if opt.PrefixName == "" {
		tr.ObjectMeta.GenerateName = tname + "-run-"
	} else {
		tr.ObjectMeta.GenerateName = opt.PrefixName + "-"
	}

	if opt.Last || opt.UseTaskRun != "" {
		err := useTaskRunFrom(opt, tr, cs, tname)
		if err != nil {
			return err
		}
	}

	if tr.Spec.Resources == nil {
		tr.Spec.Resources = &v1beta1.TaskRunResources{}
	}
	inputRes, err := mergeRes(tr.Spec.Resources.Inputs, opt.InputResources)
	if err != nil {
		return err
	}
	tr.Spec.Resources.Inputs = inputRes

	outRes, err := mergeRes(tr.Spec.Resources.Outputs, opt.OutputResources)
	if err != nil {
		return err
	}
	tr.Spec.Resources.Outputs = outRes

	labels, err := labels.MergeLabels(tr.ObjectMeta.Labels, opt.Labels)
	if err != nil {
		return err
	}
	tr.ObjectMeta.Labels = labels

	workspaces, err := workspaces.Merge(tr.Spec.Workspaces, opt.Workspaces)
	if err != nil {
		return err
	}
	tr.Spec.Workspaces = workspaces

	param, err := params.MergeParam(tr.Spec.Params, opt.Params)
	if err != nil {
		return err
	}
	tr.Spec.Params = param

	if len(opt.ServiceAccountName) > 0 {
		tr.Spec.ServiceAccountName = opt.ServiceAccountName
	}

	if opt.DryRun {
		return taskRunDryRun(cs, opt.Output, opt.stream, tr)
	}

	trCreated, err := traction.Create(cs, tr, metav1.CreateOptions{}, opt.cliparams.Namespace())
	if err != nil {
		return err
	}

	fmt.Fprintf(opt.stream.Out, "TaskRun started: %s\n", trCreated.Name)
	if !opt.ShowLog {
		inOrderString := "\nIn order to track the TaskRun progress run:\ntkn taskrun "
		if opt.TektonOptions.Context != "" {
			inOrderString += fmt.Sprintf("--context=%s ", opt.TektonOptions.Context)
		}
		inOrderString += fmt.Sprintf("logs %s -f -n %s\n", trCreated.Name, trCreated.Namespace)

		fmt.Fprint(opt.stream.Out, inOrderString)
		return nil
	}

	fmt.Fprintf(opt.stream.Out, "Waiting for logs to be available...\n")
	runLogOpts := &options.LogOptions{
		TaskrunName: trCreated.Name,
		Stream:      opt.stream,
		Follow:      true,
		Params:      opt.cliparams,
		AllSteps:    false,
	}
	return taskrun.Run(runLogOpts)
}

func mergeRes(r []v1beta1.TaskResourceBinding, optRes []string) ([]v1beta1.TaskResourceBinding, error) {
	res, err := parseRes(optRes)
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return r, nil
	}

	for i := range r {
		if v, ok := res[r[i].Name]; ok {
			r[i] = v
			delete(res, v.Name)
		}
	}
	for _, v := range res {
		r = append(r, v)
	}

	sort.Slice(r, func(i, j int) bool { return r[i].Name < r[j].Name })
	return r, nil
}

func parseRes(res []string) (map[string]v1beta1.TaskResourceBinding, error) {
	resources := map[string]v1beta1.TaskResourceBinding{}
	for _, v := range res {
		r := strings.SplitN(v, "=", 2)
		if len(r) != 2 {
			return nil, errors.New(invalidResource + v)
		}
		resources[r[0]] = v1beta1.TaskResourceBinding{
			PipelineResourceBinding: v1beta1.PipelineResourceBinding{
				Name: r[0],
				ResourceRef: &v1beta1.PipelineResourceRef{
					Name: r[1],
				},
			},
		}
	}
	return resources, nil
}

func taskRunDryRun(c *cli.Clients, output string, s *cli.Stream, tr *v1beta1.TaskRun) error {
	trWithVersion, err := convertedTrVersion(c, tr)
	if err != nil {
		return err
	}

	format := strings.ToLower(output)
	if format == "" || format == "yaml" {
		trBytes, err := yaml.Marshal(trWithVersion)
		if err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "%s", trBytes)
	}

	if format == "json" {
		trBytes, err := json.MarshalIndent(trWithVersion, "", "\t")
		if err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "%s\n", trBytes)
	}

	return nil
}

func getAPIVersion(discovery discovery.DiscoveryInterface) (string, error) {
	_, err := discovery.ServerResourcesForGroupVersion("tekton.dev/v1beta1")
	if err != nil {
		_, err = discovery.ServerResourcesForGroupVersion("tekton.dev/v1alpha1")
		if err != nil {
			return "", fmt.Errorf("couldn't get available Tekton api versions from server")
		}
		return "tekton.dev/v1alpha1", nil
	}
	return "tekton.dev/v1beta1", nil
}

func convertedTrVersion(c *cli.Clients, tr *v1beta1.TaskRun) (interface{}, error) {
	version, err := getAPIVersion(c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if version == "tekton.dev/v1alpha1" {
		trConverted := traction.ConvertFrom(tr)
		trConverted.APIVersion = version
		trConverted.Kind = "TaskRun"
		if err != nil {
			return nil, err
		}
		return &trConverted, nil
	}

	return tr, nil
}

func (opt *startOptions) getInputs() error {
	if opt.Last && opt.UseTaskRun != "" {
		fmt.Fprintf(opt.stream.Err, "option --last and option --use-taskrun are not compatible \n")
		return nil
	}

	intOpts := options.InteractiveOpts{
		Stream:    opt.stream,
		CliParams: opt.cliparams,
		AskOpts:   opt.askOpts,
		Ns:        opt.cliparams.Namespace(),
	}

	if opt.task.Spec.Resources != nil && !opt.Last && opt.UseTaskRun == "" {
		if len(opt.InputResources) == 0 {
			if err := intOpts.TaskInputResources(opt.task, createPipelineResource); err != nil {
				return err
			}
			opt.InputResources = append(opt.InputResources, intOpts.InputResources...)
		}
		if len(opt.OutputResources) == 0 {
			if err := intOpts.TaskOutputResources(opt.task, createPipelineResource); err != nil {
				return err
			}
			opt.OutputResources = append(opt.OutputResources, intOpts.OutputResources...)
		}
	}

	params.FilterParamsByType(opt.task.Spec.Params)
	if len(opt.Params) == 0 && !opt.Last && opt.UseTaskRun == "" {
		if err := intOpts.TaskParams(opt.task); err != nil {
			return err
		}
		opt.Params = append(opt.Params, intOpts.Params...)
	}

	if len(opt.Workspaces) == 0 && !opt.Last && opt.UseTaskRun == "" {
		if err := intOpts.TaskWorkspaces(opt.task); err != nil {
			return err
		}
		opt.Workspaces = append(opt.Workspaces, intOpts.Workspaces...)
	}

	return nil
}

func createPipelineResource(resType v1alpha1.PipelineResourceType, askOpt survey.AskOpt, p cli.Params, s *cli.Stream) (*v1alpha1.PipelineResource, error) {
	res := pipelineresource.Resource{
		AskOpts: askOpt,
		Params:  p,
		PipelineResource: v1alpha1.PipelineResource{
			ObjectMeta: metav1.ObjectMeta{Namespace: p.Namespace()},
			Spec:       v1alpha1.PipelineResourceSpec{Type: resType},
		}}

	if err := res.AskMeta(); err != nil {
		return nil, err
	}

	resourceTypeParams := map[v1alpha1.PipelineResourceType]func() error{
		v1alpha1.PipelineResourceTypeGit:         res.AskGitParams,
		v1alpha1.PipelineResourceTypeStorage:     res.AskStorageParams,
		v1alpha1.PipelineResourceTypeImage:       res.AskImageParams,
		v1alpha1.PipelineResourceTypeCluster:     res.AskClusterParams,
		v1alpha1.PipelineResourceTypePullRequest: res.AskPullRequestParams,
		v1alpha1.PipelineResourceTypeCloudEvent:  res.AskCloudEventParams,
	}
	if res.PipelineResource.Spec.Type != "" {
		if err := resourceTypeParams[res.PipelineResource.Spec.Type](); err != nil {
			return nil, err
		}
	}
	cs, err := p.Clients()
	if err != nil {
		return nil, err
	}
	newRes, err := cs.Resource.TektonV1alpha1().PipelineResources(p.Namespace()).Create(&res.PipelineResource)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(s.Out, "New %s resource \"%s\" has been created\n", newRes.Spec.Type, newRes.Name)
	return newRes, nil
}
