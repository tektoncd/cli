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

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/options"
	prhelper "github.com/tektoncd/cli/pkg/pipelinerun"
	prdesc "github.com/tektoncd/cli/pkg/pipelinerun/description"
	"github.com/tektoncd/cli/pkg/validate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
		Short:        "Describe a pipelinerun in a namespace",
		Example:      eg,
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if err := validate.NamespaceExists(p); err != nil {
				return err
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				fmt.Fprint(os.Stderr, "Error: output option not set properly \n")
				return err
			}

			if !opts.Fzf {
				if _, ok := os.LookupEnv("TKN_USE_FZF"); ok {
					opts.Fzf = true
				}
			}

			if len(args) == 0 {
				err = askPipelineRunName(opts, p)
				if err != nil {
					return err
				}
			} else {
				opts.PipelineRunName = args[0]
			}

			if output != "" {
				pipelineRunGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "pipelineruns"}
				return actions.PrintObject(pipelineRunGroupResource, opts.PipelineRunName, cmd.OutOrStdout(), p, f, p.Namespace())
			}

			return prdesc.PrintPipelineRunDescription(s, opts.PipelineRunName, p)
		},
	}

	c.Flags().IntVarP(&opts.Limit, "limit", "", defaultDescribeLimit, "lists number of pipelineruns when selecting a pipelinerun to describe")
	c.Flags().BoolVarP(&opts.Fzf, "fzf", "F", false, "use fzf to select a pipelinerun to describe")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_pipelinerun")
	f.AddFlags(c)

	return c
}

func askPipelineRunName(opts *options.DescribeOptions, p cli.Params) error {
	lOpts := metav1.ListOptions{}

	err := opts.ValidateOpts()
	if err != nil {
		return err
	}
	prs, err := prhelper.GetAllPipelineRuns(opts.Params, lOpts, opts.Limit)

	if err != nil {
		return err
	}
	if len(prs) == 0 {
		return fmt.Errorf("no pipelineruns found")
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
