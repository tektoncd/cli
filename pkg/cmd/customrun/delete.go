// Copyright © 2024 The Tekton Authors.
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

package customrun

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func customRunExists(client *cli.Clients, namespace string, crName string) error {
	var cr *v1beta1.CustomRun
	err := actions.GetV1(customrunGroupResource, client, crName, namespace, metav1.GetOptions{}, &cr)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		return fmt.Errorf("CustomRun %s not found in namespace %s", crName, namespace)
	}
	return nil
}

func deleteCommand(p cli.Params) *cobra.Command {
	f := genericclioptions.NewPrintFlags("delete")
	eg := `Delete CustomRun with name 'foo' in namespace 'bar':

    tkn customrun delete foo -n bar

or

    tkn cr rm foo -n bar
`

	c := &cobra.Command{
		Use:     "delete",
		Aliases: []string{"rm"},
		Short:   "Delete CustomRuns in a namespace",
		Example: eg,
		Args:    cobra.MinimumNArgs(1), // Requires at least one argument (customrun-name)
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			crNames := args
			s := &cli.Stream{
				In:  cmd.InOrStdin(),
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			return deleteCustomRuns(s, p, crNames)

		},
	}

	f.AddFlags(c)
	return c
}

func deleteCustomRuns(s *cli.Stream, p cli.Params, crNames []string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client: %w", err)
	}
	namespace := p.Namespace()
	for _, crName := range crNames {
		// Check if CustomRun exists before attempting deletion
		err := customRunExists(cs, namespace, crName)
		if err != nil {
			fmt.Fprintf(s.Err, "CustomRun %s not found in namespace %s\n", crName, namespace)
			continue
		}

		// Proceed with deletion
		err = deleteCustomRun(cs, namespace, crName)
		if err == nil {
			fmt.Fprintf(s.Out, "CustomRun '%s' deleted successfully from namespace '%s'\n", crName, namespace)
		} else {
			fmt.Fprintf(s.Err, "failed to delete CustomRun %s: %v\n", crName, err)
			return err
		}
	}
	return nil
}

func deleteCustomRun(cs *cli.Clients, namespace, crName string) error {
	err := cs.Dynamic.Resource(customrunGroupResource).Namespace(namespace).Delete(context.TODO(), crName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete CustomRun %s: %w", crName, err)
	}
	return nil
}
