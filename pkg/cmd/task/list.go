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
	"text/tabwriter"
	"text/template"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const listTemplate = `{{- $tl := len .Tasks.Items }}{{ if eq $tl 0 -}}
No Tasks found
{{ else -}}
{{- if not $.NoHeaders -}}
{{- if $.AllNamespaces -}}
NAMESPACE	NAME	DESCRIPTION	AGE
{{ else -}}
NAME	DESCRIPTION	AGE
{{ end -}}
{{- end -}}
{{- range $_, $t := .Tasks.Items }}{{- if $t }}
{{- if $.AllNamespaces -}}
{{ $t.Namespace }}	{{ $t.Name }}	{{ formatDesc $t.Spec.Description }}	{{ formatAge $t.CreationTimestamp $.Time }}
{{ else -}}
{{ $t.Name }}	{{ formatDesc $t.Spec.Description }}	{{ formatAge $t.CreationTimestamp $.Time }}
{{ end }}{{- end }}{{- end }}
{{- end -}}
`

type ListOptions struct {
	AllNamespaces bool
	NoHeaders     bool
}

func listCommand(p cli.Params) *cobra.Command {
	opts := &ListOptions{}
	f := cliopts.NewPrintFlags("list")

	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists Tasks in a namespace",
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cs, err := p.Clients()
			if err != nil {
				return err
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("error: output option not set properly: %v", err)
			}

			ns := p.Namespace()
			if opts.AllNamespaces {
				ns = ""
			}

			if output != "" {
				p, err := f.ToPrinter()
				if err != nil {
					return err
				}
				return actions.PrintObjects(taskGroupResource, cmd.OutOrStdout(), cs.Dynamic, cs.Tekton.Discovery(), p, ns)
			}
			stream := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}
			return printTaskDetails(stream, p, opts.AllNamespaces, opts.NoHeaders)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.AllNamespaces, "all-namespaces", "A", opts.AllNamespaces, "list Tasks from all namespaces")
	c.Flags().BoolVarP(&opts.NoHeaders, "no-headers", "", opts.NoHeaders, "do not print column headers with output (default print column headers with output)")

	return c
}

func printTaskDetails(s *cli.Stream, p cli.Params, allnamespaces bool, noheaders bool) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	ns := p.Namespace()
	if allnamespaces {
		ns = ""
	}

	var tasks *v1.TaskList
	if err := actions.ListV1(taskGroupResource, cs, metav1.ListOptions{}, ns, &tasks); err != nil {
		return fmt.Errorf("failed to list Tasks from namespace %s: %v", ns, err)
	}

	var data = struct {
		Tasks         *v1.TaskList
		Time          clockwork.Clock
		AllNamespaces bool
		NoHeaders     bool
	}{
		Tasks:         tasks,
		Time:          p.Time(),
		AllNamespaces: allnamespaces,
		NoHeaders:     noheaders,
	}

	funcMap := template.FuncMap{
		"formatAge":  formatted.Age,
		"formatDesc": formatted.FormatDesc,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("List Tasks").Funcs(funcMap).Parse(listTemplate))

	err = t.Execute(w, data)
	if err != nil {
		return err
	}

	return w.Flush()
}
