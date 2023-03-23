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
	"fmt"
	"os"
	"text/tabwriter"
	"text/template"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/printer"
	taskpkg "github.com/tektoncd/cli/pkg/task"
	trsort "github.com/tektoncd/cli/pkg/taskrun/sort"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const ListTemplate = `{{- $trl := len .TaskRuns.Items -}}{{- if eq $trl 0 -}}
No TaskRuns found
{{ else -}}
{{- if not $.NoHeaders -}}
{{- if $.AllNamespaces -}}
NAMESPACE	NAME	STARTED	DURATION	STATUS
{{ else -}}
NAME	STARTED	DURATION	STATUS
{{ end -}}
{{- end -}}
{{- range $_, $tr := .TaskRuns.Items -}}{{- if $tr -}}{{- if $.AllNamespaces -}}
{{ $tr.Namespace }}	{{ $tr.Name }}	{{ formatAge $tr.Status.StartTime $.Time }}	{{ formatDuration $tr.Status.StartTime $tr.Status.CompletionTime }}	{{ formatCondition $tr.Status.Conditions }}
{{ else -}}
{{ $tr.Name }}	{{ formatAge $tr.Status.StartTime $.Time }}	{{ formatDuration $tr.Status.StartTime $tr.Status.CompletionTime }}	{{ formatCondition $tr.Status.Conditions }}
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
	eg := `List all TaskRuns in namespace 'bar':

    tkn tr list -n bar

List all TaskRuns of Task 'foo' in namespace 'bar':

    tkn taskrun list foo -n bar
`

	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists TaskRuns in a namespace",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Example: eg,
		RunE: func(cmd *cobra.Command, args []string) error {
			var task string
			if len(args) > 0 {
				task = args[0]
			}

			if opts.Limit < 0 {
				return fmt.Errorf("limit was %d but must be a positive number", opts.Limit)
			}

			trs, err := list(p, task, opts.Limit, opts.LabelSelector, opts.AllNamespaces)
			if err != nil {
				return fmt.Errorf("failed to list TaskRuns from namespace %s: %v", p.Namespace(), err)
			}

			if trs != nil && opts.Reverse {
				reverse(trs)
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}
			if output == "name" && trs != nil {
				w := cmd.OutOrStdout()
				for _, tr := range trs.Items {
					_, err := fmt.Fprintf(w, "taskrun.tekton.dev/%s\n", tr.Name)
					if err != nil {
						return err
					}
				}
				return nil
			} else if output != "" && trs != nil {
				return printer.PrintObject(cmd.OutOrStdout(), trs, f)
			}

			stream := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if trs != nil {
				err = printFormatted(stream, trs, p.Time(), opts.AllNamespaces, opts.NoHeaders)
			}

			if err != nil {
				fmt.Fprint(os.Stderr, "failed to print TaskRuns \n")
				return err
			}

			return nil
		},
	}

	f.AddFlags(c)
	c.Flags().IntVarP(&opts.Limit, "limit", "", 0, "limit TaskRuns listed (default: return all TaskRuns)")
	c.Flags().StringVarP(&opts.LabelSelector, "label", "", opts.LabelSelector, "A selector (label query) to filter on, supports '=', '==', and '!='")
	c.Flags().BoolVarP(&opts.Reverse, "reverse", "", opts.Reverse, "list TaskRuns in reverse order")
	c.Flags().BoolVarP(&opts.AllNamespaces, "all-namespaces", "A", opts.AllNamespaces, "list TaskRuns from all namespaces")
	c.Flags().BoolVarP(&opts.NoHeaders, "no-headers", "", opts.NoHeaders, "do not print column headers with output (default print column headers with output)")
	return c
}

func reverse(trs *v1.TaskRunList) {
	i := 0
	j := len(trs.Items) - 1
	trItems := trs.Items
	for i < j {
		trItems[i], trItems[j] = trItems[j], trItems[i]
		i++
		j--
	}
	trs.Items = trItems
}

func list(p cli.Params, task string, limit int, labelselector string, allnamespaces bool) (*v1.TaskRunList, error) {
	var selector string
	var options metav1.ListOptions

	if task != "" && labelselector != "" {
		return nil, fmt.Errorf("specifying a Task and labels are not compatible")
	}

	if task != "" {
		selector = fmt.Sprintf("tekton.dev/task=%s", task)
	} else if labelselector != "" {
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

	var trs *v1.TaskRunList
	if err := actions.ListV1(taskrunGroupResource, cs, options, ns, &trs); err != nil {
		return nil, err
	}

	// this is required as the same label is getting added for both task and ClusterTask
	if task != "" {
		trs.Items = taskpkg.FilterByRef(trs.Items, string(v1.NamespacedTaskKind))
	}

	trslen := len(trs.Items)

	if trslen != 0 {
		if allnamespaces {
			trsort.SortByNamespace(trs.Items)
		} else {
			trsort.SortByStartTime(trs.Items)
		}
	}

	// If greater than maximum amount of TaskRuns, return all TaskRuns by setting limit to default
	if limit > trslen {
		limit = 0
	}

	// Return all TaskRuns if limit is 0 or is same as trslen
	if limit != 0 && trslen > limit {
		trs.Items = trs.Items[0:limit]
	}

	return trs, nil
}

func printFormatted(s *cli.Stream, trs *v1.TaskRunList, c clockwork.Clock, allnamespaces bool, noheaders bool) error {

	var data = struct {
		TaskRuns      *v1.TaskRunList
		Time          clockwork.Clock
		AllNamespaces bool
		NoHeaders     bool
	}{
		TaskRuns:      trs,
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
	t := template.Must(template.New("List TaskRuns").Funcs(funcMap).Parse(ListTemplate))

	err := t.Execute(w, data)
	if err != nil {
		return err
	}

	return w.Flush()
}
