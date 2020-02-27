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
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/printer"
	trdesc "github.com/tektoncd/cli/pkg/taskrun/description"
	validate "github.com/tektoncd/cli/pkg/validate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func describeCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("describe")
	eg := `Describe a TaskRun of name 'foo' in namespace 'bar':

    tkn taskrun describe foo -n bar

or

    tkn tr desc foo -n bar
`

	c := &cobra.Command{
		Use:          "describe",
		Aliases:      []string{"desc"},
		Short:        "Describe a taskrun in a namespace",
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
				return describeTaskRunOutput(cmd.OutOrStdout(), p, f, args[0])
			}

			return trdesc.PrintTaskRunDescription(s, args[0], p)
		},
	}

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_taskrun")
	f.AddFlags(c)

	return c
}

func describeTaskRunOutput(w io.Writer, p cli.Params, f *cliopts.PrintFlags, name string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	c := cs.Tekton.TektonV1alpha1().TaskRuns(p.Namespace())

	taskrun, err := c.Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// NOTE: this is required for -o json|yaml to work properly since
	// tektoncd go client fails to set these; probably a bug
	taskrun.GetObjectKind().SetGroupVersionKind(
		schema.GroupVersionKind{
			Version: "tekton.dev/v1alpha1",
			Kind:    "TaskRun",
		})

	return printer.PrintObject(w, taskrun, f)
}
