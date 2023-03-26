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

package pipelinerun

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const describeTemplate = `{{decorate "bold" "Name"}}:	{{ .PipelineRun.Name }}
{{decorate "bold" "Namespace"}}:	{{ .PipelineRun.Namespace }}
{{- $pRefName := pipelineRefExists .PipelineRun.Spec }}{{- if ne $pRefName "" }}
{{decorate "bold" "Pipeline Ref"}}:	{{ $pRefName }}
{{- end }}
{{- if ne .PipelineRun.Spec.TaskRunTemplate.ServiceAccountName "" }}
{{decorate "bold" "Service Account"}}:	{{ .PipelineRun.Spec.TaskRunTemplate.ServiceAccountName }}
{{- end }}

{{- $l := len .PipelineRun.Labels }}{{ if eq $l 0 }}
{{- else }}
{{decorate "bold" "Labels"}}:
{{- range $k, $v := .PipelineRun.Labels }}
 {{ $k }}={{ $v }}
{{- end }}
{{- end }}
{{- $annotations := removeLastAppliedConfig .PipelineRun.Annotations -}}
{{- if $annotations }}
{{decorate "bold" "Annotations"}}:
{{- range $k, $v := $annotations }}
 {{ $k }}={{ $v }}
{{- end }}
{{- end }}

{{decorate "status" ""}}{{decorate "underline bold" "Status\n"}}
STARTED	DURATION	STATUS
{{ formatAge .PipelineRun.Status.StartTime  .Time }}	{{ formatDuration .PipelineRun.Status.StartTime .PipelineRun.Status.CompletionTime }}	{{ formatCondition .PipelineRun.Status.Conditions }}
{{- $msg := hasFailed .PipelineRun .TaskrunList -}}
{{-  if ne $msg "" }}

{{decorate "message" ""}}{{decorate "underline bold" "Message\n"}}
{{ $msg }}
{{- end }}


{{- if .PipelineRun.Spec.Timeouts }}

{{decorate "timeouts" ""}}{{decorate "underline bold" "Timeouts"}}
{{- $timeout := .PipelineRun.Spec.Timeouts.Pipeline -}}
{{- if $timeout }}
 {{decorate "bold" "Pipeline"}}:	{{ $timeout.Duration.String }}
{{- end }}
{{- $timeout := .PipelineRun.Spec.Timeouts.Tasks -}}
{{- if $timeout }}
 {{decorate "bold" "Tasks"}}:	{{ $timeout.Duration.String }}
{{- end }}
{{- $timeout := .PipelineRun.Spec.Timeouts.Finally -}}
{{- if $timeout }}
 {{decorate "bold" "Finally"}}:	{{ $timeout.Duration.String }}
{{- end }}
{{- end }}

{{- if ne (len .PipelineRun.Spec.Params) 0 }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}
 NAME	VALUE
{{- range $i, $p := .PipelineRun.Spec.Params }}
{{- if eq $p.Value.Type "string" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value.StringVal }}
{{- else if eq $p.Value.Type "array" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value.ArrayVal }}
{{- else }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value.ObjectVal }}
{{- end }}
{{- end }}
{{- end }}

{{- if ne (len .PipelineRun.Status.Results) 0 }}

{{decorate "results" ""}}{{decorate "underline bold" "Results\n"}}
 NAME	VALUE
{{- range $result := .PipelineRun.Status.Results }}
{{- if eq $result.Value.Type "string" }}
 {{decorate "bullet" $result.Name }}	{{ $result.Value.StringVal }}
{{- else }}
 {{decorate "bullet" $result.Name }}	{{ $result.Value.ArrayVal }}
{{- end }}
{{- end }}
{{- end }}

{{- if ne (len .PipelineRun.Spec.Workspaces) 0 }}

{{decorate "workspaces" ""}}{{decorate "underline bold" "Workspaces\n"}}
 NAME	SUB PATH	WORKSPACE BINDING
{{- range $workspace := .PipelineRun.Spec.Workspaces }}
{{- if not $workspace.SubPath }}
 {{ decorate "bullet" $workspace.Name }}	{{ "---" }}	{{ formatWorkspace $workspace }}
{{- else }}
 {{ decorate "bullet" $workspace.Name }}	{{ $workspace.SubPath }}	{{ formatWorkspace $workspace }}
{{- end }}
{{- end }}
{{- end }}

{{- if ne (len .TaskrunList) 0 }}

{{decorate "taskruns" ""}}{{decorate "underline bold" "Taskruns\n"}}
 NAME	TASK NAME	STARTED	DURATION	STATUS
{{- range $taskrun := .TaskrunList }}{{ if checkTRStatus $taskrun }}
 {{decorate "bullet" $taskrun.TaskRunName }}	{{ $taskrun.PipelineTaskName }}	{{ formatAge $taskrun.Status.StartTime $.Time }}	{{ formatDuration $taskrun.Status.StartTime $taskrun.Status.CompletionTime }}	{{ formatCondition $taskrun.Status.Conditions }}
{{- end }}
{{- end }}
{{- end }}

{{- if ne (len .PipelineRun.Status.SkippedTasks) 0 }}

{{decorate "skippedtasks" ""}}{{decorate "underline bold" "Skipped Tasks\n"}}
 NAME
{{- range $skippedTask := .PipelineRun.Status.SkippedTasks }}
 {{decorate "bullet" $skippedTask.Name }}
{{- end }}
{{- end }}
`

