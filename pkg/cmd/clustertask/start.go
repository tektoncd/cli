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
	"log"
	"sort"
	"strings"
	"time"

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
	ct, err := ctactions.Get(c, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf(errInvalidClusterTask, name)
	}

	if ct.Spec.Params != nil {
		params.FilterParamsByType(ct.Spec.Params)
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
			if opt.TimeOut != 3600 {
				log.Println("WARNING: The --timeout flag will no longer be specified in seconds in v1.0.0. Learn more here: https://github.com/tektoncd/cli/issues/784")
				log.Println("WARNING: The -t shortand for --timeout will no longer be available in v1.0.0.")
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
	c.Flags().StringArrayVarP(&opt.Params, "param", "p", []string{}, "pass the param as key=value for string type, or key=value1,value2,... for array type")
	c.Flags().StringVarP(&opt.ServiceAccountName, "serviceaccount", "s", "", "pass the serviceaccount name")
	flags.AddShellCompletion(c.Flags().Lookup("serviceaccount"), "__kubectl_get_serviceaccount")
	c.Flags().BoolVarP(&opt.Last, "last", "L", false, "re-run the clustertask using last taskrun values")
	c.Flags().StringSliceVarP(&opt.Labels, "labels", "l", []string{}, "pass labels as label=value.")
	c.Flags().BoolVarP(&opt.ShowLog, "showlog", "", false, "show logs right after starting the clustertask")
	c.Flags().StringVar(&opt.TimeOut, "timeout", "1h", "timeout for taskrun")
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

	var ctname string
	timeout, err := time.ParseDuration(opt.TimeOut)
	if err != nil {
		return err
	}
	ctname = args[0]
	tr.Spec = v1beta1.TaskRunSpec{
		TaskRef: &v1beta1.TaskRef{
			Name: ctname,
			Kind: v1beta1.ClusterTaskKind, //Specify TaskRun is for a ClusterTask kind
		},
		Timeout: &metav1.Duration{Duration: timeout},
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
		trConverted := tractions.ConvertDown(tr)
		trConverted.APIVersion = version
		trConverted.Kind = "TaskRun"
		if err != nil {
			return nil, err
		}
		return &trConverted, nil
	}

	return tr, nil
}
