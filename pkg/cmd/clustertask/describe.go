// Copyright © 2020 The Tekton Authors.
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
	"text/template"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	clustertaskpkg "github.com/tektoncd/cli/pkg/clustertask"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	trsort "github.com/tektoncd/cli/pkg/taskrun/sort"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const describeTemplate = `{{decorate "bold" "Name"}}:	{{ .ClusterTask.Name }}
{{- if ne .ClusterTask.Spec.Description "" }}
{{decorate "bold" "Description"}}:	{{ .ClusterTask.Spec.Description }}
{{- end }}
{{- $annotations := removeLastAppliedConfig .ClusterTask.Annotations -}}
{{- if $annotations }}
{{decorate "bold" "Annotations"}}:
{{- range $k, $v := $annotations }}
 {{ $k }}={{ $v }}
{{- end }}
{{- end }}

{{- if ne (len .ClusterTask.Spec.Params) 0 }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}
 NAME	TYPE	DESCRIPTION	DEFAULT VALUE
{{- range $p := .ClusterTask.Spec.Params }}
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

{{- if ne (len .ClusterTask.Spec.Results) 0 }}

{{decorate "results" ""}}{{decorate "underline bold" "Results\n"}}
 NAME	DESCRIPTION
{{- range $result := .ClusterTask.Spec.Results }}
 {{ decorate "bullet" $result.Name }}	{{ formatDesc $result.Description }}
{{- end }}
{{- end }}

{{- if ne (len .ClusterTask.Spec.Workspaces) 0 }}

{{decorate "workspaces" ""}}{{decorate "underline bold" "Workspaces\n"}}
 NAME	DESCRIPTION
{{- range $workspace := .ClusterTask.Spec.Workspaces }}
 {{ decorate "bullet" $workspace.Name }}	{{ formatDesc $workspace.Description }}
{{- end }}
{{- end }}

{{- if ne (len .ClusterTask.Spec.Steps) 0 }}

{{decorate "steps" ""}}{{decorate "underline bold" "Steps\n"}}
{{- range $step := .ClusterTask.Spec.Steps }}
 {{ autoStepName $step.Name | decorate "bullet" }}
{{- end }}
{{- end }}

{{- if ne (len .TaskRuns.Items) 0 }}

{{decorate "taskruns" ""}}{{decorate "underline bold" "Taskruns\n"}}
NAME	STARTED	DURATION	STATUS
{{ range $tr:=.TaskRuns.Items }}
{{- $tr.Name }}	{{ formatAge $tr.Status.StartTime $.Time}}	{{ formatDuration $tr.Status.StartTime $tr.Status.CompletionTime }}	{{ formatCondition $tr.Status.Conditions }}
{{ end }}
{{- end }}
`

func describeCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("describe")
	opts := &options.DescribeOptions{Params: p}
	eg := `Describe a ClusterTask of name 'foo':

    tkn clustertask describe foo

or

    tkn ct desc foo
`

	c := &cobra.Command{
		Use:     "describe",
		Aliases: []string{"desc"},
		Short:   "Describe a ClusterTask",
		Example: eg,
		Annotations: map[string]string{
			"commandType": "main",
		},
		SilenceUsage:      true,
		ValidArgsFunction: formatted.ParentCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}

			cs, err := p.Clients()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				clusterTaskNames, err := clustertaskpkg.GetAllClusterTaskNames(clustertaskGroupResource, cs)
				if err != nil {
					return err
				}
				if len(clusterTaskNames) == 1 {
					opts.ClusterTaskName = clusterTaskNames[0]
				} else {
					err = askClusterTaskName(opts, clusterTaskNames)
					if err != nil {
						return err
					}
				}
			} else {
				opts.ClusterTaskName = args[0]
			}

			if output != "" {
				p, err := f.ToPrinter()
				if err != nil {
					return err
				}
				return actions.PrintObjectV1(clustertaskGroupResource, opts.ClusterTaskName, cmd.OutOrStdout(), cs, p, "")
			}

			return printClusterTaskDescription(s, cs, opts.ClusterTaskName, p.Namespace(), p.Time())
		},
	}

	f.AddFlags(c)
	c.Deprecated = "ClusterTasks are deprecated, this command will be removed in future releases."
	return c
}

func printClusterTaskDescription(s *cli.Stream, cs *cli.Clients, ctname, namespace string, time clockwork.Clock) error {
	var clustertask *v1beta1.ClusterTask
	err := actions.GetV1(clustertaskGroupResource, cs, ctname, "", metav1.GetOptions{}, &clustertask)
	if err != nil {
		return fmt.Errorf("failed to get ClusterTask %s", ctname)
	}

	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/clusterTask=%s", ctname),
	}

	var taskRuns *v1.TaskRunList
	if err := actions.ListV1(taskrunGroupResource, cs, opts, namespace, &taskRuns); err != nil {
		return fmt.Errorf("failed to get TaskRuns for ClusterTask %s", ctname)
	}

	trsort.SortByStartTime(taskRuns.Items)

	var data = struct {
		ClusterTask *v1beta1.ClusterTask
		TaskRuns    *v1.TaskRunList
		Time        clockwork.Clock
	}{
		ClusterTask: clustertask,
		TaskRuns:    taskRuns,
		Time:        time,
	}

	funcMap := template.FuncMap{
		"formatAge":               formatted.Age,
		"formatDuration":          formatted.Duration,
		"formatCondition":         formatted.Condition,
		"decorate":                formatted.DecorateAttr,
		"autoStepName":            formatted.AutoStepName,
		"formatDesc":              formatted.FormatDesc,
		"removeLastAppliedConfig": formatted.RemoveLastAppliedConfig,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Describe ClusterTask").Funcs(funcMap).Parse(describeTemplate))
	err = t.Execute(w, data)
	if err != nil {
		return fmt.Errorf("failed to execute template")
	}
	return w.Flush()
}

func askClusterTaskName(opts *options.DescribeOptions, clusterTaskNames []string) error {
	if len(clusterTaskNames) == 0 {
		return fmt.Errorf("no ClusterTasks found")
	}
	err := opts.Ask(options.ResourceNameClusterTask, clusterTaskNames)
	if err != nil {
		return err
	}

	return nil
}
