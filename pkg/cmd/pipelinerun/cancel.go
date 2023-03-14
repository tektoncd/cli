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

package pipelinerun

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	pipelinerunpkg "github.com/tektoncd/cli/pkg/pipelinerun"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func cancelCommand(p cli.Params) *cobra.Command {
	eg := `Cancel the PipelineRun named 'foo' from namespace 'bar':

    tkn pipelinerun cancel foo -n bar
`

	graceCancelDescription := `Gracefully cancel a PipelineRun
To use this, you need to change the feature-flags configmap enable-api-fields to alpha instead of stable.
Set to 'CancelledRunFinally' if you want to cancel the current running task and directly run the finally tasks.
Set to 'StoppedRunFinally' if you want to cancel the remaining non-final task and directly run the finally tasks.
`

	graceCancelStatus := ""

	c := &cobra.Command{
		Use:     "cancel",
		Short:   "Cancel a PipelineRun in a namespace",
		Example: eg,

		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		ValidArgsFunction: formatted.ParentCompletion,
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pr := args[0]

			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			return cancelPipelineRun(p, s, pr, graceCancelStatus)
		},
	}

	c.Flags().StringVarP(&graceCancelStatus, "grace", "", "", graceCancelDescription)
	return c
}

func cancelPipelineRun(p cli.Params, s *cli.Stream, prName string, graceCancelStatus string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	var pr *v1.PipelineRun
	err = actions.GetV1(pipelineRunGroupResource, cs, prName, p.Namespace(), metav1.GetOptions{}, &pr)
	if err != nil {
		return fmt.Errorf("failed to find PipelineRun: %s", prName)
	}

	if len(pr.Status.Conditions) > 0 {
		if pr.Status.Conditions[0].Status != corev1.ConditionUnknown {
			return fmt.Errorf("failed to cancel PipelineRun %s: PipelineRun has already finished execution", prName)
		}
	}

	cancelStatus := v1.PipelineRunSpecStatusCancelled
	switch strings.ToLower(graceCancelStatus) {
	case strings.ToLower(v1.PipelineRunSpecStatusCancelledRunFinally):
		cancelStatus = v1.PipelineRunSpecStatusCancelledRunFinally
	case strings.ToLower(v1.PipelineRunSpecStatusStoppedRunFinally):
		cancelStatus = v1.PipelineRunSpecStatusStoppedRunFinally
	}

	if _, err = pipelinerunpkg.Cancel(cs, prName, metav1.PatchOptions{}, cancelStatus, p.Namespace()); err != nil {
		return fmt.Errorf("failed to cancel PipelineRun: %s: %v", prName, err)
	}

	fmt.Fprintf(s.Out, "PipelineRun cancelled: %s\n", pr.Name)
	return nil
}
