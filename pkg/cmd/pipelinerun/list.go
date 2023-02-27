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

package pipelinerun

import (
	"fmt"
	"text/tabwriter"
	"text/template"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	prsort "github.com/tektoncd/cli/pkg/pipelinerun/sort"
	"github.com/tektoncd/cli/pkg/printer"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const listTemplate = `{{- $prl := len .PipelineRuns.Items -}}{{- if eq $prl 0 -}}
No PipelineRuns found
{{ else -}}
{{- if not $.NoHeaders -}}
{{- if $.AllNamespaces -}}
NAMESPACE	NAME	STARTED	DURATION	STATUS
{{ else -}}
NAME	STARTED	DURATION	STATUS
{{ end -}}
{{- end -}}
{{- range $_, $pr := .PipelineRuns.Items }}{{- if $pr }}{{- if $.AllNamespaces -}}
{{ $pr.Namespace }}	{{ $pr.Name }}	{{ formatAge $pr.Status.StartTime $.Time }}	{{ formatDuration $pr.Status.StartTime $pr.Status.CompletionTime }}	{{ formatCondition $pr.Status.Conditions }}
{{ else -}}
{{ $pr.Name }}	{{ formatAge $pr.Status.StartTime $.Time }}	{{ formatDuration $pr.Status.StartTime $pr.Status.CompletionTime }}	{{ formatCondition $pr.Status.Conditions }}
{{ end -}}{{- end -}}{{- end -}}
{{- end -}}`

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
	eg := `List all PipelineRuns of Pipeline 'foo':

    tkn pipelinerun list foo -n bar

List all PipelineRuns in a namespace 'foo':

    tkn pr list -n foo
`

	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists PipelineRuns in a namespace",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Example: eg,
		RunE: func(cmd *cobra.Command, args []string) error {
			var pipeline string

			if len(args) > 0 {
				pipeline = args[0]
			}

			if opts.Limit < 0 {
				return fmt.Errorf("limit was %d but must be a positive number", opts.Limit)
			}

			prs, err := list(p, pipeline, opts.Limit, opts.LabelSelector, opts.AllNamespaces)
			if err != nil {
				return fmt.Errorf("failed to list PipelineRuns from namespace %s: %v", p.Namespace(), err)
			}

			if prs != nil && opts.Reverse {
				reverse(prs)
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}

			if output == "name" && prs != nil {
				w := cmd.OutOrStdout()
				for _, pr := range prs.Items {
					_, err := fmt.Fprintf(w, "pipelinerun.tekton.dev/%s\n", pr.Name)
					if err != nil {
						return err
					}
				}
				return nil
			} else if output != "" && prs != nil {
				return printer.PrintObject(cmd.OutOrStdout(), prs, f)
			}
			stream := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if prs != nil {
				err = printFormatted(stream, prs, p.Time(), opts.AllNamespaces, opts.NoHeaders)
			}
			if err != nil {
				return fmt.Errorf("failed to print PipelineRuns: %v", err)
			}

			return nil
		},
	}

	f.AddFlags(c)
	c.Flags().IntVarP(&opts.Limit, "limit", "", 0, "limit PipelineRuns listed (default: return all PipelineRuns)")
	c.Flags().StringVarP(&opts.LabelSelector, "label", "", opts.LabelSelector, "A selector (label query) to filter on, supports '=', '==', and '!='")
	c.Flags().BoolVarP(&opts.Reverse, "reverse", "", opts.Reverse, "list PipelineRuns in reverse order")
	c.Flags().BoolVarP(&opts.AllNamespaces, "all-namespaces", "A", opts.AllNamespaces, "list PipelineRuns from all namespaces")
	c.Flags().BoolVarP(&opts.NoHeaders, "no-headers", "", opts.NoHeaders, "do not print column headers with output (default print column headers with output)")
	return c
}

func list(p cli.Params, pipeline string, limit int, labelselector string, allnamespaces bool) (*v1.PipelineRunList, error) {
	var selector string
	var options metav1.ListOptions

	cs, err := p.Clients()
	if err != nil {
		return nil, err
	}

	if pipeline != "" && labelselector != "" {
		return nil, fmt.Errorf("specifying a Pipeline and labels are not compatible")
	}

	if pipeline != "" {
		selector = fmt.Sprintf("tekton.dev/pipeline=%s", pipeline)
	} else if labelselector != "" {
		selector = labelselector
	}

	if selector != "" {
		options = metav1.ListOptions{
			LabelSelector: selector,
		}
	}

	ns := p.Namespace()
	if allnamespaces {
		ns = ""
	}
	var pipelineRuns *v1.PipelineRunList
	if err := actions.ListV1(pipelineRunGroupResource, cs, options, ns, &pipelineRuns); err != nil {
		return nil, err
	}

	prslen := len(pipelineRuns.Items)

	if prslen != 0 {
		if allnamespaces {
			prsort.SortByNamespace(pipelineRuns.Items)
		} else {
			prsort.SortByStartTime(pipelineRuns.Items)
		}
	}

	// If greater than maximum amount of pipelineruns, return all pipelineruns by setting limit to default
	if limit > prslen {
		limit = 0
	}

	// Return all pipelineruns if limit is 0 or is same as prslen
	if limit != 0 && prslen > limit {
		pipelineRuns.Items = pipelineRuns.Items[0:limit]
	}

	return pipelineRuns, nil
}

func reverse(prs *v1.PipelineRunList) {
	i := 0
	j := len(prs.Items) - 1
	prItems := prs.Items
	for i < j {
		prItems[i], prItems[j] = prItems[j], prItems[i]
		i++
		j--
	}
	prs.Items = prItems
}

func printFormatted(s *cli.Stream, prs *v1.PipelineRunList, c clockwork.Clock, allnamespaces bool, noheaders bool) error {
	var data = struct {
		PipelineRuns  *v1.PipelineRunList
		Time          clockwork.Clock
		AllNamespaces bool
		NoHeaders     bool
	}{
		PipelineRuns:  prs,
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
	t := template.Must(template.New("List PipelineRuns").Funcs(funcMap).Parse(listTemplate))

	err := t.Execute(w, data)
	if err != nil {
		return err
	}

	return w.Flush()
}
