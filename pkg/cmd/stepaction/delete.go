// Copyright © 2026 The Tekton Authors.
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

package stepaction

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/exitcode"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/multierr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func stepActionExists(client *cli.Clients, namespace, name string) error {
	var sa *v1beta1.StepAction
	if err := actions.GetV1(stepactionGroupResource, client, name, namespace, metav1.GetOptions{}, &sa); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		return &exitcode.Error{Code: exitcode.NotFound, Message: fmt.Sprintf("StepAction %s not found in namespace %s", name, namespace), Err: err}
	}
	return nil
}

func deleteCommand(p cli.Params) *cobra.Command {
	eg := `Delete StepAction with name 'foo' in namespace 'bar':

    tkn stepaction delete foo -n bar

or

    tkn stepaction rm foo -n bar

Delete a StepAction and print the result as JSON:

    tkn stepaction delete foo -n bar -o json

Delete a StepAction and print the result as YAML:

    tkn stepaction delete foo -n bar -o yaml
`

	output := ""

	c := &cobra.Command{
		Use:     "delete",
		Aliases: []string{"rm"},
		Short:   "Delete StepActions in a namespace",
		Example: eg,
		Args:    cobra.MinimumNArgs(1),
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				In:  cmd.InOrStdin(),
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if output != "" {
				output = formatted.NormalizeOutput(output)
				if !formatted.IsStructured(output) {
					return exitcode.New(exitcode.InvalidInput, "invalid output format %q: must be json or yaml", output)
				}
			}

			return deleteStepActions(s, p, args, output)
		},
	}

	c.Flags().StringVarP(&output, "output", "o", "", formatted.OutputFlagUsage)
	return c
}

func deleteStepActions(s *cli.Stream, p cli.Params, names []string, output string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client: %w", err)
	}
	namespace := p.Namespace()
	var deleted []string
	var deleteErr error
	for _, name := range names {
		if err := stepActionExists(cs, namespace, name); err != nil {
			deleteErr = multierr.Append(deleteErr, err)
			continue
		}
		if err := deleteStepAction(cs, namespace, name); err != nil {
			deleteErr = multierr.Append(deleteErr, err)
			continue
		}
		deleted = append(deleted, name)
		if !formatted.IsStructured(output) {
			fmt.Fprintf(s.Out, "StepAction '%s' deleted successfully from namespace '%s'\n", name, namespace)
		}
	}

	if formatted.IsStructured(output) {
		if err := formatted.PrintStructuredOutput(s.Out, output, formatted.NewDeleteResult(deleted)); err != nil {
			return err
		}
	}
	return deleteErr
}

func deleteStepAction(cs *cli.Clients, namespace, name string) error {
	if err := cs.Dynamic.Resource(stepactionGroupResource).Namespace(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete StepAction %s: %w", name, err)
	}
	return nil
}
