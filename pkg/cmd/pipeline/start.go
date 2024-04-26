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

package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	prcmd "github.com/tektoncd/cli/pkg/cmd/pipelinerun"
	"github.com/tektoncd/cli/pkg/file"
	"github.com/tektoncd/cli/pkg/flags"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/labels"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/params"
	pipelinepkg "github.com/tektoncd/cli/pkg/pipeline"
	"github.com/tektoncd/cli/pkg/pipelinerun"
	"github.com/tektoncd/cli/pkg/pods"
	"github.com/tektoncd/cli/pkg/workspaces"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

var (
	errNoPipeline      = errors.New("missing Pipeline name")
	errInvalidPipeline = "Pipeline name %s does not exist in namespace %s"
)

const (
	invalidSvc = "invalid service account parameter: "
)

type startOptions struct {
	cliparams             cli.Params
	stream                *cli.Stream
	askOpts               survey.AskOpt
	Params                []string
	ServiceAccountName    string
	ServiceAccounts       []string
	Last                  bool
	UsePipelineRun        string
	Labels                []string
	ShowLog               bool
	DryRun                bool
	ExitWithPrError       bool
	Output                string
	PrefixName            string
	TimeOut               string
	PipelineTimeOut       string
	TasksTimeOut          string
	FinallyTimeOut        string
	Filename              string
	Workspaces            []string
	UseParamDefaults      bool
	TektonOptions         flags.TektonOptions
	PodTemplate           string
	SkipOptionalWorkspace bool
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
		Use:   "start",
		Short: "Start Pipelines",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Example: `Start Pipeline foo by creating a PipelineRun named "foo-run-xyz123" from namespace 'bar':

    tkn pipeline start foo -s ServiceAccountName -n bar

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
`,
		SilenceUsage: true,

		ValidArgsFunction: formatted.ParentCompletion,
		Args: func(cmd *cobra.Command, _ []string) error {
			if err := flags.InitParams(p, cmd); err != nil {
				return err
			}
			if opt.Last && opt.UsePipelineRun != "" {
				return errors.New("option --last and option --use-pipelinerun can't be specify together")
			}
			if opt.Filename != "" && opt.Last {
				return errors.New("cannot use --last option with --filename option")
			}
			if opt.UseParamDefaults && (opt.Last || opt.UsePipelineRun != "") {
				return errors.New("cannot use --last or --use-pipelinerun options with --use-param-defaults option")
			}
			format := strings.ToLower(opt.Output)
			if format != "" && format != "json" && format != "yaml" && format != "name" {
				return fmt.Errorf("output format specified is %s but must be yaml or json", opt.Output)
			}
			if format != "" && opt.ShowLog {
				return errors.New("cannot use --output option with --showlog option")
			}
			opt.TektonOptions = flags.GetTektonOptions(cmd)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opt.stream = &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			pipeline, err := NameArg(args, p, opt.Filename)
			if err != nil {
				return err
			}

			return opt.run(pipeline)
		},
	}

	c.Flags().BoolVarP(&opt.ShowLog, "showlog", "", false, "show logs right after starting the Pipeline")
	c.Flags().StringArrayVarP(&opt.Params, "param", "p", []string{}, "pass the param as key=value for string type, or key=value1,value2,... for array type, or key=\"key1:value1, key2:value2\" for object type")
	c.Flags().BoolVarP(&opt.Last, "last", "L", false, "re-run the Pipeline using last PipelineRun values")
	c.Flags().StringVarP(&opt.UsePipelineRun, "use-pipelinerun", "", "", "use this pipelinerun values to re-run the pipeline. ")
	_ = c.RegisterFlagCompletionFunc("use-pipelinerun",
		func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return formatted.BaseCompletion("pipelinerun", args)
		},
	)

	c.Flags().StringSliceVarP(&opt.Labels, "labels", "l", []string{}, "pass labels as label=value.")
	c.Flags().StringArrayVarP(&opt.Workspaces, "workspace", "w", []string{}, "pass one or more workspaces to map to the corresponding physical volumes")
	c.Flags().BoolVarP(&opt.DryRun, "dry-run", "", false, "preview PipelineRun without running it")
	c.Flags().StringVarP(&opt.Output, "output", "o", "", "format of PipelineRun (yaml, json or name)")
	c.Flags().StringVarP(&opt.PrefixName, "prefix-name", "", "", "specify a prefix for the PipelineRun name (must be lowercase alphanumeric characters)")
	c.Flags().StringVarP(&opt.TimeOut, "timeout", "", "", "timeout for PipelineRun")
	_ = c.Flags().MarkDeprecated("timeout", "please use --pipeline-timeout flag instead")
	c.Flags().StringVarP(&opt.PipelineTimeOut, "pipeline-timeout", "", "", "timeout for PipelineRun")
	c.Flags().StringVarP(&opt.TasksTimeOut, "tasks-timeout", "", "", "timeout for Pipeline TaskRuns")
	c.Flags().StringVarP(&opt.FinallyTimeOut, "finally-timeout", "", "", "timeout for Finally TaskRuns")
	c.Flags().StringVarP(&opt.Filename, "filename", "f", "", "local or remote file name containing a Pipeline definition to start a PipelineRun")
	c.Flags().BoolVarP(&opt.UseParamDefaults, "use-param-defaults", "", false, "use default parameter values without prompting for input")
	c.Flags().StringVar(&opt.PodTemplate, "pod-template", "", "local or remote file containing a PodTemplate definition")
	c.Flags().BoolVarP(&opt.SkipOptionalWorkspace, "skip-optional-workspace", "", false, "skips the prompt for optional workspaces")
	c.Flags().BoolVarP(&opt.ExitWithPrError, "exit-with-pipelinerun-error", "E", false, "when using --showlog, exit with pipelinerun to the unix shell, 0 if success, 1 if error, 2 on unknown status")

	c.Flags().StringVarP(&opt.ServiceAccountName, "serviceaccount", "s", "", "pass the serviceaccount name")
	_ = c.RegisterFlagCompletionFunc("serviceaccount",
		func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return formatted.BaseCompletion("serviceaccount", args)
		},
	)

	c.Flags().StringSliceVar(&opt.ServiceAccounts, "task-serviceaccount", []string{}, "pass the service account corresponding to the task")
	_ = c.RegisterFlagCompletionFunc("task-serviceaccount",
		func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return formatted.BaseCompletion("serviceaccount", args)
		},
	)
	return c
}

