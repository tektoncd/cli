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
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions/list"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	validate "github.com/tektoncd/cli/pkg/validate"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	emptyMsg = "No tasks found"
	header   = "NAME\tDESCRIPTION\tAGE"
	body     = "%s\t%s\t%s\n"
)

func listCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("list")

	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists tasks in a namespace",
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			if err := validate.NamespaceExists(p); err != nil {
				return err
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				fmt.Fprint(os.Stderr, "Error: output option not set properly \n")
				return err
			}

			if output != "" {
				taskGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "tasks"}
				return list.PrintObject(taskGroupResource, cmd.OutOrStdout(), p, f, p.Namespace())
			}
			stream := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}
			return printTaskDetails(stream, p)
		},
	}
	f.AddFlags(c)

	return c
}

func printTaskDetails(s *cli.Stream, p cli.Params) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	unstructuredTask, err := list.AllObjecs(schema.GroupVersionResource{Group: "tekton.dev", Resource: "tasks"}, cs, p.Namespace(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	var tasks v1beta1.TaskList
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredTask.UnstructuredContent(), &tasks); err != nil {
		return err
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list tasks from %s namespace \n", p.Namespace())
		return err
	}

	if len(tasks.Items) == 0 {
		fmt.Fprintln(s.Err, emptyMsg)
		return nil
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, header)

	for _, task := range tasks.Items {
		fmt.Fprintf(w, body,
			task.Name,
			formatted.FormatDesc(task.Spec.Description),
			formatted.Age(&task.CreationTimestamp, p.Time()),
		)
	}
	return w.Flush()
}