type TaskRunWithStatus struct {
	TaskRunName      string
	PipelineTaskName string
	Status           *v1.TaskRunStatus
}

type TaskRunWithStatusList []TaskRunWithStatus

func (trs TaskRunWithStatusList) Len() int      { return len(trs) }
func (trs TaskRunWithStatusList) Swap(i, j int) { trs[i], trs[j] = trs[j], trs[i] }
func (trs TaskRunWithStatusList) Less(i, j int) bool {
	if trs[j].Status == nil || trs[j].Status.StartTime == nil {
		return false
	}

	if trs[i].Status == nil || trs[i].Status.StartTime == nil {
		return true
	}

	return trs[j].Status.StartTime.Before(trs[i].Status.StartTime)
}

func PrintPipelineRunDescription(out io.Writer, c *cli.Clients, ns string, prName string, time clockwork.Clock) error {
	pr, err := GetPipelineRun(pipelineRunGroupResource, c, prName, ns)
	if err != nil {
		return fmt.Errorf("failed to find pipelinerun %q", prName)
	}

	var taskRunList TaskRunWithStatusList
	for _, child := range pr.Status.ChildReferences {
		if child.Kind == "TaskRun" {
			var tr *v1.TaskRun
			err = actions.GetV1(taskrunGroupResource, c, child.Name, ns, metav1.GetOptions{}, &tr)
			if err != nil {
				return fmt.Errorf("failed to find get taskruns of the pipelineruns")
			}
			taskRunList = append(taskRunList, TaskRunWithStatus{
				tr.Name,
				child.PipelineTaskName,
				&tr.Status,
			})
		}
	}

	if len(taskRunList) != 0 {
		sort.Sort(taskRunList)
	}

	var data = struct {
		PipelineRun *v1.PipelineRun
		Time        clockwork.Clock
		TaskrunList TaskRunWithStatusList
	}{
		PipelineRun: pr,
		Time:        time,
		TaskrunList: taskRunList,
	}

	funcMap := template.FuncMap{
		"formatAge":               formatted.Age,
		"formatDuration":          formatted.Duration,
		"formatCondition":         formatted.Condition,
		"formatWorkspace":         formatted.Workspace,
		"hasFailed":               hasFailed,
		"pipelineRefExists":       formatted.PipelineRefExists,
		"decorate":                formatted.DecorateAttr,
		"checkTRStatus":           checkTaskRunStatus,
		"removeLastAppliedConfig": formatted.RemoveLastAppliedConfig,
	}

	w := tabwriter.NewWriter(out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Describe Pipelinerun").Funcs(funcMap).Parse(describeTemplate))

	if err = t.Execute(w, data); err != nil {
		return err
	}
	return w.Flush()
}

func GetPipelineRun(gr schema.GroupVersionResource, c *cli.Clients, prName, ns string) (*v1.PipelineRun, error) {
	var pipelinerun v1.PipelineRun
	gvr, err := actions.GetGroupVersionResource(gr, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if gvr.Version == "v1" {
		err := actions.GetV1(pipelineRunGroupResource, c, prName, ns, metav1.GetOptions{}, &pipelinerun)
		if err != nil {
			return nil, err
		}
		return &pipelinerun, nil
	}

	var pipelinerunV1beta1 v1beta1.PipelineRun
	err = actions.GetV1(pipelineRunGroupResource, c, prName, ns, metav1.GetOptions{}, &pipelinerunV1beta1)
	if err != nil {
		return nil, err
	}

	err = pipelinerunV1beta1.ConvertTo(context.Background(), &pipelinerun)
	if err != nil {
		return nil, err
	}
	return &pipelinerun, nil
}

func hasFailed(pr *v1.PipelineRun, taskruns TaskRunWithStatusList) string {
	if len(pr.Status.Conditions) == 0 {
		return ""
	}

	if pr.Status.Conditions[0].Status == corev1.ConditionFalse {
		var trNames []string
		for _, taskrun := range taskruns {
			if taskrun.Status == nil {
				continue
			}
			if len(taskrun.Status.Conditions) == 0 {
				continue
			}
			if taskrun.Status.Conditions[0].Status == corev1.ConditionFalse {
				trNames = append(trNames, taskrun.TaskRunName)
			}
		}
		message := pr.Status.Conditions[0].Message
		if len(trNames) != 0 {
			sort.Strings(trNames)
			message += fmt.Sprintf("\nTaskRun(s) cancelled: %s", strings.Join(trNames, ", "))
		}
		return message
	}
	return ""
}

func checkTaskRunStatus(taskRun TaskRunWithStatus) bool {
	return taskRun.Status != nil
}
