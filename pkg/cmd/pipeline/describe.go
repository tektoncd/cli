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
	"io"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	pipelinepkg "github.com/tektoncd/cli/pkg/pipeline"
	prsort "github.com/tektoncd/cli/pkg/pipelinerun/sort"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const DescribeTemplate = `{{decorate "bold" "Name"}}:	{{ .PipelineName }}
{{decorate "bold" "Namespace"}}:	{{ .Pipeline.Namespace }}
{{- if ne .Pipeline.Spec.Description "" }}
{{decorate "bold" "Description"}}:	{{ .Pipeline.Spec.Description }}
{{- end }}
{{- $annotations := removeLastAppliedConfig .Pipeline.Annotations -}}
{{- if $annotations }}
{{decorate "bold" "Annotations"}}:
{{- range $k, $v := $annotations }}
 {{ $k }}={{ $v }}
{{- end }}
{{- end }}

{{- if ne (len .Pipeline.Spec.Params) 0 }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}
 NAME	TYPE	DESCRIPTION	DEFAULT VALUE
{{- range $i, $p := .Pipeline.Spec.Params }}
{{- if not $p.Default }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ formatDesc $p.Description }}	{{ "---" }}
{{- else }}
{{- if eq $p.Type "string" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ formatDesc $p.Description }}	{{ $p.Default.StringVal }}
{{- else if eq $p.Type "array" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ formatDesc $p.Description }}	{{ $p.Default.ArrayVal }}
{{- else }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ formatDesc $p.Description }}	{{ $p.Default.ObjectVal }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{- if ne (len .Pipeline.Spec.Results) 0 }}

{{decorate "results" ""}}{{decorate "underline bold" "Results\n"}}
 NAME	DESCRIPTION
{{- range $result := .Pipeline.Spec.Results }}
 {{ decorate "bullet" $result.Name }}	{{ formatDesc $result.Description }}
{{- end }}
{{- end }}

{{- if ne (len .Pipeline.Spec.Workspaces) 0 }}

{{decorate "workspaces" ""}}{{decorate "underline bold" "Workspaces\n"}}
 NAME	DESCRIPTION	OPTIONAL
{{- range $workspace := .Pipeline.Spec.Workspaces }}
 {{ decorate "bullet" $workspace.Name }}	{{ formatDesc $workspace.Description }}	{{ $workspace.Optional }}
{{- end }}
{{- end }}

{{- if ne (len .Pipeline.Spec.Tasks) 0 }}

{{decorate "tasks" ""}}{{decorate "underline bold" "Tasks\n"}}
 NAME	TASKREF	RUNAFTER	TIMEOUT	PARAMS
{{- range $i, $t := .Pipeline.Spec.Tasks }}
 {{decorate "bullet" $t.Name }}	{{ getTaskRefName $t }}	{{ join $t.RunAfter ", " }}	{{ formatTimeout $t.Timeout }}	{{ formatParam $t.Params $.Pipeline.Spec.Params }}
{{- end }}
{{- end }}

{{- if ne (len .PipelineRuns.Items) 0 }}

{{decorate "pipelineruns" ""}}{{decorate "underline bold" "PipelineRuns\n"}}
 NAME	STARTED	DURATION	STATUS
{{- range $i, $pr := .PipelineRuns.Items }}
 {{decorate "bullet" $pr.Name }}	{{ formatAge $pr.Status.StartTime $.Time }}	{{ formatDuration $pr.Status.StartTime $pr.Status.CompletionTime }}	{{ formatCondition $pr.Status.Conditions }}
{{- end }}
{{- end }}
`

func describeCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("describe")
	opts := &options.DescribeOptions{Params: p}

	c := &cobra.Command{
		Use:     "describe",
		Aliases: []string{"desc"},
		Short:   "Describes a Pipeline in a namespace",
		Annotations: map[string]string{
			"commandType": "main",
		},
		SilenceUsage:      true,
		ValidArgsFunction: formatted.ParentCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}

			cs, err := p.Clients()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				pipelineNames, err := pipelinepkg.GetAllPipelineNames(pipelineGroupResource, cs, p.Namespace())
				if err != nil {
					return err
				}
				if len(pipelineNames) == 1 {
					opts.PipelineName = pipelineNames[0]
				} else {
					err = askPipelineName(opts, pipelineNames)
					if err != nil {
						return err
					}
				}
			} else {
				opts.PipelineName = args[0]
			}

			if output != "" {
				printer, err := f.ToPrinter()
				if err != nil {
					return err
				}
				return actions.PrintObjectV1(pipelineGroupResource, opts.PipelineName, cmd.OutOrStdout(), cs, printer, p.Namespace())
			}

			return printPipelineDescription(cmd.OutOrStdout(), cs, p.Namespace(), opts.PipelineName, p.Time())
		},
	}

	f.AddFlags(c)
	return c
}

func printPipelineDescription(out io.Writer, c *cli.Clients, ns string, pName string, time clockwork.Clock) error {
	pipeline, err := pipelinepkg.GetPipeline(pipelineGroupResource, c, pName, ns)
	if err != nil {
		return err
	}

	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/pipeline=%s", pName),
	}

	var pipelineRuns *v1.PipelineRunList
	err = actions.ListV1(pipelineRunGroupResource, c, opts, ns, &pipelineRuns)
	if err != nil {
		return err
	}
	prsort.SortByStartTime(pipelineRuns.Items)

	var data = struct {
		Pipeline     *v1.Pipeline
		PipelineRuns *v1.PipelineRunList
		PipelineName string
		Time         clockwork.Clock
	}{
		Pipeline:     pipeline,
		PipelineRuns: pipelineRuns,
		PipelineName: pName,
		Time:         time,
	}

	funcMap := template.FuncMap{
		"formatAge":               formatted.Age,
		"formatDuration":          formatted.Duration,
		"formatCondition":         formatted.Condition,
		"decorate":                formatted.DecorateAttr,
		"formatDesc":              formatted.FormatDesc,
		"formatTimeout":           formatted.Timeout,
		"formatParam":             formatted.Param,
		"join":                    strings.Join,
		"getTaskRefName":          formatted.GetTaskRefName,
		"removeLastAppliedConfig": formatted.RemoveLastAppliedConfig,
	}

	w := tabwriter.NewWriter(out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Describe Pipeline").Funcs(funcMap).Parse(DescribeTemplate))
	err = t.Execute(w, data)
	if err != nil {
		return err
	}

	return w.Flush()
}

func askPipelineName(opts *options.DescribeOptions, pipelineNames []string) error {
	if len(pipelineNames) == 0 {
		return fmt.Errorf("no Pipelines found")
	}

	err := opts.Ask(options.ResourceNamePipeline, pipelineNames)
	if err != nil {
		return err
	}

	return nil
}
