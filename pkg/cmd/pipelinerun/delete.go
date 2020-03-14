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
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/deleter"
	"github.com/tektoncd/cli/pkg/options"
	prsort "github.com/tektoncd/cli/pkg/pipelinerun/sort"
	validate "github.com/tektoncd/cli/pkg/validate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "pipelinerun", ForceDelete: false, ParentResource: "pipeline", DeleteAllNs: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete PipelineRuns with names 'foo' and 'bar' in namespace 'quux':

    tkn pipelinerun delete foo bar -n quux

or

    tkn pr rm foo bar -n quux
`

	c := &cobra.Command{
		Use:          "delete",
		Aliases:      []string{"rm"},
		Short:        "Delete pipelineruns in a namespace",
		Example:      eg,
		Args:         cobra.MinimumNArgs(0),
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

			return deletePipelineRuns(s, p, args, opts)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().StringVarP(&opts.ParentResourceName, "pipeline", "p", "", "The name of a pipeline whose pipelineruns should be deleted (does not delete the pipeline)")
	c.Flags().IntVarP(&opts.Keep, "keep", "", 0, "Keep n least recent number of pipelineruns when using --all with delete")
	c.Flags().BoolVarP(&opts.DeleteAllNs, "all", "", false, "Delete all pipelineruns in a namespace (default: false)")
	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_pipelinerun")
	return c
}

func deletePipelineRuns(s *cli.Stream, p cli.Params, prNames []string, opts *options.DeleteOptions) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}
	var d *deleter.Deleter
	switch {
	case opts.DeleteAllNs:
		d = deleter.New("PipelineRun", func(pipelineRunName string) error {
			return cs.Tekton.TektonV1alpha1().PipelineRuns(p.Namespace()).Delete(pipelineRunName, &metav1.DeleteOptions{})
		})
		prs, err := allPipelineRunNames(p, cs, opts.Keep)
		if err != nil {
			return err
		}
		d.Delete(s, prs)
	case opts.ParentResourceName == "":
		d = deleter.New("PipelineRun", func(pipelineRunName string) error {
			return cs.Tekton.TektonV1alpha1().PipelineRuns(p.Namespace()).Delete(pipelineRunName, &metav1.DeleteOptions{})
		})
		d.Delete(s, prNames)
	default:
		d = deleter.New("Pipeline", func(_ string) error {
			return errors.New("the pipeline should not be deleted")
		})
		d.WithRelated("PipelineRun", pipelineRunLister(p, cs), func(pipelineRunName string) error {
			return cs.Tekton.TektonV1alpha1().PipelineRuns(p.Namespace()).Delete(pipelineRunName, &metav1.DeleteOptions{})
		})
		d.DeleteRelated(s, []string{opts.ParentResourceName})
	}
	if !opts.DeleteAllNs {
		d.PrintSuccesses(s)
	} else if opts.DeleteAllNs {
		if d.Errors() == nil {
			if opts.Keep > 0 {
				fmt.Fprintf(s.Out, "All but %d PipelineRuns deleted in namespace %q\n", opts.Keep, p.Namespace())
			} else {
				fmt.Fprintf(s.Out, "All PipelineRuns deleted in namespace %q\n", p.Namespace())
			}
		}
	}
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

func allPipelineRunNames(p cli.Params, cs *cli.Clients, keep int) ([]string, error) {
	pipelineRuns, err := cs.Tekton.TektonV1alpha1().PipelineRuns(p.Namespace()).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var names []string
	var counter = 0
	prsort.SortByStartTime(pipelineRuns.Items)
	for _, pr := range pipelineRuns.Items {
		if keep > 0 && counter != keep {
			counter++
			continue
		}
		names = append(names, pr.Name)
	}
	return names, nil
}
