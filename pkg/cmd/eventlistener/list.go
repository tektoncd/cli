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

package eventlistener

import (
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/eventlistener"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	emptyMsg = "No eventlisteners found"
)

type listOptions struct {
	AllNamespaces bool
	NoHeaders     bool
}

func listCommand(p cli.Params) *cobra.Command {
	opts := &listOptions{}
	f := cliopts.NewPrintFlags("list")

	eg := `List all EventListeners in namespace 'bar':

	tkn eventlistener list -n bar

or

	tkn el ls -n bar
`

	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists EventListeners in a namespace",
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

			els, err := eventlistener.List(cs, metav1.ListOptions{}, namespace)
			if err != nil {
				if opts.AllNamespaces {
					return fmt.Errorf("failed to list EventListeners from all namespaces: %v", err)
				}
				return fmt.Errorf("failed to list EventListeners from %s namespace: %v", namespace, err)
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return errors.New(`output option not set properly \n`)
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
				return p.PrintObj(els, stream.Out)
			}

			if err = printFormatted(stream, els, p, opts.AllNamespaces, opts.NoHeaders); err != nil {
				return errors.New(`failed to print EventListeners`)
			}
			return nil
		},
	}

	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.AllNamespaces, "all-namespaces", "A", opts.AllNamespaces, "list EventListeners from all namespaces")
	c.Flags().BoolVar(&opts.NoHeaders, "no-headers", opts.NoHeaders, "do not print column headers with output (default print column headers with output)")
	return c
}

func printFormatted(s *cli.Stream, els *v1beta1.EventListenerList, p cli.Params, allNamespaces bool, noHeaders bool) error {
	if len(els.Items) == 0 {
		fmt.Fprintln(s.Err, emptyMsg)
		return nil
	}

	headers := "NAME\tAGE\tURL\tAVAILABLE"
	if allNamespaces {
		headers = "NAMESPACE\t" + headers
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	if !noHeaders {
		fmt.Fprintln(w, headers)
	}

	eventListeners := els.Items

	for idx := range eventListeners {

		status := corev1.ConditionStatus("---")
		if len(eventListeners[idx].Status.Conditions) > 0 && len(eventListeners[idx].Status.Conditions[0].Status) > 0 {
			status = eventListeners[idx].Status.Conditions[0].Status
		}

		if allNamespaces {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				eventListeners[idx].Namespace,
				eventListeners[idx].Name,
				formatted.Age(&eventListeners[idx].CreationTimestamp, p.Time()),
				formatted.FormatAddress(getURL(eventListeners[idx])),
				status,
			)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				eventListeners[idx].Name,
				formatted.Age(&eventListeners[idx].CreationTimestamp, p.Time()),
				formatted.FormatAddress(getURL(eventListeners[idx])),
				status,
			)
		}
	}
	return w.Flush()
}
