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
	"os"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	remoteimg "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/tektoncd/cli/pkg/bundle"
	"k8s.io/apimachinery/pkg/runtime"

	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cmd/taskrun"
	"github.com/tektoncd/cli/pkg/file"
	"github.com/tektoncd/cli/pkg/flags"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/labels"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/params"
	"github.com/tektoncd/cli/pkg/pods"
	traction "github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/cli/pkg/workspaces"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

var (
	errNoTask      = errors.New("missing Task name")
	errInvalidTask = "Task name %s does not exist in namespace %s"
)

type startOptions struct {
	cliparams             cli.Params
	stream                *cli.Stream
	Params                []string
	ServiceAccountName    string
	Last                  bool
	Labels                []string
	ShowLog               bool
	Filename              string
	Image                 string
	TimeOut               string
	DryRun                bool
	Output                string
	UseTaskRun            string
	PrefixName            string
	Workspaces            []string
	task                  *v1beta1.Task
	askOpts               survey.AskOpt
	TektonOptions         flags.TektonOptions
	UseParamDefaults      bool
	PodTemplate           string
	SkipOptionalWorkspace bool
	remoteOptions         bundle.RemoteOptions
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
	t, err := getTaskV1beta1(taskGroupResource, c, name, ns)
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

The Task can either be specified by reference in a cluster using the positional argument, 
an oci bundle using the --image argument and the positional argument or in a file using the --filename argument

Authentication:
	There are three ways to authenticate against your registry when using the --image argument.
	1. By default, your docker.config in your home directory and podman's auth.json are used.
	2. Additionally, you can supply a Bearer Token via --remote-bearer
	3. Additionally, you can use Basic auth via --remote-username and --remote-password

For params values, if you want to provide multiple values, provide them comma separated
like cat,foo,bar

For passing the workspaces via flags:

- In case of emptyDir, you can pass it like -w name=my-empty-dir,emptyDir=
- In case of configMap, you can pass it like -w name=my-config,config=rpg,item=ultimav=1
- In case of secrets, you can pass it like -w name=my-secret,secret=secret-name
- In case of pvc, you can pass it like -w name=my-pvc,claimName=pvc1
- In case of volumeClaimTemplate, you can pass it like -w name=my-volume-claim-template,volumeClaimTemplateFile=workspace-template.yaml
  but before you need to create a workspace-template.yaml file. Sample contents of the file are as follows:
  spec:
   accessModes:
     - ReadWriteOnce
   resources:
     requests:
       storage: 1Gi
- In case of binding a CSI workspace, you can pass it like -w name=my-csi,csiFile=csi.yaml
  but you need to create a csi.yaml file before hand. Sample contents of the file are as follows:
  
  driver: secrets-store.csi.k8s.io
  readOnly: true
  volumeAttributes:
    secretProviderClass: "vault-database"
`,
		SilenceUsage:      true,
		ValidArgsFunction: formatted.ParentCompletion,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := flags.InitParams(p, cmd); err != nil {
				return err
			}
			if opt.UseParamDefaults && (opt.Last || opt.UseTaskRun != "") {
				return errors.New("cannot use --last or --use-taskrun options with --use-param-defaults option")
			}
			format := strings.ToLower(opt.Output)
			if format != "" && format != "json" && format != "yaml" {
				return fmt.Errorf("output format specified is %s but must be yaml or json", opt.Output)
			}
			if format != "" && opt.ShowLog {
				return errors.New("cannot use --output option with --showlog option")
			}
			// classic with no image
			if len(args) != 0 && opt.Image == "" {
				return NameArg(args, p, &opt)
			}
			// image but no task name
			if len(args) == 0 && opt.Image != "" {
				return errors.New("task name required with --image option")
			}
			// trying to use a file and image
			if opt.Filename != "" && opt.Image != "" {
				return errors.New("cannot use --filename option with --image option")
			}
			// not passing enough
			if opt.Filename == "" && len(args) == 0 {
				return errors.New("either a Task name or a --filename argument must be supplied")
			}
			// cant mix image/file with --last ( I think this is true? )
			if (opt.Filename != "" || opt.Image != "") && opt.Last {
				return errors.New("cannot use --last option with --filename or --image option")
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

	c.Flags().StringArrayVarP(&opt.Params, "param", "p", []string{}, "pass the param as key=value for string type, or key=value1,value2,... for array type, or key=\"key1:value1, key2:value2\" for object type")
	c.Flags().StringVarP(&opt.ServiceAccountName, "serviceaccount", "s", "", "pass the serviceaccount name")
	_ = c.RegisterFlagCompletionFunc("serviceaccount",
		func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return formatted.BaseCompletion("serviceaccount", args)
		},
	)
	c.Flags().BoolVarP(&opt.Last, "last", "L", false, "re-run the Task using last TaskRun values")
	c.Flags().StringVarP(&opt.UseTaskRun, "use-taskrun", "", "", "specify a TaskRun name to use its values to re-run the TaskRun")
	_ = c.RegisterFlagCompletionFunc("use-taskrun",
		func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return formatted.BaseCompletion("taskrun", args)
		},
	)
	c.Flags().StringSliceVarP(&opt.Labels, "labels", "l", []string{}, "pass labels as label=value.")
	c.Flags().StringArrayVarP(&opt.Workspaces, "workspace", "w", []string{}, "pass one or more workspaces to map to the corresponding physical volumes")
	c.Flags().BoolVarP(&opt.ShowLog, "showlog", "", false, "show logs right after starting the Task")
	c.Flags().StringVarP(&opt.Filename, "filename", "f", "", "local or remote file name containing a Task definition to start a TaskRun")
	c.Flags().StringVarP(&opt.Image, "image", "i", "", "use an oci bundle")

	c.Flags().StringVarP(&opt.TimeOut, "timeout", "", "", "timeout for TaskRun")
	c.Flags().BoolVarP(&opt.DryRun, "dry-run", "", false, "preview TaskRun without running it")
	c.Flags().StringVarP(&opt.Output, "output", "", "", "format of TaskRun (yaml or json)")
	c.Flags().StringVarP(&opt.PrefixName, "prefix-name", "", "", "specify a prefix for the TaskRun name (must be lowercase alphanumeric characters)")
	c.Flags().BoolVarP(&opt.UseParamDefaults, "use-param-defaults", "", false, "use default parameter values without prompting for input")
	c.Flags().StringVar(&opt.PodTemplate, "pod-template", "", "local or remote file containing a PodTemplate definition")
	c.Flags().BoolVarP(&opt.SkipOptionalWorkspace, "skip-optional-workspace", "", false, "skips the prompt for optional workspaces")
	bundle.AddRemoteFlags(c.Flags(), &opt.remoteOptions)

	return c
}

func parseTask(b []byte) (*v1beta1.Task, error) {

	m := map[string]interface{}{}
	err := yaml.UnmarshalStrict(b, &m)
	if err != nil {
		return nil, err
	}

	if m["apiVersion"] == "tekton.dev/v1alpha1" {
		return nil, fmt.Errorf("v1alpha1 is no longer supported")
	}

	task := v1beta1.Task{}
	if m["apiVersion"] == "tekton.dev/v1" {
		taskV1 := v1.Task{}
		if err := yaml.UnmarshalStrict(b, &taskV1); err != nil {
			return nil, err
		}
		err = task.ConvertFrom(context.Background(), &taskV1)
		if err != nil {
			return nil, err
		}
	} else {
		if err := yaml.UnmarshalStrict(b, &task); err != nil {
			return nil, err
		}
	}

	err = params.ValidateParamType(task.Spec.Params)
	if err != nil {
		return nil, err
	}

	return &task, nil
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

	switch {
	case len(args) > 0 && opt.Image == "":
		tname = args[0]
		tr.Spec = v1beta1.TaskRunSpec{
			TaskRef: &v1beta1.TaskRef{Name: tname},
		}
	case opt.Filename != "":
		b, err := file.LoadFileContent(cs.HTTPClient, opt.Filename, file.IsYamlFile(), fmt.Errorf("invalid file format for %s: .yaml or .yml file extension and format required", opt.Filename))
		if err != nil {
			return err
		}
		task, err := parseTask(b)
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
	case opt.Image != "":
		ref, err := name.ParseReference(opt.Image)
		if err != nil {
			return err
		}
		img, err := remoteimg.Image(ref, opt.remoteOptions.ToOptions()...)
		if err != nil {
			return err
		}
		err = bundle.Get(img, "task", args[0], func(_, _, _ string, _ runtime.Object, b []byte) {
			task, intErr := parseTask(b)
			if intErr != nil {
				err = intErr
			}
			opt.task = task
			tname = task.ObjectMeta.Name

			if task.Spec.Params != nil {
				params.FilterParamsByType(task.Spec.Params)
			}

			tr.Spec = v1beta1.TaskRunSpec{
				TaskSpec: &task.Spec,
			}
		})
		if err != nil {
			return err
		}
	}

	if err := opt.getInputs(); err != nil {
		return err
	}

	if opt.Last || opt.UseTaskRun != "" {
		taskRunOpts := options.TaskRunOpts{
			CliParams:  opt.cliparams,
			Last:       opt.Last,
			UseTaskRun: opt.UseTaskRun,
			PrefixName: opt.PrefixName,
		}
		err := taskRunOpts.UseTaskRunFrom(tr, cs, tname, "Task")
		if err != nil {
			return err
		}
	}

	if opt.PrefixName == "" && !opt.Last && opt.UseTaskRun == "" {
		tr.ObjectMeta.GenerateName = tname + "-run-"
	} else if opt.PrefixName != "" {
		tr.ObjectMeta.GenerateName = opt.PrefixName + "-"
	}

	if opt.TimeOut != "" {
		timeoutDuration, err := time.ParseDuration(opt.TimeOut)
		if err != nil {
			return err
		}
		tr.Spec.Timeout = &metav1.Duration{Duration: timeoutDuration}
	}

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
		gvr, err := actions.GetGroupVersionResource(taskrunGroupResource, cs.Tekton.Discovery())
		if err != nil {
			return err
		}
		if gvr.Version == "v1" {
			var trv1 v1.TaskRun
			err = tr.ConvertTo(context.Background(), &trv1)
			if err != nil {
				return err
			}
			trv1.Kind = "TaskRun"
			trv1.APIVersion = "tekton.dev/v1"
			return printTaskRun(opt.Output, opt.stream, &trv1)
		}
		return printTaskRun(opt.Output, opt.stream, tr)
	}

	trCreated, err := traction.Create(cs, tr, metav1.CreateOptions{}, opt.cliparams.Namespace())
	if err != nil {
		return err
	}

	if opt.Output != "" {
		gvr, err := actions.GetGroupVersionResource(taskrunGroupResource, cs.Tekton.Discovery())
		if err != nil {
			return err
		}
		if gvr.Version == "v1" {
			var trv1 v1.TaskRun
			err = trCreated.ConvertTo(context.Background(), &trv1)
			if err != nil {
				return err
			}
			trv1.Kind = "TaskRun"
			trv1.APIVersion = "tekton.dev/v1"
			return printTaskRun(opt.Output, opt.stream, &trv1)
		}
		return printTaskRun(opt.Output, opt.stream, trCreated)
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
		Prefixing:   true,
		Params:      opt.cliparams,
		AllSteps:    false,
	}
	return taskrun.Run(runLogOpts)
}

func printTaskRun(output string, s *cli.Stream, tr interface{}) error {
	format := strings.ToLower(output)
	if format == "" || format == "yaml" {
		trBytes, err := yaml.Marshal(tr)
		if err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "%s", trBytes)
	}

	if format == "json" {
		trBytes, err := json.MarshalIndent(tr, "", "\t")
		if err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "%s\n", trBytes)
	}

	return nil
}

func (opt *startOptions) getInputs() error {
	if opt.Last && opt.UseTaskRun != "" {
		fmt.Fprintf(opt.stream.Err, "option --last and option --use-taskrun are not compatible \n")
		return nil
	}

	intOpts := options.InteractiveOpts{
		Stream:                opt.stream,
		CliParams:             opt.cliparams,
		AskOpts:               opt.askOpts,
		Ns:                    opt.cliparams.Namespace(),
		SkipOptionalWorkspace: opt.SkipOptionalWorkspace,
	}

	params.FilterParamsByType(opt.task.Spec.Params)
	if !opt.Last && opt.UseTaskRun == "" {
		skipParams, err := params.ParseParams(opt.Params)
		if err != nil {
			return err
		}
		if err := intOpts.TaskParams(opt.task, skipParams, opt.UseParamDefaults); err != nil {
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

func getTaskV1beta1(gr schema.GroupVersionResource, c *cli.Clients, tName, ns string) (*v1beta1.Task, error) {
	var task v1beta1.Task
	gvr, err := actions.GetGroupVersionResource(gr, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if gvr.Version == "v1beta1" {
		err := actions.GetV1(taskGroupResource, c, tName, ns, metav1.GetOptions{}, &task)
		if err != nil {
			return nil, err
		}
		return &task, nil
	}

	var taskV1 v1.Task
	err = actions.GetV1(taskGroupResource, c, tName, ns, metav1.GetOptions{}, &taskV1)
	if err != nil {
		return nil, err
	}
	err = task.ConvertFrom(context.Background(), &taskV1)
	if err != nil {
		return nil, err
	}
	return &task, nil
}