func (opt *startOptions) run(pipeline *v1beta1.Pipeline) error {
	if err := opt.getInput(pipeline); err != nil {
		return err
	}

	return opt.startPipeline(pipeline)
}

func (opt *startOptions) startPipeline(pipelineStart *v1beta1.Pipeline) error {
	cs, err := opt.cliparams.Clients()
	if err != nil {
		return err
	}

	objMeta := metav1.ObjectMeta{
		Namespace: opt.cliparams.Namespace(),
	}
	var pr *v1beta1.PipelineRun
	if opt.Filename == "" {
		pr = &v1beta1.PipelineRun{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "PipelineRun",
			},
			ObjectMeta: objMeta,
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{Name: pipelineStart.ObjectMeta.Name},
			},
		}
	} else {
		pr = &v1beta1.PipelineRun{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "PipelineRun",
			},
			ObjectMeta: objMeta,
			Spec: v1beta1.PipelineRunSpec{
				PipelineSpec: &pipelineStart.Spec,
			},
		}
	}

	if opt.Last || opt.UsePipelineRun != "" {
		var usepr *v1beta1.PipelineRun
		if opt.Last {
			name, err := pipelinepkg.LastRunName(cs, pipelineStart.ObjectMeta.Name, opt.cliparams.Namespace())
			if err != nil {
				return err
			}
			usepr, err = getPipelineRunV1beta1(pipelineRunGroupResource, cs, name, opt.cliparams.Namespace())
			if err != nil {
				return err
			}
		} else {
			usepr, err = getPipelineRunV1beta1(pipelineRunGroupResource, cs, opt.UsePipelineRun, opt.cliparams.Namespace())
			if err != nil {
				return err
			}
		}

		if len(usepr.ObjectMeta.GenerateName) > 0 && opt.PrefixName == "" {
			pr.ObjectMeta.GenerateName = usepr.ObjectMeta.GenerateName
		} else if opt.PrefixName == "" {
			pr.ObjectMeta.GenerateName = usepr.ObjectMeta.Name + "-"
		}

		// Copy over spec from last or previous PipelineRun to use same values for this PipelineRun
		pr.Spec = usepr.Spec
		// Reapply blank status in case PipelineRun used was cancelled
		pr.Spec.Status = ""
	}

	if opt.PrefixName == "" && !opt.Last && opt.UsePipelineRun == "" {
		pr.ObjectMeta.GenerateName = pipelineStart.ObjectMeta.Name + "-run-"
	} else if opt.PrefixName != "" {
		pr.ObjectMeta.GenerateName = opt.PrefixName + "-"
	}

	if opt.TimeOut != "" {
		timeoutDuration, err := time.ParseDuration(opt.TimeOut)
		if err != nil {
			return err
		}
		pr.Spec.Timeouts = &v1beta1.TimeoutFields{
			Pipeline: &metav1.Duration{Duration: timeoutDuration},
		}
	}

	if opt.TasksTimeOut != "" || opt.PipelineTimeOut != "" || opt.FinallyTimeOut != "" {
		if err := opt.getTimeouts(pr); err != nil {
			return err
		}
	}

	labels, err := labels.MergeLabels(pr.ObjectMeta.Labels, opt.Labels)
	if err != nil {
		return err
	}
	pr.ObjectMeta.Labels = labels

	param, err := params.MergeParam(pr.Spec.Params, opt.Params)
	if err != nil {
		return err
	}
	pr.Spec.Params = param

	workspaces, err := workspaces.Merge(pr.Spec.Workspaces, opt.Workspaces, cs.HTTPClient)
	if err != nil {
		return err
	}
	pr.Spec.Workspaces = workspaces

	if err := mergeSvc(pr, opt.ServiceAccounts); err != nil {
		return err
	}

	if len(opt.ServiceAccountName) > 0 {
		pr.Spec.ServiceAccountName = opt.ServiceAccountName
	}

	podTemplateLocation := opt.PodTemplate
	if podTemplateLocation != "" {
		podTemplate, err := pods.ParsePodTemplate(cs.HTTPClient, podTemplateLocation, file.IsYamlFile(), fmt.Errorf("invalid file format for %s: .yaml or .yml file extension and format required", podTemplateLocation))
		if err != nil {
			return err
		}
		pr.Spec.PodTemplate = &podTemplate
	}

	if opt.DryRun {
		format := strings.ToLower(opt.Output)
		if format == "name" {
			fmt.Fprintf(opt.stream.Out, "%s\n", pr.GetName())
			return nil
		}
		gvr, err := actions.GetGroupVersionResource(pipelineRunGroupResource, cs.Tekton.Discovery())
		if err != nil {
			return err
		}
		if gvr.Version == "v1" {
			var prv1 v1.PipelineRun
			err = pr.ConvertTo(context.Background(), &prv1)
			if err != nil {
				return err
			}
			prv1.Kind = "PipelineRun"
			prv1.APIVersion = "tekton.dev/v1"
			return printPipelineRun(opt.Output, opt.stream, &prv1)
		}
		return printPipelineRun(opt.Output, opt.stream, pr)
	}

	prCreated, err := pipelinerun.Create(cs, pr, metav1.CreateOptions{}, opt.cliparams.Namespace())
	if err != nil {
		return err
	}

	if opt.Output != "" {
		format := strings.ToLower(opt.Output)
		if format == "name" {
			fmt.Fprintf(opt.stream.Out, "%s\n", prCreated.GetName())
			return nil
		}
		gvr, err := actions.GetGroupVersionResource(pipelineRunGroupResource, cs.Tekton.Discovery())
		if err != nil {
			return err
		}
		if gvr.Version == "v1" {
			var prv1 v1.PipelineRun
			err = prCreated.ConvertTo(context.Background(), &prv1)
			if err != nil {
				return err
			}
			prv1.Kind = "PipelineRun"
			prv1.APIVersion = "tekton.dev/v1"
			return printPipelineRun(opt.Output, opt.stream, &prv1)
		}
		return printPipelineRun(opt.Output, opt.stream, prCreated)
	}

	fmt.Fprintf(opt.stream.Out, "PipelineRun started: %s\n", prCreated.Name)
	if !opt.ShowLog {
		inOrderString := "\nIn order to track the PipelineRun progress run:\ntkn pipelinerun "
		if opt.TektonOptions.Context != "" {
			inOrderString += fmt.Sprintf("--context=%s ", opt.TektonOptions.Context)
		}
		inOrderString += fmt.Sprintf("logs %s -f -n %s\n", prCreated.Name, prCreated.Namespace)

		fmt.Fprint(opt.stream.Out, inOrderString)
		return nil
	}

	fmt.Fprintf(opt.stream.Out, "Waiting for logs to be available...\n")
	runLogOpts := &options.LogOptions{
		PipelineName:    pipelineStart.ObjectMeta.Name,
		PipelineRunName: prCreated.Name,
		Stream:          opt.stream,
		Follow:          true,
		Prefixing:       true,
		Params:          opt.cliparams,
		AllSteps:        false,
		ExitWithPrError: opt.ExitWithPrError,
	}
	return prcmd.Run(runLogOpts)
}

