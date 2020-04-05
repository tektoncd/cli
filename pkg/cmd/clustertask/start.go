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

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cmd/taskrun"
	pipelineconfig "github.com/tektoncd/cli/pkg/config"
	"github.com/tektoncd/cli/pkg/flags"
	"github.com/tektoncd/cli/pkg/labels"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/params"
	"github.com/tektoncd/cli/pkg/task"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	errNoClusterTask      = errors.New("missing clustertask name")
	errInvalidClusterTask = "clustertask name %s does not exist"
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
}

// NameArg validates that the first argument is a valid clustertask name
func NameArg(args []string, p cli.Params) error {
	if len(args) == 0 {
		return errNoClusterTask
	}

	c, err := p.Clients()
	if err != nil {
		return err
	}

	name := args[0]
	ct, err := c.Tekton.TektonV1alpha1().ClusterTasks().Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf(errInvalidClusterTask, name)
	}

	if ct.Spec.Inputs != nil {
		params.FilterParamsByType(ct.Spec.Inputs.Params)
	}

	return nil
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
			if err := flags.InitParams(p, cmd); err != nil {
				return err
			}
			return NameArg(args, p)

		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opt.stream = &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			return startClusterTask(opt, args)
		},
	}

	c.Flags().StringSliceVarP(&opt.InputResources, "inputresource", "i", []string{}, "pass the input resource name and ref as name=ref")
	c.Flags().StringSliceVarP(&opt.OutputResources, "outputresource", "o", []string{}, "pass the output resource name and ref as name=ref")
	c.Flags().StringArrayVarP(&opt.Params, "param", "p", []string{}, "pass the param as key=value or key=value1,value2")
	c.Flags().StringVarP(&opt.ServiceAccountName, "serviceaccount", "s", "", "pass the serviceaccount name")
	flags.AddShellCompletion(c.Flags().Lookup("serviceaccount"), "__kubectl_get_serviceaccount")
	c.Flags().BoolVarP(&opt.Last, "last", "L", false, "re-run the clustertask using last taskrun values")
	c.Flags().StringSliceVarP(&opt.Labels, "labels", "l", []string{}, "pass labels as label=value.")
	c.Flags().BoolVarP(&opt.ShowLog, "showlog", "", false, "show logs right after starting the clustertask")
	c.Flags().StringVarP(&opt.TimeOut, "timeout", "t", "1h", "timeout for taskrun")
	c.Flags().BoolVarP(&opt.DryRun, "dry-run", "", false, "preview taskrun without running it")
	c.Flags().StringVarP(&opt.Output, "output", "", "", "format of taskrun dry-run (yaml or json)")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_clustertask")

	return c
}

func startClusterTask(opt startOptions, args []string) error {
	tr := &v1alpha1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1alpha1",
			Kind:       "TaskRun",
		},
		ObjectMeta: metav1.ObjectMeta{},
	}

	var ctname string

	cs, err := opt.cliparams.Clients()
	if err != nil {
		return err
	}

	timeoutDefault := "1h"
	var timeoutDuration time.Duration
	// Check the default timeout given by tkn, if it exists,
	// then check for the default timeout in config-defaults
	// in another case, opt.TimeOut will be provided by the
	// user and will take precedence over any default
	if opt.TimeOut == timeoutDefault {
		opt.TimeOut, _ = pipelineconfig.GetDefaultTimeout(cs)
	}
	timeoutDuration, err = time.ParseDuration(opt.TimeOut)
	if err != nil {
		return err
	}

	ctname = args[0]
	tr.Spec = v1alpha1.TaskRunSpec{
		TaskRef: &v1alpha1.TaskRef{
			Name: ctname,
			Kind: v1alpha1.ClusterTaskKind, //Specify TaskRun is for a ClusterTask kind
		},
		Timeout: &metav1.Duration{Duration: timeoutDuration},
	}
	tr.ObjectMeta.GenerateName = ctname + "-run-"

	//TaskRuns are namespaced so using same LastRun method as Task
	if opt.Last {
		trLast, err := task.LastRun(cs.Tekton, ctname, opt.cliparams.Namespace(), "clustertask")
		if err != nil {
			return err
		}
		tr.Spec.Inputs = trLast.Spec.Inputs
		tr.Spec.Outputs = trLast.Spec.Outputs
		tr.Spec.ServiceAccountName = trLast.Spec.ServiceAccountName
		tr.Spec.Workspaces = trLast.Spec.Workspaces
	}

	if tr.Spec.Inputs == nil {
		tr.Spec.Inputs = &v1alpha1.TaskRunInputs{}
	}
	inputRes, err := mergeRes(tr.Spec.Inputs.Resources, opt.InputResources)
	if err != nil {
		return err
	}
	tr.Spec.Inputs.Resources = inputRes

	if tr.Spec.Outputs == nil {
		tr.Spec.Outputs = &v1alpha1.TaskRunOutputs{}
	}
	outRes, err := mergeRes(tr.Spec.Outputs.Resources, opt.OutputResources)
	if err != nil {
		return err
	}
	tr.Spec.Outputs.Resources = outRes

	labels, err := labels.MergeLabels(tr.ObjectMeta.Labels, opt.Labels)
	if err != nil {
		return err
	}
	tr.ObjectMeta.Labels = labels

	param, err := params.MergeParam(tr.Spec.Inputs.Params, opt.Params)
	if err != nil {
		return err
	}
	tr.Spec.Inputs.Params = param

	if len(opt.ServiceAccountName) > 0 {
		tr.Spec.ServiceAccountName = opt.ServiceAccountName
	}

	if opt.DryRun {
		return taskRunDryRun(opt.Output, opt.stream, tr)
	}

	trCreated, err := cs.Tekton.TektonV1alpha1().TaskRuns(opt.cliparams.Namespace()).Create(tr)
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

func mergeRes(r []v1alpha1.TaskResourceBinding, optRes []string) ([]v1alpha1.TaskResourceBinding, error) {
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

func parseRes(res []string) (map[string]v1alpha1.TaskResourceBinding, error) {
	resources := map[string]v1alpha1.TaskResourceBinding{}
	for _, v := range res {
		r := strings.SplitN(v, "=", 2)
		if len(r) != 2 {
			return nil, errors.New(invalidResource + v)
		}
		resources[r[0]] = v1alpha1.TaskResourceBinding{
			PipelineResourceBinding: v1alpha1.PipelineResourceBinding{
				Name: r[0],
				ResourceRef: &v1alpha1.PipelineResourceRef{
					Name: r[1],
				},
			},
		}
	}
	return resources, nil
}

func taskRunDryRun(output string, s *cli.Stream, tr *v1alpha1.TaskRun) error {
	format := strings.ToLower(output)

	if format != "" && format != "json" && format != "yaml" {
		return fmt.Errorf("output format specified is %s but must be yaml or json", output)
	}

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
