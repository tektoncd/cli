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

package description

import (
	"fmt"
	"sort"
	"text/tabwriter"
	"text/template"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/pipelinerun"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const templ = `{{decorate "bold" "Name"}}:	{{ .PipelineRun.Name }}
{{decorate "bold" "Namespace"}}:	{{ .PipelineRun.Namespace }}
{{- $pRefName := pipelineRefExists .PipelineRun.Spec }}{{- if ne $pRefName "" }}
{{decorate "bold" "Pipeline Ref"}}:	{{ $pRefName }}
{{- end }}
{{- if ne .PipelineRun.Spec.ServiceAccountName "" }}
{{decorate "bold" "Service Account"}}:	{{ .PipelineRun.Spec.ServiceAccountName }}
{{- end }}

{{- $timeout := getTimeout .PipelineRun -}}
{{- if and (ne $timeout "") (ne $timeout "0s") }}
{{decorate "bold" "Timeout"}}:	{{ .PipelineRun.Spec.Timeout.Duration.String }}
{{- end }}
{{- $l := len .PipelineRun.Labels }}{{ if eq $l 0 }}
{{- else }}
{{decorate "bold" "Labels"}}:
{{- range $k, $v := .PipelineRun.Labels }}
 {{ $k }}={{ $v }}
{{- end }}
{{- end }}

{{decorate "status" ""}}{{decorate "underline bold" "Status\n"}}
STARTED	DURATION	STATUS
{{ formatAge .PipelineRun.Status.StartTime  .Params.Time }}	{{ formatDuration .PipelineRun.Status.StartTime .PipelineRun.Status.CompletionTime }}	{{ formatCondition .PipelineRun.Status.Conditions }}
{{- $msg := hasFailed .PipelineRun -}}
{{-  if ne $msg "" }}

{{decorate "message" ""}}{{decorate "underline bold" "Message\n"}}
{{ $msg }}
{{- end }}

{{decorate "resources" ""}}{{decorate "underline bold" "Resources\n"}}
{{- $l := len .PipelineRun.Spec.Resources }}{{ if eq $l 0 }}
 No resources
{{- else }}
 NAME	RESOURCE REF
{{- range $i, $r := .PipelineRun.Spec.Resources }}
{{- $rRefName := pipelineResourceRefExists $r }}{{- if ne $rRefName "" }}
 {{decorate "bullet" $r.Name }}	{{ $r.ResourceRef.Name }}
{{- else }}
 {{decorate "bullet" $r.Name }}	{{ "" }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}
{{- $l := len .PipelineRun.Spec.Params }}{{ if eq $l 0 }}
 No params
{{- else }}
 NAME	VALUE
{{- range $i, $p := .PipelineRun.Spec.Params }}
{{- if eq $p.Value.Type "string" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value.StringVal }}
{{- else }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value.ArrayVal }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "results" ""}}{{decorate "underline bold" "Results\n"}}
{{- if eq (len .PipelineRun.Status.PipelineResults) 0 }}
 No results
{{- else }}
 NAME	VALUE
{{- range $result := .PipelineRun.Status.PipelineResults }}
 {{decorate "bullet" $result.Name }}	{{ formatResult $result.Value }}
{{- end }}
{{- end }}

{{decorate "workspaces" ""}}{{decorate "underline bold" "Workspaces\n"}}

{{- if eq (len .PipelineRun.Spec.Workspaces) 0 }}
 No workspaces
{{- else }}
 NAME	SUB PATH	WORKSPACE BINDING
{{- range $workspace := .PipelineRun.Spec.Workspaces }}
{{- if not $workspace.SubPath }}
 {{ decorate "bullet" $workspace.Name }}	{{ "---" }}	{{ formatWorkspace $workspace }}
{{- else }}
 {{ decorate "bullet" $workspace.Name }}	{{ $workspace.SubPath }}	{{ formatWorkspace $workspace }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "taskruns" ""}}{{decorate "underline bold" "Taskruns\n"}}
{{- $l := len .TaskrunList }}{{ if eq $l 0 }}
 No taskruns
{{- else }}
 NAME	TASK NAME	STARTED	DURATION	STATUS
{{- range $taskrun := .TaskrunList }}{{ if checkTRStatus $taskrun }}
 {{decorate "bullet" $taskrun.TaskrunName }}	{{ $taskrun.PipelineTaskName }}	{{ formatAge $taskrun.Status.StartTime $.Params.Time }}	{{ formatDuration $taskrun.Status.StartTime $taskrun.Status.CompletionTime }}	{{ formatCondition $taskrun.Status.Conditions }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "skippedtasks" ""}}{{decorate "underline bold" "Skipped Tasks\n"}}

