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
	prdesc "github.com/tektoncd/cli/pkg/pipelinerun/description"
	"github.com/tektoncd/cli/pkg/validate"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func describeCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("describe")
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
		Args:         cobra.MinimumNArgs(1),
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

			if output != "" {
				pipelineRunGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "pipelineruns"}
				return actions.PrintObject(pipelineRunGroupResource, args[0], cmd.OutOrStdout(), p, f, p.Namespace())
			}

			return prdesc.PrintPipelineRunDescription(s, args[0], p)
		},
	}

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_pipelinerun")
	f.AddFlags(c)

	return c
}
