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

package task

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
	opts := &options.DeleteOptions{Resource: "task", ForceDelete: false, DeleteAll: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete Tasks with names 'foo' and 'bar' in namespace 'quux':

    tkn task delete foo bar -n quux

or

    tkn t rm foo bar -n quux
`

	c := &cobra.Command{
		Use:          "delete",
		Aliases:      []string{"rm"},
		Short:        "Delete task resources in a namespace",
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

			return deleteTask(opts, s, p, args)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().BoolVarP(&opts.DeleteAll, "all", "a", false, "Whether to delete related resources (taskruns) (default: false)")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_task")
	return c
}

func deleteTask(opts *options.DeleteOptions, s *cli.Stream, p cli.Params, taskNames []string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}
	d := deleter.New("Task", func(taskName string) error {
		return cs.Tekton.TektonV1alpha1().Tasks(p.Namespace()).Delete(taskName, &metav1.DeleteOptions{})
	})
	if opts.DeleteAll {
		d.WithRelated("TaskRun", taskRunLister(p, cs), func(taskRunName string) error {
			return cs.Tekton.TektonV1alpha1().TaskRuns(p.Namespace()).Delete(taskRunName, &metav1.DeleteOptions{})
		})
	}
	return d.Execute(s, taskNames)
}

func taskRunLister(p cli.Params, cs *cli.Clients) func(string) ([]string, error) {
	return func(taskName string) ([]string, error) {
		lOpts := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tekton.dev/task=%s", taskName),
		}
		taskRuns, err := cs.Tekton.TektonV1alpha1().TaskRuns(p.Namespace()).List(lOpts)
		if err != nil {
			return nil, err
		}
		var names []string
		for _, tr := range taskRuns.Items {
			names = append(names, tr.Name)
		}
		return names, nil
	}
}
