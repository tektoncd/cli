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
	"sort"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cmd/pipelineresource"
	prcmd "github.com/tektoncd/cli/pkg/cmd/pipelinerun"
	"github.com/tektoncd/cli/pkg/file"
	"github.com/tektoncd/cli/pkg/flags"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/labels"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/params"
	"github.com/tektoncd/cli/pkg/pipeline"
	"github.com/tektoncd/cli/pkg/pipelinerun"
	"github.com/tektoncd/cli/pkg/pods"
	"github.com/tektoncd/cli/pkg/workspaces"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	versionedResource "github.com/tektoncd/pipeline/pkg/client/resource/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/yaml"
)

var (
	errNoPipeline      = errors.New("missing Pipeline name")
	errInvalidPipeline = "Pipeline name %s does not exist in namespace %s"
)

const (
	invalidResource = "invalid input format for resource parameter: "
	invalidSvc      = "invalid service account parameter: "
)

type startOptions struct {
	cliparams             cli.Params
	stream                *cli.Stream
	askOpts               survey.AskOpt
	Params                []string
	Resources             []string
	ServiceAccountName    string
	ServiceAccounts       []string
	Last                  bool
	UsePipelineRun        string
	Labels                []string
	ShowLog               bool
	DryRun                bool
	Output                string
	PrefixName            string
	TimeOut               string
	Filename              string
	Workspaces            []string
	UseParamDefaults      bool
	TektonOptions         flags.TektonOptions
	PodTemplate           string
	SkipOptionalWorkspace bool
}

