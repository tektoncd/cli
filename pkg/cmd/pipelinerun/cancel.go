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

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/pipelinerun"
	"go.uber.org/multierr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

type cancelOptions struct {
	PipelineName string
}

func cancelCommand(p cli.Params) *cobra.Command {
	opts := &options.CancelOptions{CancelAll: false}
	cancelOpts := &cancelOptions{}
	f := cliopts.NewPrintFlags("cancel")
	eg := `Cancel the PipelineRun named 'foo' from namespace 'bar':

    tkn pipelinerun cancel foo -n bar
`

	c := &cobra.Command{
		Use:          "cancel",
		Short:        "Cancel a PipelineRun in a namespace",
		Example:      eg,
		Args:         cobra.RangeArgs(0, 1),
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		ValidArgsFunction: formatted.ParentCompletion,

		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if opts.CancelAll {
				if cancelOpts.PipelineName != "" {
					return cancelPipelineRuns(p, s, cancelOpts.PipelineName)
				}
				return fmt.Errorf("must provide Pipeline name with --all flag")
			}

			pr := args[0]
			return cancelPipelineRun(p, s, pr)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.CancelAll, "all", "", false, "Cancel all PipelineRuns associated with a particular Pipeline")
	c.Flags().StringVarP(&cancelOpts.PipelineName, "pipeline", "p", "", "The name of a Pipeline whose PipelineRuns should be cancelled")
	return c
}

func cancelPipelineRun(p cli.Params, s *cli.Stream, prName string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	pr, err := pipelinerun.GetV1beta1(cs, prName, metav1.GetOptions{}, p.Namespace())
	if err != nil {
		return fmt.Errorf("failed to find PipelineRun: %s", prName)
	}

	if len(pr.Status.Conditions) > 0 {
		if pr.Status.Conditions[0].Status != corev1.ConditionUnknown {
			return fmt.Errorf("failed to cancel PipelineRun %s: PipelineRun has already finished execution", prName)
		}
	}

	if _, err = pipelinerun.Patch(cs, prName, metav1.PatchOptions{}, p.Namespace()); err != nil {
		return fmt.Errorf("failed to cancel PipelineRun: %s: %v", prName, err)

	}

	fmt.Fprintf(s.Out, "PipelineRun cancelled: %s\n", pr.Name)
	return nil
}

func cancelPipelineRuns(p cli.Params, s *cli.Stream, pipelineName string) error {
	var errors []error

	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}
	selector := fmt.Sprintf("tekton.dev/pipeline=%s", pipelineName)
	options := v1.ListOptions{
		LabelSelector: selector,
	}
	prs, err := pipelinerun.List(cs, options, p.Namespace())
	if err != nil {
		return err
	}

	for _, pr := range prs.Items {
		if len(pr.Status.Conditions) > 0 {
			if pr.Status.Conditions[0].Status != corev1.ConditionUnknown {
				errors = append(errors, fmt.Errorf("failed to cancel PipelineRun %s: PipelineRun has already finished execution", pr.Name))
			}
		}

		if _, err = pipelinerun.Patch(cs, pr.Name, metav1.PatchOptions{}, p.Namespace()); err != nil {
			errors = append(errors, fmt.Errorf("failed to cancel PipelineRun: %s: %v", pr.Name, err))
		}
	}

	fmt.Fprintf(s.Out, "All PipelineRuns associated with Pipeline %q cancelled in namespace %q\n", pipelineName, p.Namespace())
	return multierr.Combine(errors...)
}
