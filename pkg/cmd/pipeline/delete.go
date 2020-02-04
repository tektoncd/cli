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
	"github.com/tektoncd/cli/pkg/helper/deleter"
	"github.com/tektoncd/cli/pkg/helper/options"
	validate "github.com/tektoncd/cli/pkg/helper/validate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "pipeline", ForceDelete: false, DeleteRelated: false}
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

			if err := opts.CheckOptions(s, args, p.Namespace()); err != nil {
				return err
			}

			return deletePipelines(opts, s, p, args)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().BoolVarP(&opts.DeleteRelated, "all", "a", false, "Whether to also delete related resources (pipelineruns) (default: false)")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_pipeline")
	return c
}

func deletePipelines(opts *options.DeleteOptions, s *cli.Stream, p cli.Params, pNames []string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}
	d := deleter.New("Pipeline", func(pipelineName string) error {
		return cs.Tekton.TektonV1alpha1().Pipelines(p.Namespace()).Delete(pipelineName, &metav1.DeleteOptions{})
	})
	d.WithRelated("PipelineRun", pipelineRunLister(p, cs), func(pipelineRunName string) error {
		return cs.Tekton.TektonV1alpha1().PipelineRuns(p.Namespace()).Delete(pipelineRunName, &metav1.DeleteOptions{})
	})
	deletedPipelineNames := d.Delete(s, pNames)
	if opts.DeleteRelated {
		d.DeleteRelated(s, deletedPipelineNames)
	}
	d.PrintSuccesses(s)
	return d.Errors()
}

func pipelineRunLister(p cli.Params, cs *cli.Clients) func(string) ([]string, error) {
	return func(pipelineName string) ([]string, error) {
		lOpts := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tekton.dev/pipeline=%s", pipelineName),
		}
		pipelineRuns, err := cs.Tekton.TektonV1alpha1().PipelineRuns(p.Namespace()).List(lOpts)
		if err != nil {
			return nil, err
		}
		var names []string
		for _, pr := range pipelineRuns.Items {
			names = append(names, pr.Name)
		}
		return names, nil
	}
}
