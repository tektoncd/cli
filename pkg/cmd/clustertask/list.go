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
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	emptyMsg = "No ClusterTasks found\n"
	body     = "%s\t%s\t%s\n"
	header   = "NAME\tDESCRIPTION\tAGE"
)

type listOptions struct {
	NoHeaders bool
}

func listCommand(p cli.Params) *cobra.Command {
	opts := &listOptions{}
	f := cliopts.NewPrintFlags("list")

	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists ClusterTasks",
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, _ []string) error {

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}

			cs, err := p.Clients()
			if err != nil {
				return err
			}

			if output != "" {
				ctGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "clustertasks"}
				p, err := f.ToPrinter()
				if err != nil {
					return err
				}
				return actions.PrintObjects(ctGroupResource, cmd.OutOrStdout(), cs.Dynamic, cs.Tekton.Discovery(), p, "")
			}
			stream := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}
			return printClusterTaskDetails(stream, p, opts.NoHeaders)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVar(&opts.NoHeaders, "no-headers", opts.NoHeaders, "do not print column headers with output (default print column headers with output)")
	c.Deprecated = "ClusterTasks are deprecated, this command will be removed in future releases."
	return c
}

func printClusterTaskDetails(s *cli.Stream, p cli.Params, noHeaders bool) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	var clustertasks *v1beta1.ClusterTaskList
	err = actions.ListV1(clustertaskGroupResource, cs, metav1.ListOptions{}, "", &clustertasks)
	if err != nil {
		return fmt.Errorf("failed to list ClusterTasks")
	}

	if len(clustertasks.Items) == 0 {
		fmt.Fprint(s.Out, emptyMsg)
		return nil
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)

	if !noHeaders {
		fmt.Fprintln(w, header)
	}

	clusterTaskItems := clustertasks.Items

	for idx := range clusterTaskItems {
		fmt.Fprintf(w, body,
			clusterTaskItems[idx].Name,
			formatted.FormatDesc(clusterTaskItems[idx].Spec.Description),
			formatted.Age(&clusterTaskItems[idx].CreationTimestamp, p.Time()),
		)
	}
	return w.Flush()
}