func (opt *startOptions) getInput(pipeline *v1beta1.Pipeline) error {
	params.FilterParamsByType(pipeline.Spec.Params)
	if !opt.Last && opt.UsePipelineRun == "" {
		skipParams, err := params.ParseParams(opt.Params)
		if err != nil {
			return err
		}
		if err = opt.getInputParams(pipeline, skipParams, opt.UseParamDefaults); err != nil {
			return err
		}
	}

	if len(opt.Workspaces) == 0 && !opt.Last && opt.UsePipelineRun == "" {
		if err := opt.getInputWorkspaces(pipeline); err != nil {
			return err
		}
	}

	return nil
}

func (opt *startOptions) getTimeouts(pr *v1beta1.PipelineRun) error {
	pr.Spec.Timeouts = &v1beta1.TimeoutFields{}

	if opt.PipelineTimeOut != "" {
		timeoutDuration, err := time.ParseDuration(opt.PipelineTimeOut)
		if err != nil {
			return err
		}
		pr.Spec.Timeouts.Pipeline = &metav1.Duration{Duration: timeoutDuration}
	}

	if opt.TasksTimeOut != "" {
		timeoutDuration, err := time.ParseDuration(opt.TasksTimeOut)
		if err != nil {
			return err
		}
		pr.Spec.Timeouts.Tasks = &metav1.Duration{Duration: timeoutDuration}
	}

	if opt.FinallyTimeOut != "" {
		timeoutDuration, err := time.ParseDuration(opt.FinallyTimeOut)
		if err != nil {
			return err
		}
		pr.Spec.Timeouts.Finally = &metav1.Duration{Duration: timeoutDuration}
	}
	return nil
}