{{- $l := len .PipelineRun.Status.SkippedTasks }}{{ if eq $l 0 }}
 No Skipped Tasks
{{- else }}
 NAME
{{- range $skippedTask := .PipelineRun.Status.SkippedTasks }}
 {{decorate "bullet" $skippedTask.Name }}
{{- end }}
{{- end }}
`

type tkr struct {
	TaskrunName string
	*v1beta1.PipelineRunTaskRunStatus
}

type taskrunList []tkr

func (trs taskrunList) Len() int      { return len(trs) }
func (trs taskrunList) Swap(i, j int) { trs[i], trs[j] = trs[j], trs[i] }
func (trs taskrunList) Less(i, j int) bool {
	if trs[j].Status == nil || trs[j].Status.StartTime == nil {
		return false
	}

	if trs[i].Status == nil || trs[i].Status.StartTime == nil {
		return true
	}

	return trs[j].Status.StartTime.Before(trs[i].Status.StartTime)
}

func newTaskrunListFromMap(statusMap map[string]*v1beta1.PipelineRunTaskRunStatus) taskrunList {
	var trl taskrunList
	for taskrunName, taskrunStatus := range statusMap {
		trl = append(trl, tkr{
			taskrunName,
			taskrunStatus,
		})
	}
	return trl
}

func PrintPipelineRunDescription(s *cli.Stream, prName string, p cli.Params) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client: %v", err)
	}

	pr, err := pipelinerun.Get(cs, prName, metav1.GetOptions{}, p.Namespace())
	if err != nil {
		return fmt.Errorf("failed to find pipelinerun %q", prName)
	}

	var trl taskrunList
	if len(pr.Status.TaskRuns) != 0 {
		trl = newTaskrunListFromMap(pr.Status.TaskRuns)
		sort.Sort(trl)
	}

	var data = struct {
		PipelineRun *v1beta1.PipelineRun
		Params      cli.Params
		TaskrunList taskrunList
	}{
		PipelineRun: pr,
		Params:      p,
		TaskrunList: trl,
	}

	funcMap := template.FuncMap{
		"formatAge":                 formatted.Age,
		"formatDuration":            formatted.Duration,
		"formatCondition":           formatted.Condition,
		"formatResult":              formatted.Result,
		"formatWorkspace":           formatted.Workspace,
		"hasFailed":                 hasFailed,
		"pipelineRefExists":         pipelineRefExists,
		"pipelineResourceRefExists": pipelineResourceRefExists,
		"decorate":                  formatted.DecorateAttr,
		"getTimeout":                getTimeoutValue,
		"checkTRStatus":             checkTaskRunStatus,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Describe Pipelinerun").Funcs(funcMap).Parse(templ))

	if err = t.Execute(w, data); err != nil {
		fmt.Fprintf(s.Err, "failed to execute template: ")
		return err
	}
	return w.Flush()
}

func hasFailed(pr *v1beta1.PipelineRun) string {
	if len(pr.Status.Conditions) == 0 {
		return ""
	}

	if pr.Status.Conditions[0].Status == corev1.ConditionFalse {
		for _, tr := range pr.Status.TaskRuns {
			if tr.Status == nil {
				continue
			}
			if len(tr.Status.Conditions) == 0 {
				continue
			}
			if tr.Status.Conditions[0].Status == corev1.ConditionFalse {
				return fmt.Sprintf("%s (%s)", pr.Status.Conditions[0].Message,
					tr.Status.Conditions[0].Message)
			}
		}
		return pr.Status.Conditions[0].Message
	}
	return ""
}

func getTimeoutValue(pr *v1beta1.PipelineRun) string {
	if pr.Spec.Timeout != nil {
		return pr.Spec.Timeout.Duration.String()
	}
	return ""
}

func checkTaskRunStatus(taskRun tkr) bool {
	return taskRun.PipelineRunTaskRunStatus.Status != nil
}

// Check if PipelineRef exists on a PipelineRunSpec. Returns empty string if not present.
func pipelineRefExists(spec v1beta1.PipelineRunSpec) string {
	if spec.PipelineRef == nil {
		return ""
	}

	return spec.PipelineRef.Name
}

// Check if PipelineResourceRef exists on a PipelineResourceBinding. Returns empty string if not present.
func pipelineResourceRefExists(res v1beta1.PipelineResourceBinding) string {
	if res.ResourceRef == nil {
		return ""
	}

	return res.ResourceRef.Name
}
