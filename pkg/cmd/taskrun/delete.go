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
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	traction "github.com/tektoncd/cli/pkg/actions/delete"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/deleter"
	"github.com/tektoncd/cli/pkg/options"
	trlist "github.com/tektoncd/cli/pkg/taskrun/list"
	validate "github.com/tektoncd/cli/pkg/validate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "taskrun", ForceDelete: false, ParentResource: "task", DeleteAllNs: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete TaskRuns with names 'foo' and 'bar' in namespace 'quux':

    tkn taskrun delete foo bar -n quux

or

    tkn tr rm foo bar -n quux
`

	c := &cobra.Command{
		Use:          "delete",
		Aliases:      []string{"rm"},
		Short:        "Delete taskruns in a namespace",
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

			return deleteTaskRuns(s, p, args, opts)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().StringVarP(&opts.ParentResourceName, "task", "t", "", "The name of a task whose taskruns should be deleted (does not delete the task)")
	c.Flags().BoolVarP(&opts.DeleteAllNs, "all", "", false, "Delete all taskruns in a namespace (default: false)")
	c.Flags().IntVarP(&opts.Keep, "keep", "", 0, "Keep n least recent number of taskruns when using --all with delete")
	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_taskrun")
	return c
}

func deleteTaskRuns(s *cli.Stream, p cli.Params, trNames []string, opts *options.DeleteOptions) error {
	trGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}
	var d *deleter.Deleter
	switch {
	case opts.DeleteAllNs:
		d = deleter.New("TaskRun", func(taskRunName string) error {
			return traction.Delete(trGroupResource, cs, taskRunName, p.Namespace(), &metav1.DeleteOptions{})
		})
		trs, err := allTaskRunNames(cs, opts.Keep, p.Namespace())
		if err != nil {
			return err
		}
		d.Delete(s, trs)
	case opts.ParentResourceName == "":
		d = deleter.New("TaskRun", func(taskRunName string) error {
			return traction.Delete(trGroupResource, cs, taskRunName, p.Namespace(), &metav1.DeleteOptions{})
		})
		d.Delete(s, trNames)
	default:
		d = deleter.New("Task", func(_ string) error {
			return errors.New("the task should not be deleted")
		})
		d.WithRelated("TaskRun", taskRunLister(p, cs), func(taskRunName string) error {
			return traction.Delete(trGroupResource, cs, taskRunName, p.Namespace(), &metav1.DeleteOptions{})
		})
		d.DeleteRelated(s, []string{opts.ParentResourceName})
	}
	if !opts.DeleteAllNs {
		d.PrintSuccesses(s)
	} else if opts.DeleteAllNs {
		if d.Errors() == nil {
			if opts.Keep > 0 {
				fmt.Fprintf(s.Out, "All but %d TaskRuns deleted in namespace %q\n", opts.Keep, p.Namespace())
			} else {
				fmt.Fprintf(s.Out, "All TaskRuns deleted in namespace %q\n", p.Namespace())
			}
		}
	}
	return d.Errors()
}

func taskRunLister(p cli.Params, cs *cli.Clients) func(string) ([]string, error) {
	return func(taskName string) ([]string, error) {
		lOpts := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tekton.dev/task=%s", taskName),
		}
		taskRuns, err := trlist.TaskRuns(cs, lOpts, p.Namespace())
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

func allTaskRunNames(cs *cli.Clients, keep int, ns string) ([]string, error) {

	taskRuns, err := trlist.TaskRuns(cs, metav1.ListOptions{}, ns)
	if err != nil {
		return nil, err
	}
	var names []string
	var counter = 0
	for _, tr := range taskRuns.Items {
		if keep > 0 && counter != keep {
			counter++
			continue
		}
		names = append(names, tr.Name)
	}
	return names, nil
}
