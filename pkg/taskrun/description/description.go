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

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
{{- if ne $timeout "" }}
{{decorate "bold" "Timeout"}}:	{{ .TaskRun.Spec.Timeout.Duration.String }}
{{- end }}
{{- $l := len .TaskRun.Labels }}{{ if eq $l 0 }}
{{- else }}
{{decorate "bold" "Labels"}}:
{{- range $k, $v := .TaskRun.Labels }}
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

{{decorate "inputresources" ""}}{{decorate "underline bold" "Input Resources\n"}}

{{- if not .TaskRun.Spec.Resources }}
 No input resources
{{- else }}
{{- $l := len .TaskRun.Spec.Resources.Inputs }}{{ if eq $l 0 }}
 No input resources
{{- else }}
 NAME	RESOURCE REF
{{- range $ir := .TaskRun.Spec.Resources.Inputs }}
{{- $rRefName := taskResourceRefExists $ir }}{{- if ne $rRefName "" }}
 {{decorate "bullet" $ir.Name }}	{{ $ir.ResourceRef.Name }}
{{- else }}
 {{decorate "bullet" $ir.Name }}	{{ "" }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "outputresources" ""}}{{decorate "underline bold" "Output Resources\n"}}

{{- if not .TaskRun.Spec.Resources }}
 No output resources
{{- else }}
{{- $l := len .TaskRun.Spec.Resources.Outputs }}{{ if eq $l 0 }}
 No output resources
{{- else }}
 NAME	RESOURCE REF
{{- range $or := .TaskRun.Spec.Resources.Outputs }}
{{- $rRefName := taskResourceRefExists $or }}{{- if ne $rRefName "" }}
 {{decorate "bullet" $or.Name }}	{{ $or.ResourceRef.Name }}
{{- else }}
 {{decorate "bullet" $or.Name }}	{{ "" }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}

{{- $l := len .TaskRun.Spec.Params }}{{ if eq $l 0 }}
 No params
{{- else }}
 NAME	VALUE
{{- range $i, $p := .TaskRun.Spec.Params }}
{{- if eq $p.Value.Type "string" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value.StringVal }}
{{- else }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value.ArrayVal }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "steps" ""}}{{decorate "underline bold" "Steps"}}
{{$sortedSteps := sortStepStates .TaskRun.Status.Steps }}
{{- $l := len $sortedSteps }}{{ if eq $l 0 }}
No steps
{{- else }}
 NAME	STATUS
{{- range $step := $sortedSteps }}
{{- $reason := stepReasonExists $step }}
 {{decorate "bullet" $step.Name }}	{{ $reason }}
{{- end }}
{{- end }}

{{decorate "sidecars" ""}}{{decorate "underline bold" "Sidecars"}}
{{$sidecars := .TaskRun.Status.Sidecars }}
{{- $l := len $sidecars }}{{ if eq $l 0 }}
No sidecars
{{- else }}
 NAME	STATUS
{{- range $sidecar := $sidecars }}
{{- $reason := sidecarReasonExists $sidecar }}
 {{decorate "bullet" $sidecar.Name }}	{{ $reason }}
{{- end }}
{{- end }}
`

func sortStepStatesByStartTime(steps []v1beta1.StepState) []v1beta1.StepState {
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

func PrintTaskRunDescription(s *cli.Stream, trName string, p cli.Params) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client: %v", err)
	}

	tr, err := taskrun.Get(cs, trName, metav1.GetOptions{}, p.Namespace())
	if err != nil {
		return fmt.Errorf("failed to get TaskRun %s: %v", trName, err)
	}

	var data = struct {
		TaskRun *v1beta1.TaskRun
		Time    clockwork.Clock
	}{
		TaskRun: tr,
		Time:    p.Time(),
	}

	funcMap := template.FuncMap{
		"formatAge":             formatted.Age,
		"formatDuration":        formatted.Duration,
		"formatCondition":       formatted.Condition,
		"hasFailed":             hasFailed,
		"taskRefExists":         taskRefExists,
		"taskResourceRefExists": taskResourceRefExists,
		"stepReasonExists":      stepReasonExists,
		"sidecarReasonExists":   sidecarReasonExists,
		"decorate":              formatted.DecorateAttr,
		"sortStepStates":        sortStepStatesByStartTime,
		"getTimeout":            getTimeoutValue,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Describe TaskRun").Funcs(funcMap).Parse(templ))

	err = t.Execute(w, data)
	if err != nil {
		fmt.Fprintf(s.Err, "failed to execute template: ")
		return err
	}
	return w.Flush()
}

func hasFailed(tr *v1beta1.TaskRun) string {
	if len(tr.Status.Conditions) == 0 {
		return ""
	}

	if tr.Status.Conditions[0].Status == corev1.ConditionFalse {
		return tr.Status.Conditions[0].Message
	}

	return ""
}

func getTimeoutValue(tr *v1beta1.TaskRun) string {
	if tr.Spec.Timeout != nil {
		return tr.Spec.Timeout.Duration.String()
	}
	return ""
}

// Check if TaskRef exists on a TaskRunSpec. Returns empty string if not present.
func taskRefExists(spec v1beta1.TaskRunSpec) string {
	if spec.TaskRef == nil {
		return ""
	}

	return spec.TaskRef.Name
}

// Check if TaskResourceRef exists on a TaskResourceBinding. Returns empty string if not present.
func taskResourceRefExists(res v1beta1.TaskResourceBinding) string {
	if res.ResourceRef == nil {
		return ""
	}

	return res.ResourceRef.Name
}

// Check if step is in waiting, running, or terminated state by checking StepState of the step.
func stepReasonExists(state v1beta1.StepState) string {
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
func sidecarReasonExists(state v1beta1.SidecarState) string {
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
