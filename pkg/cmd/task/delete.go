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
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/deleter"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/task"
	trlist "github.com/tektoncd/cli/pkg/taskrun/list"
	validate "github.com/tektoncd/cli/pkg/validate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "task", ForceDelete: false, DeleteRelated: false}
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

			return deleteTask(opts, s, p, args)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().BoolVarP(&opts.DeleteRelated, "trs", "", false, "Whether to delete Task(s) and related resources (TaskRuns) (default: false)")
	c.Flags().BoolVarP(&opts.DeleteAllNs, "all", "", false, "Delete all Tasks in a namespace (default: false)")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_task")
	return c
}

func deleteTask(opts *options.DeleteOptions, s *cli.Stream, p cli.Params, taskNames []string) error {
	taskGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "tasks"}
	taskrunGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}

	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}
	d := deleter.New("Task", func(taskName string) error {
		return actions.Delete(taskGroupResource, cs, taskName, p.Namespace(), &metav1.DeleteOptions{})
	})
	switch {
	case opts.DeleteAllNs:
		taskNames, err = allTaskNames(cs, p.Namespace())
		if err != nil {
			return err
		}
		d.Delete(s, taskNames)
	case opts.DeleteRelated:
		d.WithRelated("TaskRun", taskRunLister(cs, p.Namespace()), func(taskRunName string) error {
			return actions.Delete(taskrunGroupResource, cs, taskRunName, p.Namespace(), &metav1.DeleteOptions{})
		})
		deletedTaskNames := d.Delete(s, taskNames)
		d.DeleteRelated(s, deletedTaskNames)
	default:
		d.Delete(s, taskNames)
	}
	if !opts.DeleteAllNs {
		d.PrintSuccesses(s)
	} else if opts.DeleteAllNs {
		if d.Errors() == nil {
			fmt.Fprintf(s.Out, "All Tasks deleted in namespace %q\n", p.Namespace())
		}
	}
	return d.Errors()
}

func taskRunLister(cs *cli.Clients, ns string) func(string) ([]string, error) {
	return func(taskName string) ([]string, error) {
		lOpts := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tekton.dev/task=%s", taskName),
		}
		taskRuns, err := trlist.TaskRuns(cs, lOpts, ns)
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

func allTaskNames(cs *cli.Clients, ns string) ([]string, error) {
	ts, err := task.List(cs, metav1.ListOptions{}, ns)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, t := range ts.Items {
		names = append(names, t.Name)
	}
	return names, nil
}