func (opt *startOptions) getInputParams(pipeline *v1beta1.Pipeline, skipParams map[string]string, useParamDefaults bool) error {
	for _, param := range pipeline.Spec.Params {
		if param.Default == nil && useParamDefaults || !useParamDefaults {
			if _, toSkip := skipParams[param.Name]; toSkip {
				continue
			}
			var ans, ques, defaultValue string
			ques = fmt.Sprintf("Value for param `%s` of type `%s`?", param.Name, param.Type)
			input := &survey.Input{}
			if param.Default != nil {
				if param.Type == "string" {
					defaultValue = param.Default.StringVal
				}
				if param.Type == "array" {
					defaultValue = strings.Join(param.Default.ArrayVal, ",")
				}
				if param.Type == "object" {
					defaultValue = fmt.Sprintf("%+v", param.Default.ObjectVal)
				}
				ques += fmt.Sprintf(" (Default is `%s`)", defaultValue)
				input.Default = defaultValue
			}
			input.Message = ques

			qs := []*survey.Question{
				{
					Name:   "pipeline param",
					Prompt: input,
				},
			}

			if err := survey.Ask(qs, &ans, opt.askOpts); err != nil {
				return err
			}

			opt.Params = append(opt.Params, param.Name+"="+ans)
		}
	}
	return nil
}

func mergeSvc(pr *v1beta1.PipelineRun, optSvc []string) error {
	svcs, err := parseTaskSvc(optSvc)
	if err != nil {
		return err
	}

	if len(svcs) == 0 {
		return nil
	}

	for i := range pr.Spec.TaskRunSpecs {
		if v, ok := svcs[pr.Spec.TaskRunSpecs[i].PipelineTaskName]; ok {
			pr.Spec.TaskRunSpecs[i].TaskServiceAccountName = v.TaskServiceAccountName
			delete(svcs, v.PipelineTaskName)
		}
	}

	for _, v := range svcs {
		pr.Spec.TaskRunSpecs = append(pr.Spec.TaskRunSpecs, v)
	}

	return nil
}

