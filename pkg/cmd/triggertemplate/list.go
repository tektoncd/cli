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

package triggertemplate

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/triggertemplate"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	emptyMsg = "No TriggerTemplates found"
)

type ListOptions struct {
	AllNamespaces bool
	NoHeaders     bool
}

func listCommand(p cli.Params) *cobra.Command {
	opts := &ListOptions{}
	f := cliopts.NewPrintFlags("list")

	eg := `List all TriggerTemplates in namespace 'bar':

	tkn triggertemplate list -n bar

or

	tkn tt ls -n bar
`

	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists TriggerTemplates in a namespace",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Example: eg,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cs, err := p.Clients()
			if err != nil {
				return err
			}

			namespace := p.Namespace()
			if opts.AllNamespaces {
				namespace = ""
			}
			tts, err := triggertemplate.List(cs, metav1.ListOptions{}, namespace)
			if err != nil {
				if opts.AllNamespaces {
					return fmt.Errorf("failed to list TriggerTemplates from all namespaces: %v", err)
				}
				return fmt.Errorf("failed to list TriggerTemplates from %s namespace: %v", namespace, err)
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}

			stream := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if output != "" {
				p, err := f.ToPrinter()
				if err != nil {
					return err
				}
				return p.PrintObj(tts, stream.Out)
			}

			if err = printFormatted(stream, tts, p, opts.AllNamespaces, opts.NoHeaders); err != nil {
				return fmt.Errorf("failed to print TriggerTemplates: %v", err)
			}
			return nil

		},
	}

	f.AddFlags(c)

	c.Flags().BoolVarP(&opts.AllNamespaces, "all-namespaces", "A", opts.AllNamespaces, "list TriggerTemplates from all namespaces")
	c.Flags().BoolVar(&opts.NoHeaders, "no-headers", opts.NoHeaders, "do not print column headers with output (default print column headers with output)")
	return c
}

func printFormatted(s *cli.Stream, tts *v1beta1.TriggerTemplateList, p cli.Params, allNamespaces bool, noHeaders bool) error {
	if len(tts.Items) == 0 {
		fmt.Fprintln(s.Err, emptyMsg)
		return nil
	}

	headers := "NAME\tAGE"
	if allNamespaces {
		headers = "NAMESPACE\t" + headers
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	if !noHeaders {
		fmt.Fprintln(w, headers)
	}

	triggerTemplates := tts.Items

	for idx := range triggerTemplates {
		if allNamespaces {
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				triggerTemplates[idx].Namespace,
				triggerTemplates[idx].Name,
				formatted.Age(&triggerTemplates[idx].CreationTimestamp, p.Time()),
			)
		} else {
			fmt.Fprintf(w, "%s\t%s\n",
				triggerTemplates[idx].Name,
				formatted.Age(&triggerTemplates[idx].CreationTimestamp, p.Time()),
			)
		}
	}

	return w.Flush()
}
