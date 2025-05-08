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
	"context"
	"fmt"
	"text/tabwriter"
	"text/template"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/task"
	trsort "github.com/tektoncd/cli/pkg/taskrun/sort"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const describeTemplate = `{{decorate "bold" "Name"}}:	{{ .Task.Name }}
{{decorate "bold" "Namespace"}}:	{{ .Task.Namespace }}
{{- if ne .Task.Spec.Description "" }}
{{decorate "bold" "Description"}}:	{{ .Task.Spec.Description }}
{{- end }}
{{- $v := findVersion .Task.Labels }} {{- if ne $v ""}}
{{decorate "bold" "Version"}}:    	{{ $v }}
{{- end }}
{{- $annotations := removeLastAppliedConfig .Task.Annotations -}}
{{- if $annotations }}
{{decorate "bold" "Annotations"}}:
{{- range $k, $v := $annotations }}
 {{ $k }}={{ $v }}
{{- end }}
{{- end }}

{{- if ne (len .Task.Spec.Params) 0 }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}
 NAME	TYPE	DESCRIPTION	DEFAULT VALUE
{{- range $p := .Task.Spec.Params }}
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

{{- if ne (len .Task.Spec.Results) 0 }}

{{decorate "results" ""}}{{decorate "underline bold" "Results\n"}}
 NAME	DESCRIPTION
{{- range $result := .Task.Spec.Results }}
 {{ decorate "bullet" $result.Name }}	{{ formatDesc $result.Description }}
{{- end }}
{{- end }}

{{- if ne (len .Task.Spec.Workspaces) 0 }}

{{decorate "workspaces" ""}}{{decorate "underline bold" "Workspaces\n"}}
 NAME	DESCRIPTION	OPTIONAL
{{- range $workspace := .Task.Spec.Workspaces }}
 {{ decorate "bullet" $workspace.Name }}	{{ formatDesc $workspace.Description }}	{{ $workspace.Optional }}
{{- end }}
{{- end }}

{{- if ne (len .Task.Spec.Steps) 0 }}

{{decorate "steps" ""}}{{decorate "underline bold" "Steps\n"}}
{{- range $step := .Task.Spec.Steps }}
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
	eg := `Describe a Task of name 'foo' in namespace 'bar':

    tkn task describe foo -n bar

or

   tkn t desc foo -n bar
`

	c := &cobra.Command{
		Use:               "describe",
		Aliases:           []string{"desc"},
		ValidArgsFunction: formatted.ParentCompletion,
		Short:             "Describe a Task in a namespace",
		Example:           eg,
		Annotations: map[string]string{
			"commandType": "main",
		},
		SilenceUsage: true,
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
				taskNames, err := task.GetAllTaskNames(taskGroupResource, cs, p.Namespace())
				if err != nil {
					return err
				}
				if len(taskNames) == 1 {
					opts.TaskName = taskNames[0]
				} else {
					err = askTaskName(opts, cs, p.Namespace())
					if err != nil {
						return err
					}
				}
			} else {
				opts.TaskName = args[0]
			}

			if output != "" {
				taskGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "tasks"}
				printer, err := f.ToPrinter()
				if err != nil {
					return err
				}
				return actions.PrintObjectV1(taskGroupResource, opts.TaskName, cmd.OutOrStdout(), cs, printer, p.Namespace())
			}

			return printTaskDescription(s, p, opts.TaskName)
		},
	}

	f.AddFlags(c)
	return c
}

func printTaskDescription(s *cli.Stream, p cli.Params, tname string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	t, err := getTask(taskGroupResource, cs, tname, p.Namespace())
	if err != nil {
		return fmt.Errorf("failed to get Task %s: %v", tname, err)
	}

	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/task=%s", tname),
	}

	var taskRuns *v1.TaskRunList
	if err := actions.ListV1(taskrunGroupResource, cs, opts, p.Namespace(), &taskRuns); err != nil {
		return fmt.Errorf("failed to get TaskRuns for Task %s: %v", tname, err)
	}

	// this is required as the same label is getting added for task
	taskRuns.Items = task.FilterByRef(taskRuns.Items, "Task")

	trsort.SortByStartTime(taskRuns.Items)

	var data = struct {
		Task     *v1.Task
		TaskRuns *v1.TaskRunList
		Time     clockwork.Clock
	}{
		Task:     t,
		TaskRuns: taskRuns,
		Time:     p.Time(),
	}

	funcMap := template.FuncMap{
		"formatAge":               formatted.Age,
		"formatDuration":          formatted.Duration,
		"formatCondition":         formatted.Condition,
		"decorate":                formatted.DecorateAttr,
		"autoStepName":            formatted.AutoStepName,
		"formatDesc":              formatted.FormatDesc,
		"findVersion":             formatted.FindVersion,
		"removeLastAppliedConfig": formatted.RemoveLastAppliedConfig,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	tparsed := template.Must(template.New("Describe Task").Funcs(funcMap).Parse(describeTemplate))
	err = tparsed.Execute(w, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return w.Flush()
}

func askTaskName(opts *options.DescribeOptions, c *cli.Clients, ns string) error {
	taskNames, err := task.GetAllTaskNames(taskGroupResource, c, ns)
	if err != nil {
		return err
	}
	if len(taskNames) == 0 {
		return fmt.Errorf("no Tasks found")
	}

	err = opts.Ask(options.ResourceNameTask, taskNames)
	if err != nil {
		return err
	}

	return nil
}

func getTask(gr schema.GroupVersionResource, c *cli.Clients, tName, ns string) (*v1.Task, error) {
	var task v1.Task
	gvr, err := actions.GetGroupVersionResource(gr, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if gvr.Version == "v1" {
		err := actions.GetV1(taskGroupResource, c, tName, ns, metav1.GetOptions{}, &task)
		if err != nil {
			return nil, err
		}
		return &task, nil

	}

	var taskV1beta1 v1beta1.Task
	err = actions.GetV1(taskGroupResource, c, tName, ns, metav1.GetOptions{}, &taskV1beta1)
	if err != nil {
		return nil, err
	}
	err = taskV1beta1.ConvertTo(context.Background(), &task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}
