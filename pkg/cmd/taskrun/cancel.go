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

package taskrun

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func cancelCommand(p cli.Params) *cobra.Command {
	eg := `Cancel the TaskRun named 'foo' from namespace 'bar':

    tkn taskrun cancel foo -n bar
`

	c := &cobra.Command{
		Use:               "cancel",
		Short:             "Cancel a TaskRun in a namespace",
		Example:           eg,
		ValidArgsFunction: formatted.ParentCompletion,
		Args:              cobra.ExactArgs(1),
		SilenceUsage:      true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			return cancelTaskRun(p, s, args[0])
		},
	}

	return c
}

func cancelTaskRun(p cli.Params, s *cli.Stream, trName string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	var taskrun *v1.TaskRun
	err = actions.GetV1(taskrunGroupResource, cs, trName, p.Namespace(), metav1.GetOptions{}, &taskrun)
	if err != nil {
		return fmt.Errorf("failed to find TaskRun: %s", trName)
	}

	if len(taskrun.Status.Conditions) > 0 {
		if taskrun.Status.Conditions[0].Status != corev1.ConditionUnknown {
			return fmt.Errorf("failed to cancel TaskRun %s: TaskRun has already finished execution", trName)
		}
	}

	if _, err := patch(cs, trName, metav1.PatchOptions{}, p.Namespace()); err != nil {
		return fmt.Errorf("failed to cancel TaskRun %s: %v", trName, err)
	}

	fmt.Fprintf(s.Out, "TaskRun cancelled: %s\n", taskrun.Name)
	return nil
}

func patch(c *cli.Clients, trname string, opts metav1.PatchOptions, ns string) (*v1.TaskRun, error) {
	payload := []patchStringValue{{
		Op:    "replace",
		Path:  "/spec/status",
		Value: v1.TaskRunSpecStatusCancelled,
	}}

	data, _ := json.Marshal(payload)
	var taskrun *v1.TaskRun
	var trGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}
	err := actions.Patch(trGroupResource, c, trname, data, opts, ns, &taskrun)
	if err != nil {
		return nil, err
	}

	return taskrun, nil
}
