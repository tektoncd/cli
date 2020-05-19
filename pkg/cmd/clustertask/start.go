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
	"sort"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	ctactions "github.com/tektoncd/cli/pkg/clustertask"
	"github.com/tektoncd/cli/pkg/cmd/taskrun"
	"github.com/tektoncd/cli/pkg/flags"
	"github.com/tektoncd/cli/pkg/labels"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/params"
	"github.com/tektoncd/cli/pkg/task"
	tractions "github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/cli/pkg/workspaces"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

var (
	errNoClusterTask      = errors.New("missing clustertask name")
	errInvalidClusterTask = "clustertask name %s does not exist"
)

const invalidResource = "invalid input format for resource parameter: "

type startOptions struct {
	cliparams          cli.Params
	stream             *cli.Stream
	askOpts            survey.AskOpt
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
}

// NameArg validates that the first argument is a valid clustertask name
func NameArg(args []string, p cli.Params) (*v1beta1.ClusterTask, error) {
	clusterTaskErr := &v1beta1.ClusterTask{}
	if len(args) == 0 {
		return clusterTaskErr, errNoClusterTask
	}

	c, err := p.Clients()
	if err != nil {
		return clusterTaskErr, err
	}

	name := args[0]
	ct, err := ctactions.Get(c, name, metav1.GetOptions{})
	if err != nil {
		return clusterTaskErr, fmt.Errorf(errInvalidClusterTask, name)
	}

	if ct.Spec.Params != nil {
		params.FilterParamsByType(ct.Spec.Params)
	}

	return ct, nil
}

func startCommand(p cli.Params) *cobra.Command {
	opt := startOptions{
		cliparams: p,
	}

	eg := `Start ClusterTask foo by creating a TaskRun named "foo-run-xyz123" in namespace 'bar':

    tkn clustertask start foo -n bar

or

    tkn ct start foo -n bar

For params value, if you want to provide multiple values, provide them comma separated
like cat,foo,bar
`

	c := &cobra.Command{
		Use:   "start clustertask [RESOURCES...] [PARAMS...] [SERVICEACCOUNT]",
		Short: "Start clustertasks",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Example:      eg,
		SilenceUsage: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if opt.DryRun {
				format := strings.ToLower(opt.Output)
				if format != "" && format != "json" && format != "yaml" {
					return fmt.Errorf("output format specified is %s but must be yaml or json", opt.Output)
				}
			}
			if err := flags.InitParams(p, cmd); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opt.stream = &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			clusterTask, err := NameArg(args, p)
			if err != nil {
				return err
			}
			if len(opt.Workspaces) == 0 && !opt.Last {
				err := opt.getInputWorkspaces(clusterTask)
				if err != nil {
					return err
				}
			}

			return startClusterTask(opt, args)
		},
	}

	c.Flags().StringSliceVarP(&opt.InputResources, "inputresource", "i", []string{}, "pass the input resource name and ref as name=ref")
	c.Flags().StringSliceVarP(&opt.OutputResources, "outputresource", "o", []string{}, "pass the output resource name and ref as name=ref")
	c.Flags().StringArrayVarP(&opt.Params, "param", "p", []string{}, "pass the param as key=value for string type, or key=value1,value2,... for array type")
	c.Flags().StringVarP(&opt.ServiceAccountName, "serviceaccount", "s", "", "pass the serviceaccount name")
	flags.AddShellCompletion(c.Flags().Lookup("serviceaccount"), "__kubectl_get_serviceaccount")
	c.Flags().BoolVarP(&opt.Last, "last", "L", false, "re-run the clustertask using last taskrun values")
	c.Flags().StringSliceVarP(&opt.Labels, "labels", "l", []string{}, "pass labels as label=value.")
	c.Flags().StringArrayVarP(&opt.Workspaces, "workspace", "w", []string{}, "pass the workspace.")
	c.Flags().BoolVarP(&opt.ShowLog, "showlog", "", false, "show logs right after starting the clustertask")
	c.Flags().StringVar(&opt.TimeOut, "timeout", "", "timeout for taskrun")
	c.Flags().BoolVarP(&opt.DryRun, "dry-run", "", false, "preview taskrun without running it")
	c.Flags().StringVarP(&opt.Output, "output", "", "", "format of taskrun dry-run (yaml or json)")

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

	trCreated, err := tractions.Create(cs, tr, metav1.CreateOptions{}, opt.cliparams.Namespace())
	if err != nil {
		return err
	}

	fmt.Fprintf(opt.stream.Out, "Taskrun started: %s\n", trCreated.Name)
	if !opt.ShowLog {
		fmt.Fprintf(opt.stream.Out, "\nIn order to track the taskrun progress run:\ntkn taskrun logs %s -f -n %s\n", trCreated.Name, trCreated.Namespace)
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

	var qs = []*survey.Question{
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

func (opt *startOptions) getInputWorkspaces(clustertask *v1beta1.ClusterTask) error {
	for _, ws := range clustertask.Spec.Workspaces {
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
		var qs = []*survey.Question{
			{
				Name: "workspace param",
				Prompt: &survey.Select{
					Message: " Type of the Workspace :",
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
			kind, err := askParam("Type of EmtpyDir :", opt.askOpts, "")
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
