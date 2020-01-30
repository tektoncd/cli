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

package pipelineresource

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/helper/deleter"
	"github.com/tektoncd/cli/pkg/helper/options"
	validateinput "github.com/tektoncd/cli/pkg/helper/validate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "pipelineresource", ForceDelete: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete PipelineResources with names 'foo' and 'bar' in namespace 'quux':

    tkn resource delete foo bar -n quux

or

    tkn res rm foo bar -n quux
`

	c := &cobra.Command{
		Use:          "delete",
		Aliases:      []string{"rm"},
		Short:        "Delete pipeline resources in a namespace",
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

			if err := validateinput.NamespaceExists(p); err != nil {
				return err
			}

			if err := opts.CheckOptions(s, args); err != nil {
				return err
			}

			return deleteResources(s, p, args)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_pipelineresource")
	return c
}

func deleteResources(s *cli.Stream, p cli.Params, preNames []string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	d := deleter.New("PipelineResource", func(resourceName string) error {
		return cs.Resource.TektonV1alpha1().PipelineResources(p.Namespace()).Delete(resourceName, &metav1.DeleteOptions{})
	})
	d.Delete(s, preNames)
	d.PrintSuccesses(s)
	return d.Errors()
}
