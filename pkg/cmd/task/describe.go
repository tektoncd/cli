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
	"io"
	"os"
	"sort"
	"text/tabwriter"
	"text/template"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	validate "github.com/tektoncd/cli/pkg/helper/validate"
	"github.com/tektoncd/cli/pkg/printer"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const describeTemplate = `{{decorate "bold" "Name"}}:	{{ .Task.Name }}
{{decorate "bold" "Namespace"}}:	{{ .Task.Namespace }}

{{decorate "inputresources" ""}}{{decorate "underline bold" "Input Resources\n"}}

{{- if not .Task.Spec.Inputs }}
 No input resources
{{- else }}
{{- if eq (len .Task.Spec.Inputs.Resources) 0 }}
 No input resources
{{- else }}
 NAME	TYPE
{{- range $ir := .Task.Spec.Inputs.Resources }}
 {{decorate "bullet" $ir.Name }}	{{ $ir.Type }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "outputresources" ""}}{{decorate "underline bold" "Output Resources\n"}}

{{- if not .Task.Spec.Outputs }}
 No output resources
{{- else }}
{{- if eq (len .Task.Spec.Outputs.Resources) 0 }}
 No output resources
{{- else }}
 NAME	  TYPE
 
{{- range $or := .Task.Spec.Outputs.Resources }}
 {{decorate "bullet" $or.Name }}	{{ $or.Type }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}

{{- if not .Task.Spec.Inputs }}
 No params
{{- else }}
{{- if eq (len .Task.Spec.Inputs.Params) 0 }}
No params
{{- else }}
 NAME	TYPE	DEFAULT VALUE
{{- range $p := .Task.Spec.Inputs.Params }}
{{- if not $p.Default }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ "---" }}
{{- else }}
{{- if eq $p.Type "string" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ $p.Default.StringVal }}
{{- else }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ $p.Default.ArrayVal }}
{{- end }}
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
		Args:         cobra.MinimumNArgs(1),
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

			if output != "" {
				return describeTaskOutput(cmd.OutOrStdout(), p, f, args[0])
			}

			return printTaskDescription(s, p, args[0])
		},
	}

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_task")
	f.AddFlags(c)
	return c
}

func describeTaskOutput(w io.Writer, p cli.Params, f *cliopts.PrintFlags, name string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	c := cs.Tekton.TektonV1alpha1().Tasks(p.Namespace())

	task, err := c.Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// NOTE: this is required for -o json|yaml to work properly since
	// tektoncd go client fails to set these; probably a bug
	task.GetObjectKind().SetGroupVersionKind(
		schema.GroupVersionKind{
			Version: "tekton.dev/v1alpha1",
			Kind:    "Task",
		})

	return printer.PrintObject(w, task, f)
}

func printTaskDescription(s *cli.Stream, p cli.Params, tname string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	task, err := cs.Tekton.TektonV1alpha1().Tasks(p.Namespace()).Get(tname, metav1.GetOptions{})
	if err != nil {
		fmt.Fprintf(s.Err, "failed to get task %s\n", tname)
		return err
	}

	if task.Spec.Inputs != nil {
		task.Spec.Inputs.Resources = sortResourcesByTypeAndName(task.Spec.Inputs.Resources)
	}

	if task.Spec.Outputs != nil {
		task.Spec.Outputs.Resources = sortResourcesByTypeAndName(task.Spec.Outputs.Resources)
	}

	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/task=%s", tname),
	}
	taskRuns, err := cs.Tekton.TektonV1alpha1().TaskRuns(p.Namespace()).List(opts)
	if err != nil {
		fmt.Fprintf(s.Err, "failed to get taskruns for task %s \n", tname)
		return err
	}

	var data = struct {
		Task     *v1alpha1.Task
		TaskRuns *v1alpha1.TaskRunList
		Time     clockwork.Clock
	}{
		Task:     task,
		TaskRuns: taskRuns,
		Time:     p.Time(),
	}

	funcMap := template.FuncMap{
		"formatAge":       formatted.Age,
		"formatDuration":  formatted.Duration,
		"formatCondition": formatted.Condition,
		"decorate":        formatted.DecorateAttr,
		"autoStepName":    formatted.AutoStepName,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Describe Task").Funcs(funcMap).Parse(describeTemplate))
	err = t.Execute(w, data)
	if err != nil {
		fmt.Fprintf(s.Err, "Failed to execute template \n")
		return err
	}
	return nil
}

// this will sort the Task Resource by Type and then by Name
func sortResourcesByTypeAndName(tres []v1alpha1.TaskResource) []v1alpha1.TaskResource {
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