func parseTaskSvc(s []string) (map[string]v1beta1.PipelineTaskRunSpec, error) {
	svcs := map[string]v1beta1.PipelineTaskRunSpec{}
	for _, v := range s {
		r := strings.Split(v, "=")
		if len(r) != 2 || len(r[0]) == 0 {
			errMsg := invalidSvc + v +
				"\nPlease pass Task service accounts as " +
				"--task-serviceaccount TaskName=ServiceAccount"
			return nil, errors.New(errMsg)
		}
		svcs[r[0]] = v1beta1.PipelineTaskRunSpec{
			PipelineTaskName:       r[0],
			TaskServiceAccountName: r[1],
		}
	}
	return svcs, nil
}

func printPipelineRun(output string, s *cli.Stream, pr interface{}) error {
	format := strings.ToLower(output)
	if format == "" || format == "yaml" {
		prBytes, err := yaml.Marshal(pr)
		if err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "%s", prBytes)
	}

	if format == "json" {
		prBytes, err := json.MarshalIndent(pr, "", "\t")
		if err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "%s\n", prBytes)
	}
	return nil
}

// NameArg validates that the first argument is a valid pipeline name
func NameArg(args []string, p cli.Params, file string) (*v1beta1.Pipeline, error) {
	pipelineErr := &v1beta1.Pipeline{}
	if len(args) == 0 && file == "" {
		return pipelineErr, errNoPipeline
	}

	c, err := p.Clients()
	if err != nil {
		return pipelineErr, err
	}

	if file == "" {
		name, ns := args[0], p.Namespace()
		// get pipeline by pipeline name passed as arg[0] from namespace
		pipelineNs, err := getPipelineV1beta1(pipelineGroupResource, c, name, ns)
		if err != nil {
			return pipelineErr, fmt.Errorf(errInvalidPipeline, name, ns)
		}
		return pipelineNs, nil
	}

	// file does not equal "" so the pipeline is parsed from local or remote file
	pipelineFile, err := parsePipeline(file, c.HTTPClient)
	if err != nil {
		return pipelineErr, err
	}

	return pipelineFile, nil
}

func parsePipeline(pipelineLocation string, httpClient http.Client) (*v1beta1.Pipeline, error) {
	b, err := file.LoadFileContent(httpClient, pipelineLocation, file.IsYamlFile(), fmt.Errorf("invalid file format for %s: .yaml or .yml file extension and format required", pipelineLocation))
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	err = yaml.UnmarshalStrict(b, &m)
	if err != nil {
		return nil, err
	}
	if m["apiVersion"] == "tekton.dev/v1alpha1" {
		return nil, fmt.Errorf("v1alpha1 is no longer supported")
	}

	pipeline := v1beta1.Pipeline{}
	if m["apiVersion"] == "tekton.dev/v1" {
		pipelineV1 := v1.Pipeline{}
		if err := yaml.UnmarshalStrict(b, &pipelineV1); err != nil {
			return nil, err
		}
		err = pipeline.ConvertFrom(context.Background(), &pipelineV1)
		if err != nil {
			return nil, err
		}
	} else {
		if err := yaml.UnmarshalStrict(b, &pipeline); err != nil {
			return nil, err
		}
	}

	err = params.ValidateParamType(pipeline.Spec.Params)
	if err != nil {
		return nil, err
	}

	return &pipeline, nil
}

