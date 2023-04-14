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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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
	tractions "github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/cli/pkg/workspaces"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var (
	errNoClusterTask             = errors.New("missing ClusterTask name")
	errInvalidClusterTask        = "ClusterTask name %s does not exist"
	errClusterTaskAlreadyPresent = "ClusterTask with name %s already exists"
)

type startOptions struct {
	cliparams             cli.Params
	stream                *cli.Stream
	Params                []string
	ServiceAccountName    string
	Last                  bool
	Labels                []string
	ShowLog               bool
	TimeOut               string
	DryRun                bool
	Output                string
	PrefixName            string
	Workspaces            []string
	UseParamDefaults      bool
	clustertask           *v1beta1.ClusterTask
	askOpts               survey.AskOpt
	TektonOptions         flags.TektonOptions
	PodTemplate           string
	UseTaskRun            string
	SkipOptionalWorkspace bool
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
	var ct *v1beta1.ClusterTask
	err = actions.GetV1(clustertaskGroupResource, c, name, "", metav1.GetOptions{}, &ct)
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
`

	c := &cobra.Command{
		Use:   "start",
		Short: "Start ClusterTasks",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Example:           eg,
		SilenceUsage:      true,
		ValidArgsFunction: formatted.ParentCompletion,
		Args: func(cmd *cobra.Command, args []string) error {
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
			if opt.Last && opt.UseTaskRun != "" {
				return fmt.Errorf("using --last and --use-taskrun are not compatible")
			}
			opt.TektonOptions = flags.GetTektonOptions(cmd)
			return startClusterTask(opt, args)
		},
	}
	c.Flags().StringArrayVarP(&opt.Params, "param", "p", []string{}, "pass the param as key=value for string type, or key=value1,value2,... for array type, or key=\"key1:value1, key2:value2\" for object type")
	c.Flags().StringVarP(&opt.ServiceAccountName, "serviceaccount", "s", "", "pass the serviceaccount name")
	c.Flags().BoolVarP(&opt.Last, "last", "L", false, "re-run the ClusterTask using last TaskRun values")
	c.Flags().StringVarP(&opt.UseTaskRun, "use-taskrun", "", "", "specify a TaskRun name to use its values to re-run the TaskRun")
	c.Flags().StringSliceVarP(&opt.Labels, "labels", "l", []string{}, "pass labels as label=value.")
	c.Flags().StringArrayVarP(&opt.Workspaces, "workspace", "w", []string{}, "pass one or more workspaces to map to the corresponding physical volumes")
	c.Flags().BoolVarP(&opt.ShowLog, "showlog", "", false, "show logs right after starting the ClusterTask")
	c.Flags().StringVar(&opt.TimeOut, "timeout", "", "timeout for TaskRun")
	c.Flags().BoolVarP(&opt.DryRun, "dry-run", "", false, "preview TaskRun without running it")
	c.Flags().StringVarP(&opt.Output, "output", "", "", "format of TaskRun (yaml or json)")
	c.Flags().StringVarP(&opt.PrefixName, "prefix-name", "", "", "specify a prefix for the TaskRun name (must be lowercase alphanumeric characters)")
	c.Flags().StringVar(&opt.PodTemplate, "pod-template", "", "local or remote file containing a PodTemplate definition")
	c.Flags().BoolVar(&opt.UseParamDefaults, "use-param-defaults", false, "use default parameter values without prompting for input")
	c.Flags().BoolVarP(&opt.SkipOptionalWorkspace, "skip-optional-workspace", "", false, "skips the prompt for optional workspaces")
	c.Deprecated = "ClusterTasks are deprecated, this command will be removed in future releases."
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
			Kind: v1beta1.ClusterTaskKind, // Specify TaskRun is for a ClusterTask kind
		},
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
		err := taskRunOpts.UseTaskRunFrom(tr, cs, ctname, "ClusterTask")
		if err != nil {
			return err
		}
	}

	if opt.PrefixName == "" && !opt.Last && opt.UseTaskRun == "" {
		tr.ObjectMeta.GenerateName = ctname + "-run-"
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

	trCreated, err := tractions.Create(cs, tr, metav1.CreateOptions{}, opt.cliparams.Namespace())
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
	intOpts := options.InteractiveOpts{
		Stream:                opt.stream,
		CliParams:             opt.cliparams,
		AskOpts:               opt.askOpts,
		Ns:                    opt.cliparams.Namespace(),
		SkipOptionalWorkspace: opt.SkipOptionalWorkspace,
	}

	params.FilterParamsByType(opt.clustertask.Spec.Params)
	if !opt.Last && opt.UseTaskRun == "" {
		skipParams, err := params.ParseParams(opt.Params)
		if err != nil {
			return err
		}
		if err := intOpts.ClusterTaskParams(opt.clustertask, skipParams, opt.UseParamDefaults); err != nil {
			return err
		}
		opt.Params = append(opt.Params, intOpts.Params...)
	}

	if len(opt.Workspaces) == 0 && !opt.Last && opt.UseTaskRun == "" {
		if err := intOpts.ClusterTaskWorkspaces(opt.clustertask); err != nil {
			return err
		}
		opt.Workspaces = append(opt.Workspaces, intOpts.Workspaces...)
	}

	return nil
}
