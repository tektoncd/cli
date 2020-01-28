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
	"io"
	"os"
	"sort"
	"text/tabwriter"
	"text/template"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/printer"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const describeTemplate = `{{decorate "bold" "Name"}}:	{{ .ClusterTask.Name }}

{{decorate "inputresources" ""}}{{decorate "underline bold" "Input Resources\n"}}

{{- if not .ClusterTask.Spec.Inputs }}
 No input resources
{{- else }}
{{- if eq (len .ClusterTask.Spec.Inputs.Resources) 0 }}
 No input resources
{{- else }}
 NAME	TYPE
{{- range $ir := .ClusterTask.Spec.Inputs.Resources }}
 {{decorate "bullet" $ir.Name }}	{{ $ir.Type }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "outputresources" ""}}{{decorate "underline bold" "Output Resources\n"}}

{{- if not .ClusterTask.Spec.Outputs }}
 No output resources
{{- else }}
{{- if eq (len .ClusterTask.Spec.Outputs.Resources) 0 }}
 No output resources
{{- else }}
 NAME	TYPE
 
{{- range $or := .ClusterTask.Spec.Outputs.Resources }}
 {{decorate "bullet" $or.Name }}	{{ $or.Type }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}

{{- if not .ClusterTask.Spec.Inputs }}
 No params
{{- else }}
{{- if eq (len .ClusterTask.Spec.Inputs.Params) 0 }}
No params
{{- else }}
 NAME	TYPE	DEFAULT VALUE
{{- range $p := .ClusterTask.Spec.Inputs.Params }}
{{- if not $p.Default }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ "" }}
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

{{- if eq (len .ClusterTask.Spec.Steps) 0 }}
 No steps
{{- else }}
{{- range $step := .ClusterTask.Spec.Steps }}
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
	eg := `Describe a ClusterTask of name 'foo':

    tkn clustertask describe foo

or

    tkn ct desc foo
`

	c := &cobra.Command{
		Use:     "describe",
		Aliases: []string{"desc"},
		Short:   "Describes a clustertask",
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

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				fmt.Fprint(os.Stderr, "Error: output option not set properly \n")
				return err
			}

			if output != "" {
				return describeClusterTaskOutput(cmd.OutOrStdout(), p, f, args[0])
			}

			return printClusterTaskDescription(s, p, args[0])
		},
	}

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_clustertasks")
	f.AddFlags(c)
	return c
}

func describeClusterTaskOutput(w io.Writer, p cli.Params, f *cliopts.PrintFlags, name string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	c := cs.Tekton.TektonV1alpha1().ClusterTasks()

	clustertask, err := c.Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// NOTE: this is required for -o json|yaml to work properly since
	// tektoncd go client fails to set these; probably a bug
	clustertask.GetObjectKind().SetGroupVersionKind(
		schema.GroupVersionKind{
			Version: "tekton.dev/v1alpha1",
			Kind:    "ClusterTask",
		})

	return printer.PrintObject(w, clustertask, f)
}

func printClusterTaskDescription(s *cli.Stream, p cli.Params, tname string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	ct, err := cs.Tekton.TektonV1alpha1().ClusterTasks().Get(tname, metav1.GetOptions{})
	if err != nil {
		fmt.Fprintf(s.Err, "failed to get clustertask %s\n", tname)
		return err
	}

	if ct.Spec.Inputs != nil {
		ct.Spec.Inputs.Resources = sortResourcesByTypeAndName(ct.Spec.Inputs.Resources)
	}

	if ct.Spec.Outputs != nil {
		ct.Spec.Outputs.Resources = sortResourcesByTypeAndName(ct.Spec.Outputs.Resources)
	}

	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/task=%s", tname),
	}
	taskRuns, err := cs.Tekton.TektonV1alpha1().TaskRuns(p.Namespace()).List(opts)
	if err != nil {
		fmt.Fprintf(s.Err, "failed to get taskruns for clustertask %s \n", tname)
		return err
	}

	var data = struct {
		ClusterTask *v1alpha1.ClusterTask
		TaskRuns    *v1alpha1.TaskRunList
		Time        clockwork.Clock
	}{
		ClusterTask: ct,
		TaskRuns:    taskRuns,
		Time:        p.Time(),
	}

	funcMap := template.FuncMap{
		"formatAge":       formatted.Age,
		"formatDuration":  formatted.Duration,
		"formatCondition": formatted.Condition,
		"decorate":        formatted.DecorateAttr,
		"autoStepName":    formatted.AutoStepName,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Describe ClusterTask").Funcs(funcMap).Parse(describeTemplate))
	err = t.Execute(w, data)
	if err != nil {
		fmt.Fprintf(s.Err, "Failed to execute template \n")
		return err
	}
	return nil
}

// this will sort the ClusterTask Resource by Type and then by Name
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
