// Copyright © 2026 The Tekton Authors.
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

package stepaction

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"text/template"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/exitcode"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const listTemplate = `{{- $sal := len .StepActions.Items -}}{{- if eq $sal 0 -}}
No StepActions found
{{ else -}}
{{- if not $.NoHeaders -}}
{{- if $.AllNamespaces -}}
NAMESPACE	NAME	DESCRIPTION	AGE
{{ else -}}
NAME	DESCRIPTION	AGE
{{ end -}}
{{- end -}}
{{- range $_, $sa := .StepActions.Items -}}{{- if $.AllNamespaces -}}
{{ $sa.Namespace }}	{{ $sa.Name }}	{{ formatDesc $sa.Spec.Description }}	{{ formatAge $sa.CreationTimestamp $.Time }}
{{ else -}}
{{ $sa.Name }}	{{ formatDesc $sa.Spec.Description }}	{{ formatAge $sa.CreationTimestamp $.Time }}
{{ end -}}{{- end -}}
{{- end -}}
`

type listOptions struct {
	Limit         int
	LabelSelector string
	Reverse       bool
	AllNamespaces bool
	NoHeaders     bool
}

func listCommand(p cli.Params) *cobra.Command {
	opts := &listOptions{Limit: 0}
	f := cliopts.NewPrintFlags("list")
	eg := `List all StepActions in namespace 'bar':

    tkn stepaction list -n bar

List StepActions as a JSON array:

    tkn stepaction list -o json

List StepActions as a YAML array:

    tkn stepaction list -o yaml
`

	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists StepActions in a namespace",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Example: eg,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.Limit < 0 {
				return exitcode.New(exitcode.InvalidInput, "limit was %d but must be a positive number", opts.Limit)
			}

			sas, err := list(p, opts.Limit, opts.LabelSelector, opts.AllNamespaces)
			if err != nil {
				return fmt.Errorf("failed to list StepActions from namespace %s: %w", p.Namespace(), err)
			}

			if opts.Reverse {
				reverse(sas)
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}
			if output == "name" {
				w := cmd.OutOrStdout()
				for _, sa := range sas.Items {
					if _, err := fmt.Fprintf(w, "stepaction.tekton.dev/%s\n", sa.Name); err != nil {
						return err
					}
				}
				return nil
			} else if output != "" {
				printer, err := f.ToPrinter()
				if err != nil {
					return err
				}
				return printer.PrintObj(sas, cmd.OutOrStdout())
			}

			stream := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}
			if err := printFormatted(stream, sas, p.Time(), opts.AllNamespaces, opts.NoHeaders); err != nil {
				fmt.Fprint(os.Stderr, "failed to print StepActions \n")
				return err
			}
			return nil
		},
	}

	f.AddFlags(c)
	c.Flags().IntVarP(&opts.Limit, "limit", "", 0, "Limits the number of StepActions. If the limit value is 0 returns all")
	c.Flags().StringVarP(&opts.LabelSelector, "label", "", opts.LabelSelector, "A selector (label query) to filter on, supports '=', '==', and '!='")
	c.Flags().BoolVarP(&opts.Reverse, "reverse", "", opts.Reverse, "list StepActions in reverse order")
	c.Flags().BoolVarP(&opts.AllNamespaces, "all-namespaces", "A", opts.AllNamespaces, "list StepActions from all namespaces")
	c.Flags().BoolVarP(&opts.NoHeaders, "no-headers", "", opts.NoHeaders, "do not print column headers with output (default print column headers with output)")
	return c
}

func reverse(sas *v1beta1.StepActionList) {
	for i, j := 0, len(sas.Items)-1; i < j; i, j = i+1, j-1 {
		sas.Items[i], sas.Items[j] = sas.Items[j], sas.Items[i]
	}
}

func list(p cli.Params, limit int, labelSelector string, allNamespaces bool) (*v1beta1.StepActionList, error) {
	options := metav1.ListOptions{}
	if labelSelector != "" {
		options.LabelSelector = labelSelector
	}

	cs, err := p.Clients()
	if err != nil {
		return nil, err
	}

	ns := p.Namespace()
	if allNamespaces {
		ns = ""
	}

	var sas *v1beta1.StepActionList
	if err := actions.ListV1(stepactionGroupResource, cs, options, ns, &sas); err != nil {
		return nil, err
	}

	// StepActions have no run time, so keep a stable order by namespace and name
	sort.SliceStable(sas.Items, func(i, j int) bool {
		if sas.Items[i].Namespace != sas.Items[j].Namespace {
			return sas.Items[i].Namespace < sas.Items[j].Namespace
		}
		return sas.Items[i].Name < sas.Items[j].Name
	})

	if limit > 0 && len(sas.Items) > limit {
		sas.Items = sas.Items[:limit]
	}

	return sas, nil
}

func printFormatted(s *cli.Stream, sas *v1beta1.StepActionList, c clockwork.Clock, allNamespaces bool, noHeaders bool) error {
	data := struct {
		StepActions   *v1beta1.StepActionList
		Time          clockwork.Clock
		AllNamespaces bool
		NoHeaders     bool
	}{
		StepActions:   sas,
		Time:          c,
		AllNamespaces: allNamespaces,
		NoHeaders:     noHeaders,
	}

	funcMap := template.FuncMap{
		"formatAge":  formatted.Age,
		"formatDesc": formatted.FormatDesc,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("List StepActions").Funcs(funcMap).Parse(listTemplate))
	if err := t.Execute(w, data); err != nil {
		return err
	}
	return w.Flush()
}