func (opt *startOptions) getInputWorkspaces(pipeline *v1beta1.Pipeline) error {
	for _, ws := range pipeline.Spec.Workspaces {
		if ws.Optional && opt.SkipOptionalWorkspace {
			continue
		}
		if ws.Optional {
			isOptional, err := askParam(fmt.Sprintf("Do you want to give specifications for the optional workspace `%s`: (y/N)", ws.Name), opt.askOpts)
			if err != nil {
				return err
			}
			if strings.ToLower(isOptional) == "n" {
				continue
			}
		}
		fmt.Fprintf(opt.stream.Out, "Please give specifications for the workspace: %s \n", ws.Name)
		name, err := askParam("Name for the workspace :", opt.askOpts)
		if err != nil {
			return err
		}
		workspace := "name=" + name
		subPath, err := askParam("Value of the Sub Path :", opt.askOpts, " ")
		if err != nil {
			return err
		}
		if subPath != " " {
			workspace = workspace + ",subPath=" + subPath
		}

		var kind string
		qs := []*survey.Question{
			{
				Name: "workspace param",
				Prompt: &survey.Select{
					Message: "Type of the Workspace :",
					Options: []string{"config", "emptyDir", "secret", "pvc"},
					Default: "emptyDir",
				},
			},
		}
		if err := survey.Ask(qs, &kind, opt.askOpts); err != nil {
			return err
		}
		switch kind {
		case "pvc":
			claimName, err := askParam("Value of Claim Name :", opt.askOpts)
			if err != nil {
				return err
			}
			workspace = workspace + ",claimName=" + claimName
		case "emptyDir":
			kind, err := askParam("Type of EmptyDir :", opt.askOpts, "")
			if err != nil {
				return err
			}
			workspace = workspace + ",emptyDir=" + kind
		case "config":
			config, err := askParam("Name of the configmap :", opt.askOpts)
			if err != nil {
				return err
			}
			workspace = workspace + ",config=" + config
			items, err := getItems(opt.askOpts)
			if err != nil {
				return err
			}
			workspace += items
		case "secret":
			secret, err := askParam("Name of the secret :", opt.askOpts)
			if err != nil {
				return err
			}
			workspace = workspace + ",secret=" + secret
			items, err := getItems(opt.askOpts)
			if err != nil {
				return err
			}
			workspace += items
		}
		opt.Workspaces = append(opt.Workspaces, workspace)

	}
	return nil
}

func getItems(askOpts survey.AskOpt) (string, error) {
	var items string
	for {
		it, err := askParam("Item Value :", askOpts, " ")
		if err != nil {
			return "", err
		}
		if it != " " {
			items = items + ",item=" + it
		} else {
			return items, nil
		}
	}
}

func askParam(ques string, askOpts survey.AskOpt, def ...string) (string, error) {
	var ans string
	input := &survey.Input{
		Message: ques,
	}
	if len(def) != 0 {
		input.Default = def[0]
	}

	qs := []*survey.Question{
		{
			Name:   "workspace param",
			Prompt: input,
		},
	}
	if err := survey.Ask(qs, &ans, askOpts); err != nil {
		return "", err
	}

	return ans, nil
}

func getPipelineV1beta1(gr schema.GroupVersionResource, c *cli.Clients, pName, ns string) (*v1beta1.Pipeline, error) {
	var pipeline v1beta1.Pipeline
	gvr, err := actions.GetGroupVersionResource(gr, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if gvr.Version == "v1beta1" {
		err := actions.GetV1(gr, c, pName, ns, metav1.GetOptions{}, &pipeline)
		if err != nil {
			return nil, err
		}
		return &pipeline, nil
	}

	var pipelineV1 v1.Pipeline
	err = actions.GetV1(gr, c, pName, ns, metav1.GetOptions{}, &pipelineV1)
	if err != nil {
		return nil, err
	}
	err = pipeline.ConvertFrom(context.Background(), &pipelineV1)
	if err != nil {
		return nil, err
	}
	return &pipeline, nil
}

func getPipelineRunV1beta1(gr schema.GroupVersionResource, c *cli.Clients, prName, ns string) (*v1beta1.PipelineRun, error) {
	var pipelinerun v1beta1.PipelineRun
	gvr, err := actions.GetGroupVersionResource(gr, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if gvr.Version == "v1beta1" {
		err := actions.GetV1(gr, c, prName, ns, metav1.GetOptions{}, &pipelinerun)
		if err != nil {
			return nil, err
		}
		return &pipelinerun, nil
	}

	var pipelinerunV1 v1.PipelineRun
	err = actions.GetV1(gr, c, prName, ns, metav1.GetOptions{}, &pipelinerunV1)
	if err != nil {
		return nil, err
	}
	err = pipelinerun.ConvertFrom(context.Background(), &pipelinerunV1)
	if err != nil {
		return nil, err
	}
	return &pipelinerun, nil
}
