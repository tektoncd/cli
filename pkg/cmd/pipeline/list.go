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

package pipeline

import (
	"fmt"
	"text/tabwriter"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/pipeline"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const listTemplate = `{{- $pl := len .Pipelines.Items }}{{ if eq $pl 0 -}}
No Pipelines found
{{ else -}}
{{- if not $.NoHeaders -}}
{{- if $.AllNamespaces -}}
NAMESPACE	NAME	AGE	LAST RUN	STARTED	DURATION	STATUS
{{ else -}}
NAME	AGE	LAST RUN	STARTED	DURATION	STATUS
{{ end }}
{{- end -}}
{{- range $_, $p := .Pipelines.Items }}
{{- $pr := accessMap $.PipelineRuns $p.Name }}
{{- if $pr }}
{{- if $.AllNamespaces -}}
{{ $p.Namespace }}	{{ $p.Name }}	{{ formatAge $p.CreationTimestamp $.Params.Time }}	{{ $pr.Name }}	{{ formatAge $pr.Status.StartTime $.Params.Time }}	{{ formatDuration $pr.Status.StartTime $pr.Status.CompletionTime }}	{{ formatCondition $pr.Status.Conditions }}
{{ else -}}
{{ $p.Name }}	{{ formatAge $p.CreationTimestamp $.Params.Time }}	{{ $pr.Name }}	{{ formatAge $pr.Status.StartTime $.Params.Time }}	{{ formatDuration $pr.Status.StartTime $pr.Status.CompletionTime }}	{{ formatCondition $pr.Status.Conditions }}
{{ end }}
{{- else }}
{{- if $.AllNamespaces -}}
{{ $p.Namespace }}	{{ $p.Name }}	{{ formatAge $p.CreationTimestamp $.Params.Time }}	---	---	---	---
{{ else -}}
{{ $p.Name }}	{{ formatAge $p.CreationTimestamp $.Params.Time }}	---	---	---	---
{{ end }}
{{- end }}
{{- end }}
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
		Short:   "Lists Pipelines in a namespace",
		Annotations: map[string]string{
			"commandType": "main",
		},
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cs, err := p.Clients()
			if err != nil {
				return err
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
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
				return actions.PrintObjects(pipelineGroupResource, cmd.OutOrStdout(), cs.Dynamic, cs.Tekton.Discovery(), p, ns)
			}
			stream := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}
			return printPipelineDetails(stream, p, opts.AllNamespaces, opts.NoHeaders)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.AllNamespaces, "all-namespaces", "A", opts.AllNamespaces, "list Pipelines from all namespaces")
	c.Flags().BoolVarP(&opts.NoHeaders, "no-headers", "", opts.NoHeaders, "do not print column headers with output (default print column headers with output)")

	return c
}

func printPipelineDetails(s *cli.Stream, p cli.Params, allnamespaces bool, noheaders bool) error {

	cs, err := p.Clients()
	if err != nil {
		return err
	}

	ns := p.Namespace()
	if allnamespaces {
		ns = ""
	}
	ps, prs, err := listPipelineDetails(cs, ns)
	if err != nil {
		return fmt.Errorf("failed to list Pipelines from namespace %s: %v", ns, err)
	}

	var data = struct {
		Pipelines     *v1.PipelineList
		PipelineRuns  pipelineruns
		Params        cli.Params
		AllNamespaces bool
		NoHeaders     bool
	}{
		Pipelines:     ps,
		PipelineRuns:  prs,
		Params:        p,
		AllNamespaces: allnamespaces,
		NoHeaders:     noheaders,
	}

	funcMap := template.FuncMap{
		"accessMap": func(prs pipelineruns, name string) *v1.PipelineRun {
			if pr, ok := prs[name]; ok {
				return &pr
			}
			return nil
		},
		"formatAge":       formatted.Age,
		"formatDuration":  formatted.Duration,
		"formatCondition": formatted.Condition,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("List Pipeline").Funcs(funcMap).Parse(listTemplate))
	err = t.Execute(w, data)
	if err != nil {
		return err
	}

	return w.Flush()
}

type pipelineruns map[string]v1.PipelineRun

func listPipelineDetails(cs *cli.Clients, ns string) (*v1.PipelineList, pipelineruns, error) {
	var pipelines *v1.PipelineList
	if err := actions.ListV1(pipelineGroupResource, cs, metav1.ListOptions{}, ns, &pipelines); err != nil {
		return nil, nil, err
	}

	if len(pipelines.Items) == 0 {
		return pipelines, pipelineruns{}, nil
	}
	lastRuns := pipelineruns{}

	for _, p := range pipelines.Items {
		// TODO: may be just the pipeline details can be print
		lastRun, err := pipeline.LastRun(cs, p.Name, ns)
		if err != nil {
			continue
		}
		lastRuns[p.Name] = *lastRun
	}

	return pipelines, lastRuns, nil
}
