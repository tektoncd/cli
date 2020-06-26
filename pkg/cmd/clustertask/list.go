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

package clustertask

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/clustertask"
	"github.com/tektoncd/cli/pkg/formatted"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	emptyMsg = "No ClusterTasks found"
	header   = "NAME\tDESCRIPTION\tAGE"
	body     = "%s\t%s\t%s\n"
)

func listCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("list")

	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists ClusterTasks",
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}

			if output != "" {
				ctGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "clustertasks"}
				return actions.PrintObjects(ctGroupResource, cmd.OutOrStdout(), p, f, "")
			}
			stream := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}
			return printClusterTaskDetails(stream, p)
		},
	}
	f.AddFlags(c)

	return c
}

func printClusterTaskDetails(s *cli.Stream, p cli.Params) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	clustertasks, err := clustertask.List(cs, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list ClusterTasks")
	}

	if len(clustertasks.Items) == 0 {
		fmt.Fprint(s.Out, emptyMsg)
		return nil
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, header)

	for _, clustertask := range clustertasks.Items {
		fmt.Fprintf(w, body,
			clustertask.Name,
			formatted.FormatDesc(clustertask.Spec.Description),
			formatted.Age(&clustertask.CreationTimestamp, p.Time()),
		)
	}
	return w.Flush()
}
