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
	"os"
	"sort"
	"text/tabwriter"
	"text/template"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/task"
	"github.com/tektoncd/cli/pkg/taskrun/list"
	trsort "github.com/tektoncd/cli/pkg/taskrun/sort"
	"github.com/tektoncd/cli/pkg/validate"
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

{{decorate "inputresources" ""}}{{decorate "underline bold" "Input Resources\n"}}

{{- if not .Task.Spec.Resources }}
 No input resources
{{- else }}
{{- if eq (len .Task.Spec.Resources.Inputs) 0 }}
 No input resources
{{- else }}
 NAME	TYPE
{{- range $ir := .Task.Spec.Resources.Inputs }}
 {{decorate "bullet" $ir.Name }}	{{ $ir.Type }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "outputresources" ""}}{{decorate "underline bold" "Output Resources\n"}}

{{- if not .Task.Spec.Resources }}
 No output resources
{{- else }}
{{- if eq (len .Task.Spec.Resources.Outputs) 0 }}
 No output resources
{{- else }}
 NAME	TYPE
 
{{- range $or := .Task.Spec.Resources.Outputs }}
 {{decorate "bullet" $or.Name }}	{{ $or.Type }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}

{{- if eq (len .Task.Spec.Params) 0 }}
 No params
{{- else }}
 NAME	TYPE	DESCRIPTION	DEFAULT VALUE
{{- range $p := .Task.Spec.Params }}
{{- if not $p.Default }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ formatDesc $p.Description }}	{{ "---" }}
{{- else }}
{{- if eq $p.Type "string" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ formatDesc $p.Description }}	{{ $p.Default.StringVal }}
{{- else }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ formatDesc $p.Description }}	{{ $p.Default.ArrayVal }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "steps" ""}}{{decorate "underline bold" "Steps\n"}}

{{- if eq (len .Task.Spec.Steps) 0 }}
 No steps
{{- else }}
{{- range $step := .Task.Spec.Steps }}
 {{ autoStepName $step.Name | decorate "bullet" }} 
{{- end }}
{{- end }}

{{decorate "taskruns" ""}}{{decorate "underline bold" "Taskruns\n"}}

{{- if eq (len .TaskRuns.Items) 0 }}
 No taskruns
{{- else }}
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
		Use:     "describe",
		Aliases: []string{"desc"},
		Short:   "Describes a task in a namespace",
		Example: eg,
		Annotations: map[string]string{
			"commandType": "main",
		},
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if err := validate.NamespaceExists(p); err != nil {
				return err
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				fmt.Fprint(os.Stderr, "Error: output option not set properly \n")
				return err
			}

			if len(args) == 0 {
				err = askTaskName(opts, p)
				if err != nil {
					return err
				}
			} else {
				opts.TaskName = args[0]
			}

			if output != "" {
				taskGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "tasks"}
				return actions.PrintObject(taskGroupResource, opts.TaskName, cmd.OutOrStdout(), p, f, p.Namespace())
			}

			return printTaskDescription(s, p, opts.TaskName)
		},
	}

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_task")
	f.AddFlags(c)
	return c
}

func printTaskDescription(s *cli.Stream, p cli.Params, tname string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	t, err := task.Get(cs, tname, metav1.GetOptions{}, p.Namespace())
	if err != nil {
		fmt.Fprintf(s.Err, "failed to get task %s\n", tname)
		return err
	}

	if t.Spec.Resources != nil {
		t.Spec.Resources.Inputs = sortResourcesByTypeAndName(t.Spec.Resources.Inputs)
		t.Spec.Resources.Outputs = sortResourcesByTypeAndName(t.Spec.Resources.Outputs)
	}

	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/task=%s", tname),
	}
	taskRuns, err := list.TaskRuns(cs, opts, p.Namespace())
	if err != nil {
		fmt.Fprintf(s.Err, "failed to get taskruns for task %s \n", tname)
		return err
	}

	// this is required as the same label is getting added for both task and ClusterTask
	taskRuns.Items = task.FilterByRef(taskRuns.Items, "Task")

	trsort.SortByStartTime(taskRuns.Items)

	var data = struct {
		Task     *v1beta1.Task
		TaskRuns *v1beta1.TaskRunList
		Time     clockwork.Clock
	}{
		Task:     t,
		TaskRuns: taskRuns,
		Time:     p.Time(),
	}

	funcMap := template.FuncMap{
		"formatAge":       formatted.Age,
		"formatDuration":  formatted.Duration,
		"formatCondition": formatted.Condition,
		"decorate":        formatted.DecorateAttr,
		"autoStepName":    formatted.AutoStepName,
		"formatDesc":      formatted.FormatDesc,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	tparsed := template.Must(template.New("Describe Task").Funcs(funcMap).Parse(describeTemplate))
	err = tparsed.Execute(w, data)
	if err != nil {
		fmt.Fprintf(s.Err, "Failed to execute template \n")
		return err
	}
	return nil
}

// this will sort the Task Resource by Type and then by Name
func sortResourcesByTypeAndName(tres []v1beta1.TaskResource) []v1beta1.TaskResource {
	sort.Slice(tres, func(i, j int) bool {
		if tres[j].Type < tres[i].Type {
			return false
		}

		if tres[j].Type > tres[i].Type {
			return true
		}

		return tres[j].Name > tres[i].Name
	})

	return tres
}

func askTaskName(opts *options.DescribeOptions, p cli.Params) error {
	taskNames, err := task.GetAllTaskNames(p)
	if err != nil {
		return err
	}
	if len(taskNames) == 0 {
		return fmt.Errorf("No tasks found")
	}

	err = opts.Ask(options.ResourceNameTask, taskNames)
	if err != nil {
		return err
	}

	return nil
}