type resourceOptionsFilter struct {
	git         []string
	image       []string
	cluster     []string
	storage     []string
	pullRequest []string
	cloudEvent  []string
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
`,
		SilenceUsage: true,

		ValidArgsFunction: formatted.ParentCompletion,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := flags.InitParams(p, cmd); err != nil {
				return err
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
	c.Flags().StringSliceVarP(&opt.Resources, "resource", "r", []string{}, "pass the resource name and ref as name=ref")
	c.Flags().StringArrayVarP(&opt.Params, "param", "p", []string{}, "pass the param as key=value for string type, or key=value1,value2,... for array type")
	c.Flags().BoolVarP(&opt.Last, "last", "L", false, "re-run the Pipeline using last PipelineRun values")
	c.Flags().StringVarP(&opt.UsePipelineRun, "use-pipelinerun", "", "", "use this pipelinerun values to re-run the pipeline. ")
	_ = c.RegisterFlagCompletionFunc("use-pipelinerun",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return formatted.BaseCompletion("pipelinerun", args)
		},
	)

	c.Flags().StringSliceVarP(&opt.Labels, "labels", "l", []string{}, "pass labels as label=value.")
	c.Flags().StringArrayVarP(&opt.Workspaces, "workspace", "w", []string{}, "pass one or more workspaces to map to the corresponding physical volumes")
	c.Flags().BoolVarP(&opt.DryRun, "dry-run", "", false, "preview PipelineRun without running it")
	c.Flags().StringVarP(&opt.Output, "output", "o", "", "format of PipelineRun (yaml, json or name)")
	c.Flags().StringVarP(&opt.PrefixName, "prefix-name", "", "", "specify a prefix for the PipelineRun name (must be lowercase alphanumeric characters)")
	c.Flags().StringVarP(&opt.TimeOut, "timeout", "", "", "timeout for PipelineRun")
	c.Flags().StringVarP(&opt.Filename, "filename", "f", "", "local or remote file name containing a Pipeline definition to start a PipelineRun")
	c.Flags().BoolVarP(&opt.UseParamDefaults, "use-param-defaults", "", false, "use default parameter values without prompting for input")
	c.Flags().StringVar(&opt.PodTemplate, "pod-template", "", "local or remote file containing a PodTemplate definition")
	c.Flags().BoolVarP(&opt.SkipOptionalWorkspace, "skip-optional-workspace", "", false, "skips the prompt for optional workspaces")

	c.Flags().StringVarP(&opt.ServiceAccountName, "serviceaccount", "s", "", "pass the serviceaccount name")
	_ = c.RegisterFlagCompletionFunc("serviceaccount",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return formatted.BaseCompletion("serviceaccount", args)
		},
	)

	c.Flags().StringSliceVar(&opt.ServiceAccounts, "task-serviceaccount", []string{}, "pass the service account corresponding to the task")
	_ = c.RegisterFlagCompletionFunc("task-serviceaccount",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
			usepr, err = pipeline.LastRun(cs, pipelineStart.ObjectMeta.Name, opt.cliparams.Namespace())
			if err != nil {
				return err
			}
		} else {
			usepr, err = pipelinerun.Get(cs, opt.UsePipelineRun, v1.GetOptions{}, opt.cliparams.Namespace())
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
		pr.Spec.Timeout = &metav1.Duration{Duration: timeoutDuration}
	}

	if err := mergeRes(pr, opt.Resources); err != nil {
		return err
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
		return printPipelineRun(cs, opt.Output, opt.stream, pr)
	}

	prCreated, err := pipelinerun.Create(cs, pr, metav1.CreateOptions{}, opt.cliparams.Namespace())
	if err != nil {
		return err
	}

	if opt.Output != "" {
		return printPipelineRun(cs, opt.Output, opt.stream, prCreated)
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
	}
	return prcmd.Run(runLogOpts)
}

func (opt *startOptions) getInput(pipeline *v1beta1.Pipeline) error {
	cs, err := opt.cliparams.Clients()
	if err != nil {
		return err
	}

	if opt.Last && opt.UsePipelineRun != "" {
		fmt.Fprintf(opt.stream.Err, "option --last and option --use-pipelinerun are not compatible \n")
		return err
	}

	if len(opt.Resources) == 0 && !opt.Last && opt.UsePipelineRun == "" {
		pres, err := getPipelineResources(cs.Resource, opt.cliparams.Namespace())
		if err != nil {
			return fmt.Errorf("failed to list PipelineResources from namespace %s: %v", opt.cliparams.Namespace(), err)
		}

		resources := getPipelineResourcesByFormat(pres.Items)

		if err = opt.getInputResources(resources, pipeline); err != nil {
			return err
		}
	}

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
		if err = opt.getInputWorkspaces(pipeline); err != nil {
			return err
		}
	}

	return nil
}

func (opt *startOptions) getInputResources(resources resourceOptionsFilter, pipeline *v1beta1.Pipeline) error {
	for _, res := range pipeline.Spec.Resources {
		options := getOptionsByType(resources, string(res.Type))
		// directly create resource
		if len(options) == 0 {
			ns := opt.cliparams.Namespace()
			fmt.Fprintf(opt.stream.Out, "no PipelineResource of type \"%s\" found in namespace: %s\n", string(res.Type), ns)
			fmt.Fprintf(opt.stream.Out, "Please create a new \"%s\" resource for PipelineResource \"%s\"\n", string(res.Type), res.Name)
			newres, err := opt.createPipelineResource(res.Name, res.Type)
			if err != nil {
				return err
			}
			if newres.Status != nil {
				fmt.Printf("resource status %s\n\n", newres.Status)
			}
			opt.Resources = append(opt.Resources, res.Name+"="+newres.Name)
			continue
		}

		// shows create option in the resource list
		resCreateOpt := fmt.Sprintf("create new \"%s\" resource", res.Type)
		options = append(options, resCreateOpt)
		var ans string
		qs := []*survey.Question{
			{
				Name: "pipelineresource",
				Prompt: &survey.Select{
					Message: fmt.Sprintf("Choose the %s resource to use for %s:", res.Type, res.Name),
					Options: options,
				},
			},
		}

		if err := survey.Ask(qs, &ans, opt.askOpts); err != nil {
			return err
		}

		if ans == resCreateOpt {
			newres, err := opt.createPipelineResource(res.Name, res.Type)
			if err != nil {
				return err
			}
			opt.Resources = append(opt.Resources, res.Name+"="+newres.Name)
			continue
		}
		name := strings.TrimSpace(strings.Split(ans, " ")[0])
		opt.Resources = append(opt.Resources, res.Name+"="+name)
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
				} else {
					defaultValue = strings.Join(param.Default.ArrayVal, ",")
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

func getPipelineResources(client versionedResource.Interface, namespace string) (*v1alpha1.PipelineResourceList, error) {
	pres, err := client.TektonV1alpha1().PipelineResources(namespace).List(context.Background(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pres, nil
}

func getPipelineResourcesByFormat(resources []v1alpha1.PipelineResource) (ret resourceOptionsFilter) {
	for _, res := range resources {
		output := ""
		switch string(res.Spec.Type) {
		case "git":
			for _, param := range res.Spec.Params {
				if param.Name == "url" {
					output = param.Value + output
				}
				if param.Name == "revision" && param.Value != "master" {
					output = output + "#" + param.Value
				}
			}
			ret.git = append(ret.git, fmt.Sprintf("%s (%s)", res.Name, output))
		case "image":
			for _, param := range res.Spec.Params {
				if param.Name == "url" {
					output = param.Value + output
				}
			}
			ret.image = append(ret.image, fmt.Sprintf("%s (%s)", res.Name, output))
		case "pullRequest":
			for _, param := range res.Spec.Params {
				if param.Name == "url" {
					output = param.Value + output
				}
			}
			ret.pullRequest = append(ret.pullRequest, fmt.Sprintf("%s (%s)", res.Name, output))
		case "storage":
			for _, param := range res.Spec.Params {
				if param.Name == "location" {
					output = param.Value + output
				}
			}
			ret.storage = append(ret.storage, fmt.Sprintf("%s (%s)", res.Name, output))
		case "cluster":
			for _, param := range res.Spec.Params {
				if param.Name == "url" {
					output = param.Value + output
				}
				if param.Name == "user" {
					output = output + "#" + param.Value
				}
			}
			ret.cluster = append(ret.cluster, fmt.Sprintf("%s (%s)", res.Name, output))
		case "cloudEvent":
			for _, param := range res.Spec.Params {
				if param.Name == "targetURI" {
					output = param.Value + output
				}
			}
			ret.cloudEvent = append(ret.cloudEvent, fmt.Sprintf("%s (%s)", res.Name, output))
		}
	}
	return
}

func getOptionsByType(resources resourceOptionsFilter, restype string) []string {
	if restype == "git" {
		return resources.git
	}
	if restype == "image" {
		return resources.image
	}
	if restype == "pullRequest" {
		return resources.pullRequest
	}
	if restype == "cluster" {
		return resources.cluster
	}
	if restype == "storage" {
		return resources.storage
	}
	if restype == "cloudEvent" {
		return resources.cloudEvent
	}
	return []string{}
}

func mergeRes(pr *v1beta1.PipelineRun, optRes []string) error {
	res, err := parseRes(optRes)
	if err != nil {
		return err
	}

	if len(res) == 0 {
		return nil
	}

	for i := range pr.Spec.Resources {
		if v, ok := res[pr.Spec.Resources[i].Name]; ok {
			pr.Spec.Resources[i] = v
			delete(res, v.Name)
		}
	}
	for _, v := range res {
		pr.Spec.Resources = append(pr.Spec.Resources, v)
	}
	sort.Slice(pr.Spec.Resources, func(i, j int) bool { return pr.Spec.Resources[i].Name < pr.Spec.Resources[j].Name })
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

func parseRes(res []string) (map[string]v1beta1.PipelineResourceBinding, error) {
	resources := map[string]v1beta1.PipelineResourceBinding{}
	for _, v := range res {
		r := strings.SplitN(v, "=", 2)
		if len(r) != 2 {
			return nil, errors.New(invalidResource + v)
		}
		resources[r[0]] = v1beta1.PipelineResourceBinding{
			Name: r[0],
			ResourceRef: &v1beta1.PipelineResourceRef{
				Name: r[1],
			},
		}
	}
	return resources, nil
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

func (opt *startOptions) createPipelineResource(resName string, resType v1alpha1.PipelineResourceType) (*v1alpha1.PipelineResource, error) {
	res := pipelineresource.Resource{
		AskOpts: opt.askOpts,
		Params:  opt.cliparams,
		PipelineResource: v1alpha1.PipelineResource{
			ObjectMeta: v1.ObjectMeta{Namespace: opt.cliparams.Namespace()},
			Spec:       v1alpha1.PipelineResourceSpec{Type: resType},
		},
	}

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
	cs, err := opt.cliparams.Clients()
	if err != nil {
		return nil, err
	}
	newRes, err := cs.Resource.TektonV1alpha1().PipelineResources(opt.cliparams.Namespace()).Create(context.Background(), &res.PipelineResource, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(opt.stream.Out, "New %s resource \"%s\" has been created\n", newRes.Spec.Type, newRes.Name)
	return newRes, nil
}

func printPipelineRun(c *cli.Clients, output string, s *cli.Stream, pr *v1beta1.PipelineRun) error {
	prWithVersion, err := convertedPrVersion(c, pr)
	if err != nil {
		return err
	}
	format := strings.ToLower(output)
	if format == "" || format == "yaml" {
		prBytes, err := yaml.Marshal(prWithVersion)
		if err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "%s", prBytes)
	}

	if format == "name" {
		fmt.Fprintf(s.Out, "%s\n", pr.GetName())
	}

	if format == "json" {
		prBytes, err := json.MarshalIndent(prWithVersion, "", "\t")
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
		pipelineNs, err := pipeline.Get(c, name, metav1.GetOptions{}, ns)
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
		pipeline := v1alpha1.Pipeline{}
		if err := yaml.UnmarshalStrict(b, &pipeline); err != nil {
			return nil, err
		}
		var pipelineConverted v1beta1.Pipeline
		err = pipeline.ConvertTo(context.Background(), &pipelineConverted)
		if err != nil {
			return nil, err
		}
		pipelineConverted.TypeMeta.APIVersion = "tekton.dev/v1alpha1"
		pipelineConverted.TypeMeta.APIVersion = "Pipeline"
		return &pipelineConverted, nil
	}

	pipeline := v1beta1.Pipeline{}
	if err := yaml.UnmarshalStrict(b, &pipeline); err != nil {
		return nil, err
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

func convertedPrVersion(c *cli.Clients, pr *v1beta1.PipelineRun) (interface{}, error) {
	version, err := getAPIVersion(c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if version == "tekton.dev/v1alpha1" {
		var prConverted v1alpha1.PipelineRun
		err = prConverted.ConvertFrom(context.Background(), pr)
		prConverted.APIVersion = version
		prConverted.Kind = "PipelineRun"
		if err != nil {
			return nil, err
		}
		return &prConverted, nil
	}

	return pr, nil
}
