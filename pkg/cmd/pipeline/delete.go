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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/helper/names"
	"github.com/tektoncd/cli/pkg/helper/options"
	validate "github.com/tektoncd/cli/pkg/helper/validate"
	multierr "go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "pipeline", ForceDelete: false, DeleteAll: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete Pipelines with names 'foo' and 'bar' in namespace 'quux'

    tkn pipeline delete foo bar -n quux

or

    tkn p rm foo bar -n quux
`

	c := &cobra.Command{
		Use:          "delete",
		Aliases:      []string{"rm"},
		Short:        "Delete pipelines in a namespace",
		Example:      eg,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				In:  cmd.InOrStdin(),
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if err := validate.NamespaceExists(p); err != nil {
				return err
			}

			if err := opts.CheckOptions(s, args); err != nil {
				return err
			}

			return deletePipelines(opts, s, p, args)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().BoolVarP(&opts.DeleteAll, "all", "a", false, "Whether to delete related resources (pipelineruns) (default: false)")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_pipeline")
	return c
}

func deletePipelines(opts *options.DeleteOptions, s *cli.Stream, p cli.Params, pNames []string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	var errs []error
	addPrintErr := func(err error) {
		errs = append(errs, err)
		fmt.Fprintf(s.Err, "%s\n", err)
	}

	var successfulPipelines []string
	var successfulPipelineRuns []string

	for _, pName := range pNames {
		if err := cs.Tekton.TektonV1alpha1().Pipelines(p.Namespace()).Delete(pName, &metav1.DeleteOptions{}); err != nil {
			addPrintErr(fmt.Errorf("failed to delete pipeline %q: %s", pName, err))
			continue
		}
		successfulPipelines = append(successfulPipelines, pName)

		if !opts.DeleteAll {
			continue
		}

		lOpts := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tekton.dev/pipeline=%s", pName),
		}

		pipelineRuns, err := cs.Tekton.TektonV1alpha1().PipelineRuns(p.Namespace()).List(lOpts)
		if err != nil {
			addPrintErr(err)
			continue
		}

		for _, pr := range pipelineRuns.Items {
			if err := cs.Tekton.TektonV1alpha1().PipelineRuns(p.Namespace()).Delete(pr.Name, &metav1.DeleteOptions{}); err != nil {
				addPrintErr(fmt.Errorf("failed to delete pipelinerun %q: %s", pr.Name, err))
				continue
			}

			successfulPipelineRuns = append(successfulPipelineRuns, pr.Name)
		}
	}

	if len(successfulPipelineRuns) > 0 {
		fmt.Fprintf(s.Out, "PipelineRuns deleted: %s\n", names.QuotedList(successfulPipelineRuns))
	}
	if len(successfulPipelines) > 0 {
		fmt.Fprintf(s.Out, "Pipelines deleted: %s\n", names.QuotedList(successfulPipelines))
	}

	return multierr.Combine(errs...)
}
