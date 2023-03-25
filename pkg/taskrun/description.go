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

package taskrun

import (
	"context"
	"fmt"
	"io"
	"sort"
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

const templ = `{{decorate "bold" "Name"}}:	{{ .TaskRun.Name }}
{{decorate "bold" "Namespace"}}:	{{ .TaskRun.Namespace }}
{{- $tRefName := taskRefExists .TaskRun.Spec }}{{- if ne $tRefName "" }}
{{decorate "bold" "Task Ref"}}:	{{ $tRefName }}
{{- end }}
{{- if ne .TaskRun.Spec.ServiceAccountName "" }}
{{decorate "bold" "Service Account"}}:	{{ .TaskRun.Spec.ServiceAccountName }}
{{- end }}

{{- $timeout := getTimeout .TaskRun -}}
{{- if and (ne $timeout "") (ne $timeout "0s") }}
{{decorate "bold" "Timeout"}}:	{{ .TaskRun.Spec.Timeout.Duration.String }}
{{- end }}
{{- $l := len .TaskRun.Labels }}{{ if eq $l 0 }}
{{- else }}
{{decorate "bold" "Labels"}}:
{{- range $k, $v := .TaskRun.Labels }}
 {{ $k }}={{ $v }}
{{- end }}
{{- end }}
{{- $annotations := removeLastAppliedConfig .TaskRun.Annotations -}}
{{- if $annotations }}
{{decorate "bold" "Annotations"}}:
{{- range $k, $v := $annotations }}
 {{ $k }}={{ $v }}
{{- end }}
{{- end }}

{{decorate "status" ""}}{{decorate "underline bold" "Status"}}

STARTED 	DURATION 	STATUS
{{ formatAge .TaskRun.Status.StartTime  .Time }}	{{ formatDuration .TaskRun.Status.StartTime .TaskRun.Status.CompletionTime }}	{{ formatCondition .TaskRun.Status.Conditions }}
{{- $msg := hasFailed .TaskRun -}}
{{-  if ne $msg "" }}

{{decorate "underline bold" "Message"}}

{{ $msg }}
{{- end }}

{{- if ne (len .TaskRun.Spec.Params) 0 }}

{{decorate "params" ""}}{{decorate "underline bold" "Params"}}

 NAME	VALUE
{{- range $i, $p := .TaskRun.Spec.Params }}
{{- if eq $p.Value.Type "string" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value.StringVal }}
{{- else if eq $p.Value.Type "array" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value.ArrayVal }}
{{- else }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value.ObjectVal }}
{{- end }}
{{- end }}
{{- end }}

{{- if ne (len .TaskRun.Status.Results) 0 }}

{{decorate "results" ""}}{{decorate "underline bold" "Results"}}

 NAME	VALUE
{{- range $result := .TaskRun.Status.Results }}
 {{decorate "bullet" $result.Name }}	{{ formatResult $result.Value }}
{{- end }}
{{- end }}

{{- if ne (len .TaskRun.Spec.Workspaces) 0 }}

{{decorate "workspaces" ""}}{{decorate "underline bold" "Workspaces"}}

 NAME	SUB PATH	WORKSPACE BINDING
{{- range $workspace := .TaskRun.Spec.Workspaces }}
{{- if not $workspace.SubPath }}
 {{ decorate "bullet" $workspace.Name }}	{{ "---" }}	{{ formatWorkspace $workspace }}
{{- else }}
 {{ decorate "bullet" $workspace.Name }}	{{ $workspace.SubPath }}	{{ formatWorkspace $workspace }}
{{- end }}
{{- end }}
{{- end }}

{{- $sortedSteps := sortStepStates .TaskRun.Status.Steps }}
{{- if ne (len $sortedSteps) 0 }}

{{decorate "steps" ""}}{{decorate "underline bold" "Steps"}}

 NAME	STATUS
{{- range $step := $sortedSteps }}
{{- $reason := stepReasonExists $step }}
 {{decorate "bullet" $step.Name }}	{{ $reason }}
{{- end }}
{{- end }}

{{- $sidecars := .TaskRun.Status.Sidecars }}
{{- if ne (len $sidecars) 0 }}

{{decorate "sidecars" ""}}{{decorate "underline bold" "Sidecars"}}

 NAME	STATUS
{{- range $sidecar := $sidecars }}
{{- $reason := sidecarReasonExists $sidecar }}
 {{decorate "bullet" $sidecar.Name }}	{{ $reason }}
{{- end }}
{{- end }}
`

func sortStepStatesByStartTime(steps []v1.StepState) []v1.StepState {
	sort.Slice(steps, func(i, j int) bool {
		if steps[j].Waiting != nil && steps[i].Waiting != nil {
			return false
		}

		var jStartTime metav1.Time
		jRunning := false
		var iStartTime metav1.Time
		iRunning := false
		if steps[j].Terminated == nil {
			if steps[j].Running != nil {
				jStartTime = steps[j].Running.StartedAt
				jRunning = true
			} else {
				return true
			}
		}

		if steps[i].Terminated == nil {
			if steps[i].Running != nil {
				iStartTime = steps[i].Running.StartedAt
				iRunning = true
			} else {
				return false
			}
		}

		if !jRunning {
			jStartTime = steps[j].Terminated.StartedAt
		}

		if !iRunning {
			iStartTime = steps[i].Terminated.StartedAt
		}

		return iStartTime.Before(&jStartTime)
	})

	return steps
}

func PrintTaskRunDescription(out io.Writer, c *cli.Clients, ns string, trName string, time clockwork.Clock) error {
	tr, err := GetTaskRun(taskrunGroupResource, c, trName, ns)
	if err != nil {
		return fmt.Errorf("failed to get TaskRun %s: %v", trName, err)
	}

	var data = struct {
		TaskRun *v1.TaskRun
		Time    clockwork.Clock
	}{
		TaskRun: tr,
		Time:    time,
	}

	funcMap := template.FuncMap{
		"formatAge":               formatted.Age,
		"formatDuration":          formatted.Duration,
		"formatCondition":         formatted.Condition,
		"formatResult":            formatted.Result,
		"formatWorkspace":         formatted.Workspace,
		"hasFailed":               hasFailed,
		"taskRefExists":           formatted.TaskRefExists,
		"stepReasonExists":        stepReasonExists,
		"sidecarReasonExists":     sidecarReasonExists,
		"decorate":                formatted.DecorateAttr,
		"sortStepStates":          sortStepStatesByStartTime,
		"getTimeout":              getTimeoutValue,
		"removeLastAppliedConfig": formatted.RemoveLastAppliedConfig,
	}

	w := tabwriter.NewWriter(out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Describe TaskRun").Funcs(funcMap).Parse(templ))

	err = t.Execute(w, data)
	if err != nil {
		return err
	}
	return w.Flush()
}

func hasFailed(tr *v1.TaskRun) string {
	if len(tr.Status.Conditions) == 0 {
		return ""
	}

	if tr.Status.Conditions[0].Status == corev1.ConditionFalse {
		return tr.Status.Conditions[0].Message
	}

	return ""
}

func getTimeoutValue(tr *v1.TaskRun) string {
	if tr.Spec.Timeout != nil {
		return tr.Spec.Timeout.Duration.String()
	}
	return ""
}

// Check if step is in waiting, running, or terminated state by checking StepState of the step.
func stepReasonExists(state v1.StepState) string {
	if state.Waiting == nil {
		if state.Running != nil {
			return formatted.ColorStatus("Running")
		}

		if state.Terminated != nil {
			return formatted.ColorStatus(state.Terminated.Reason)
		}

		return formatted.ColorStatus("---")
	}

	return formatted.ColorStatus(state.Waiting.Reason)
}

// Check if sidecar is in waiting, running, or terminated state by checking SidecarState of the sidecar.
func sidecarReasonExists(state v1.SidecarState) string {
	if state.Waiting == nil {

		if state.Running != nil {
			return formatted.ColorStatus("Running")
		}

		if state.Terminated != nil {
			return formatted.ColorStatus(state.Terminated.Reason)
		}

		return formatted.ColorStatus("---")
	}

	return formatted.ColorStatus(state.Waiting.Reason)
}

func GetTaskRun(gr schema.GroupVersionResource, c *cli.Clients, trName, ns string) (*v1.TaskRun, error) {
	var taskrun v1.TaskRun
	gvr, err := actions.GetGroupVersionResource(gr, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if gvr.Version == "v1" {
		err := actions.GetV1(gr, c, trName, ns, metav1.GetOptions{}, &taskrun)
		if err != nil {
			return nil, err
		}
		return &taskrun, nil
	}

	var taskrunV1beta1 v1beta1.TaskRun
	err = actions.GetV1(gr, c, trName, ns, metav1.GetOptions{}, &taskrunV1beta1)
	if err != nil {
		return nil, err
	}

	err = taskrunV1beta1.ConvertTo(context.Background(), &taskrun)
	if err != nil {
		return nil, err
	}
	return &taskrun, nil
}
