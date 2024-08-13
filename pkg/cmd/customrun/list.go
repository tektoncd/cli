// Copyright © 2023 The Tekton Authors.
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

package customrun

import (
	"fmt"
	"os"
	"text/tabwriter"
	"text/template"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	crsort "github.com/tektoncd/cli/pkg/customrun/sort"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const ListTemplate = `{{- $trl := len .CustomRuns.Items -}}{{- if eq $trl 0 -}}
No CustomRuns found
{{ else -}}
{{- if not $.NoHeaders -}}
{{- if $.AllNamespaces -}}
NAMESPACE	NAME	STARTED	DURATION	STATUS
{{ else -}}
NAME	STARTED	DURATION	STATUS
{{ end -}}
{{- end -}}
{{- range $_, $cr := .CustomRuns.Items -}}{{- if $cr -}}{{- if $.AllNamespaces -}}
{{ $cr.Namespace }}	{{ $cr.Name }}	{{ formatAge $cr.Status.StartTime $.Time }}	{{ formatDuration $cr.Status.StartTime $cr.Status.CompletionTime }}	{{ formatCondition $cr.Status.Conditions }}
{{ else -}}
{{ $cr.Name }}	{{ formatAge $cr.Status.StartTime $.Time }}	{{ formatDuration $cr.Status.StartTime $cr.Status.CompletionTime }}	{{ formatCondition $cr.Status.Conditions }}
{{ end -}}{{- end -}}{{- end -}}
{{- end -}}
`

type ListOptions struct {
	Limit         int
	LabelSelector string
	Reverse       bool
	AllNamespaces bool
	NoHeaders     bool
}

func listCommand(p cli.Params) *cobra.Command {

	opts := &ListOptions{Limit: 0}
	f := cliopts.NewPrintFlags("list")
	eg := `List all CustomRuns in namespace 'bar':

    tkn cr list -n bar
`

	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists CustomRuns in a namespace",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Example: eg,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.Limit < 0 {
				return fmt.Errorf("limit was %d but must be a positive number", opts.Limit)
			}

			crs, err := list(p, opts.Limit, opts.LabelSelector, opts.AllNamespaces)
			if err != nil {
				return fmt.Errorf("failed to list CustomRuns from namespace %s: %v", p.Namespace(), err)
			}

			if crs != nil && opts.Reverse {
				reverse(crs)
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}
			if output == "name" && crs != nil {
				w := cmd.OutOrStdout()
				for _, tr := range crs.Items {
					_, err := fmt.Fprintf(w, "customrun.tekton.dev/%s\n", tr.Name)
					if err != nil {
						return err
					}
				}
				return nil
			} else if output != "" && crs != nil {
				p, err := f.ToPrinter()
				if err != nil {
					return err
				}
				return p.PrintObj(crs, cmd.OutOrStdout())
			}

			stream := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if crs != nil {
				err = printFormatted(stream, crs, p.Time(), opts.AllNamespaces, opts.NoHeaders)
			}

			if err != nil {
				fmt.Fprint(os.Stderr, "failed to print CustomRuns \n")
				return err
			}

			return nil
		},
	}

	f.AddFlags(c)
	c.Flags().IntVarP(&opts.Limit, "limit", "", 0, "Limits the number of CustomRuns. If the limit value is 0 returns all")
	c.Flags().StringVarP(&opts.LabelSelector, "label", "", opts.LabelSelector, "A selector (label query) to filter on, supports '=', '==', and '!='")
	c.Flags().BoolVarP(&opts.Reverse, "reverse", "", opts.Reverse, "list CustomRuns in reverse order")
	c.Flags().BoolVarP(&opts.AllNamespaces, "all-namespaces", "A", opts.AllNamespaces, "list CustomRuns from all namespaces")
	c.Flags().BoolVarP(&opts.NoHeaders, "no-headers", "", opts.NoHeaders, "do not print column headers with output (default print column headers with output)")
	return c
}

func reverse(crs *v1beta1.CustomRunList) {
	i := 0
	j := len(crs.Items) - 1
	crItems := crs.Items
	for i < j {
		crItems[i], crItems[j] = crItems[j], crItems[i]
		i++
		j--
	}
	crs.Items = crItems
}

func list(p cli.Params, limit int, labelselector string, allnamespaces bool) (*v1beta1.CustomRunList, error) {
	var selector string
	var options metav1.ListOptions

	if labelselector != "" {
		selector = labelselector
	}

	if selector != "" {
		options = metav1.ListOptions{
			LabelSelector: selector,
		}
	}

	cs, err := p.Clients()
	if err != nil {
		return nil, err
	}

	ns := p.Namespace()
	if allnamespaces {
		ns = ""
	}

	var crs *v1beta1.CustomRunList
	if err := actions.ListV1(customrunGroupResource, cs, options, ns, &crs); err != nil {
		return nil, err
	}

	crslen := len(crs.Items)

	if crslen != 0 {
		if allnamespaces {
			crsort.SortByNamespace(crs.Items)
		} else {
			crsort.SortByStartTime(crs.Items)
		}
	}

	// If greater than maximum amount of CustomRuns, return all CustomRuns by setting limit to default
	if limit > crslen {
		limit = 0
	}

	// Return all CustomRuns if limit is 0 or is same as crslen
	if limit != 0 && crslen > limit {
		crs.Items = crs.Items[0:limit]
	}

	return crs, nil
}

func printFormatted(s *cli.Stream, crs *v1beta1.CustomRunList, c clockwork.Clock, allnamespaces bool, noheaders bool) error {

	var data = struct {
		CustomRuns    *v1beta1.CustomRunList
		Time          clockwork.Clock
		AllNamespaces bool
		NoHeaders     bool
	}{
		CustomRuns:    crs,
		Time:          c,
		AllNamespaces: allnamespaces,
		NoHeaders:     noheaders,
	}

	funcMap := template.FuncMap{
		"formatAge":       formatted.Age,
		"formatDuration":  formatted.Duration,
		"formatCondition": formatted.Condition,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("List CustomRuns").Funcs(funcMap).Parse(ListTemplate))

	err := t.Execute(w, data)
	if err != nil {
		return err
	}

	return w.Flush()
}
