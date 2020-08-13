// Copyright © 2020 The Tekton Authors.
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

package clustertask

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	ctactions "github.com/tektoncd/cli/pkg/clustertask"
	"github.com/tektoncd/cli/pkg/cmd/pipelineresource"
	"github.com/tektoncd/cli/pkg/cmd/taskrun"
	"github.com/tektoncd/cli/pkg/file"
	"github.com/tektoncd/cli/pkg/flags"
	"github.com/tektoncd/cli/pkg/labels"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/params"
	"github.com/tektoncd/cli/pkg/pods"
	"github.com/tektoncd/cli/pkg/task"
	tractions "github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/cli/pkg/workspaces"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

var (
	errNoClusterTask      = errors.New("missing ClusterTask name")
	errInvalidClusterTask = "ClusterTask name %s does not exist"
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
	TimeOut            string
	DryRun             bool
	Output             string
	Workspaces         []string
	UseParamDefaults   bool
	clustertask        *v1beta1.ClusterTask
	askOpts            survey.AskOpt
	TektonOptions      flags.TektonOptions
	PodTemplate        string
}

// NameArg validates that the first argument is a valid clustertask name
func NameArg(args []string, p cli.Params, opt *startOptions) error {
	if len(args) == 0 {
		return errNoClusterTask
	}

	c, err := p.Clients()
	if err != nil {
		return err
	}

	name := args[0]
	ct, err := ctactions.Get(c, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf(errInvalidClusterTask, name)
	}
	opt.clustertask = ct
	if ct.Spec.Params != nil {
		params.FilterParamsByType(ct.Spec.Params)
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

	eg := `Start ClusterTask foo by creating a TaskRun named "foo-run-xyz123" in namespace 'bar':

    tkn clustertask start foo -n bar

or

    tkn ct start foo -n bar

For params value, if you want to provide multiple values, provide them comma separated
like cat,foo,bar
`

	c := &cobra.Command{
		Use:   "start",
		Short: "Start ClusterTasks",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Example:      eg,
		SilenceUsage: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if opt.UseParamDefaults && opt.Last {
				return errors.New("cannot use --last with --use-param-defaults option")
			}
			if opt.DryRun {
				format := strings.ToLower(opt.Output)
				if format != "" && format != "json" && format != "yaml" {
					return fmt.Errorf("output format specified is %s but must be yaml or json", opt.Output)
				}
			}
			if err := flags.InitParams(p, cmd); err != nil {
				return err
			}
			return NameArg(args, p, &opt)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opt.stream = &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			opt.TektonOptions = flags.GetTektonOptions(cmd)
			return startClusterTask(opt, args)
		},
	}

	c.Flags().StringSliceVarP(&opt.InputResources, "inputresource", "i", []string{}, "pass the input resource name and ref as name=ref")
	c.Flags().StringSliceVarP(&opt.OutputResources, "outputresource", "o", []string{}, "pass the output resource name and ref as name=ref")
	c.Flags().StringArrayVarP(&opt.Params, "param", "p", []string{}, "pass the param as key=value for string type, or key=value1,value2,... for array type")
	c.Flags().StringVarP(&opt.ServiceAccountName, "serviceaccount", "s", "", "pass the serviceaccount name")
	flags.AddShellCompletion(c.Flags().Lookup("serviceaccount"), "__kubectl_get_serviceaccount")
	c.Flags().BoolVarP(&opt.Last, "last", "L", false, "re-run the ClusterTask using last TaskRun values")
	c.Flags().StringSliceVarP(&opt.Labels, "labels", "l", []string{}, "pass labels as label=value.")
	c.Flags().StringArrayVarP(&opt.Workspaces, "workspace", "w", []string{}, "pass one or more workspaces to map to the corresponding physical volumes as name=name,claimName=pvcName or name=name,emptyDir=")
	c.Flags().BoolVarP(&opt.ShowLog, "showlog", "", false, "show logs right after starting the ClusterTask")
	c.Flags().StringVar(&opt.TimeOut, "timeout", "", "timeout for TaskRun")
	c.Flags().BoolVarP(&opt.DryRun, "dry-run", "", false, "preview TaskRun without running it")
	c.Flags().StringVarP(&opt.Output, "output", "", "", "format of TaskRun dry-run (yaml or json)")
	c.Flags().StringVar(&opt.PodTemplate, "pod-template", "", "local or remote file containing a PodTemplate definition")
	c.Flags().BoolVar(&opt.UseParamDefaults, "use-param-defaults", false, "use default parameter values without prompting for input")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_clustertask")

	return c
}

