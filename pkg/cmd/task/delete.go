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
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/task"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

// taskExists validates that the arguments are valid Task names
func taskExists(args []string, p cli.Params) ([]string, error) {

	availableTaskNames := make([]string, 0)
	c, err := p.Clients()
	if err != nil {
		return availableTaskNames, err
	}
	var errorList error
	ns := p.Namespace()
	for _, name := range args {
		var task *v1.Task
		err := actions.GetV1(taskGroupResource, c, name, ns, metav1.GetOptions{}, &task)
		if err != nil {
			errorList = multierr.Append(errorList, err)
			continue
		}
		availableTaskNames = append(availableTaskNames, name)
	}
	return availableTaskNames, errorList
}

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "Task", ForceDelete: false, DeleteRelated: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete Tasks with names 'foo' and 'bar' in namespace 'quux':

    tkn task delete foo bar -n quux

or

    tkn t rm foo bar -n quux
`

	c := &cobra.Command{
		Use:     "delete",
		Aliases: []string{"rm"},
		Short:   "Delete Tasks in a namespace",
		Example: eg,
		Args:    cobra.MinimumNArgs(0),

		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		ValidArgsFunction: formatted.ParentCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				In:  cmd.InOrStdin(),
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			availableTaskNames, errs := taskExists(args, p)
			if len(availableTaskNames) == 0 && errs != nil {
				return errs
			}

			if err := opts.CheckOptions(s, availableTaskNames, p.Namespace()); err != nil {
				return err
			}

			if err := deleteTask(opts, s, p, availableTaskNames); err != nil {
				return err
			}
			return errs
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().BoolVarP(&opts.DeleteRelated, "trs", "", false, "Whether to delete Task(s) and related resources (TaskRuns) (default: false)")
	c.Flags().BoolVarP(&opts.DeleteAllNs, "all", "", false, "Delete all Tasks in a namespace (default: false)")

	return c
}

func deleteTask(opts *options.DeleteOptions, s *cli.Stream, p cli.Params, taskNames []string) error {
	taskrunGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}

	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}
	d := deleter.New("Task", func(taskName string) error {
		return actions.Delete(taskGroupResource, cs.Dynamic, cs.Tekton.Discovery(), taskName, p.Namespace(), metav1.DeleteOptions{})
	})
	switch {
	case opts.DeleteAllNs:
		taskNames, err = task.GetAllTaskNames(taskGroupResource, cs, p.Namespace())
		if err != nil {
			return err
		}
		d.Delete(taskNames)
	case opts.DeleteRelated:
		d.WithRelated("TaskRun", taskRunLister(cs, p.Namespace()), func(taskRunName string) error {
			return actions.Delete(taskrunGroupResource, cs.Dynamic, cs.Tekton.Discovery(), taskRunName, p.Namespace(), metav1.DeleteOptions{})
		})
		deletedTaskNames := d.Delete(taskNames)
		d.DeleteRelated(deletedTaskNames)
	default:
		d.Delete(taskNames)
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

		var taskRuns *v1.TaskRunList
		if err := actions.ListV1(taskrunGroupResource, cs, lOpts, ns, &taskRuns); err != nil {
			return nil, err
		}

		// this is required as the same label is getting added for Task
		taskRuns.Items = task.FilterByRef(taskRuns.Items, "Task")
		var names []string
		for _, tr := range taskRuns.Items {
			names = append(names, tr.Name)
		}
		return names, nil
	}
}
