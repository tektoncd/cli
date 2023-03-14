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
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	pipelinerunpkg "github.com/tektoncd/cli/pkg/pipelinerun"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	defaultDescribeLimit = 5
)

func describeCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("describe")
	opts := &options.DescribeOptions{Params: p}
	eg := `Describe a PipelineRun of name 'foo' in namespace 'bar':

    tkn pipelinerun describe foo -n bar

or

    tkn pr desc foo -n bar
`

	c := &cobra.Command{
		Use:          "describe",
		Aliases:      []string{"desc"},
		Short:        "Describe a PipelineRun in a namespace",
		Example:      eg,
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

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}

			if !opts.Fzf {
				if _, ok := os.LookupEnv("TKN_USE_FZF"); ok {
					opts.Fzf = true
				}
			}

			cs, err := p.Clients()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				lOpts := metav1.ListOptions{}
				if !opts.Last {
					prs, err := pipelinerunpkg.GetAllPipelineRuns(pipelineRunGroupResource, lOpts, cs, p.Namespace(), opts.Limit, p.Time())
					if err != nil {
						return err
					}
					if len(prs) == 1 {
						opts.PipelineRunName = strings.Fields(prs[0])[0]
					} else {
						err = askPipelineRunName(opts, prs)
						if err != nil {
							return err
						}
					}
				} else {
					prs, err := pipelinerunpkg.GetAllPipelineRuns(pipelineRunGroupResource, lOpts, cs, p.Namespace(), 1, p.Time())
					if err != nil {
						return err
					}
					if len(prs) == 0 {
						fmt.Fprintf(s.Out, "No PipelineRuns present in namespace %s\n", opts.Params.Namespace())
						return nil
					}
					opts.PipelineRunName = strings.Fields(prs[0])[0]
				}
			} else {
				opts.PipelineRunName = args[0]
			}

			if output != "" {
				return actions.PrintObjectV1(pipelineRunGroupResource, opts.PipelineRunName, cmd.OutOrStdout(), cs, f, p.Namespace())
			}

			return pipelinerunpkg.PrintPipelineRunDescription(s.Out, cs, opts.Params.Namespace(), opts.PipelineRunName, opts.Params.Time())
		},
	}

	c.Flags().BoolVarP(&opts.Last, "last", "L", false, "show description for last PipelineRun")
	c.Flags().IntVarP(&opts.Limit, "limit", "", defaultDescribeLimit, "lists number of PipelineRuns when selecting a PipelineRun to describe")
	c.Flags().BoolVarP(&opts.Fzf, "fzf", "F", false, "use fzf to select a PipelineRun to describe")

	f.AddFlags(c)

	return c
}

func askPipelineRunName(opts *options.DescribeOptions, prs []string) error {
	err := opts.ValidateOpts()
	if err != nil {
		return err
	}

	if len(prs) == 0 {
		return fmt.Errorf("no PipelineRuns found")
	}

	if opts.Fzf {
		err = opts.FuzzyAsk(options.ResourceNamePipelineRun, prs)
	} else {
		err = opts.Ask(options.ResourceNamePipelineRun, prs)
	}
	if err != nil {
		return err
	}

	return nil
}