func startClusterTask(opt startOptions, args []string) error {
	tr := &v1beta1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "TaskRun",
		},
		ObjectMeta: metav1.ObjectMeta{},
	}

	cs, err := opt.cliparams.Clients()
	if err != nil {
		return err
	}

	ctname := args[0]
	tr.Spec = v1beta1.TaskRunSpec{
		TaskRef: &v1beta1.TaskRef{
			Name: ctname,
			Kind: v1beta1.ClusterTaskKind, //Specify TaskRun is for a ClusterTask kind
		},
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

	tr.ObjectMeta.GenerateName = ctname + "-run-"

	//TaskRuns are namespaced so using same LastRun method as Task
	if opt.Last {
		trLast, err := task.LastRun(cs, ctname, opt.cliparams.Namespace(), "ClusterTask")
		if err != nil {
			return err
		}
		tr.Spec.Resources = trLast.Spec.Resources
		tr.Spec.Params = trLast.Spec.Params
		tr.Spec.ServiceAccountName = trLast.Spec.ServiceAccountName
		tr.Spec.Workspaces = trLast.Spec.Workspaces
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

	workspaces, err := workspaces.Merge(tr.Spec.Workspaces, opt.Workspaces, cs.HTTPClient)
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

	podTemplateLocation := opt.PodTemplate
	if podTemplateLocation != "" {
		podTemplate, err := pods.ParsePodTemplate(cs.HTTPClient, podTemplateLocation, file.IsYamlFile(), fmt.Errorf("invalid file format for %s: .yaml or .yml file extension and format required", podTemplateLocation))
		if err != nil {
			return err
		}
		tr.Spec.PodTemplate = &podTemplate
	}

	if opt.DryRun {
		return taskRunDryRun(cs, opt.Output, opt.stream, tr)
	}

	trCreated, err := tractions.Create(cs, tr, metav1.CreateOptions{}, opt.cliparams.Namespace())
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
		trConverted := tractions.ConvertFrom(tr)
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
	intOpts := options.InteractiveOpts{
		Stream:    opt.stream,
		CliParams: opt.cliparams,
		AskOpts:   opt.askOpts,
		Ns:        opt.cliparams.Namespace(),
	}

	if opt.clustertask.Spec.Resources != nil && !opt.Last {
		if len(opt.InputResources) == 0 {
			if err := intOpts.ClusterTaskInputResources(opt.clustertask, createPipelineResource); err != nil {
				return err
			}
			opt.InputResources = append(opt.InputResources, intOpts.InputResources...)
		}
		if len(opt.OutputResources) == 0 {
			if err := intOpts.ClusterTaskOutputResources(opt.clustertask, createPipelineResource); err != nil {
				return err
			}
			opt.OutputResources = append(opt.OutputResources, intOpts.OutputResources...)
		}
	}

	params.FilterParamsByType(opt.clustertask.Spec.Params)
	if len(opt.Params) == 0 && !opt.Last {
		if err := intOpts.ClusterTaskParams(opt.clustertask, opt.UseParamDefaults); err != nil {
			return err
		}
		opt.Params = append(opt.Params, intOpts.Params...)
	}

	if len(opt.Workspaces) == 0 && !opt.Last {
		if err := intOpts.ClusterTaskWorkspaces(opt.clustertask); err != nil {
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
