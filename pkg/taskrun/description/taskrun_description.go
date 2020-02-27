package description

import (
	"fmt"
	"sort"
	"text/tabwriter"
	"text/template"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/validate"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
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

{{decorate "status" ""}}{{decorate "underline bold" "Status"}}

STARTED 	DURATION 	STATUS
{{ formatAge .TaskRun.Status.StartTime  .Params.Time }}	{{ formatDuration .TaskRun.Status.StartTime .TaskRun.Status.CompletionTime }}	{{ formatCondition .TaskRun.Status.Conditions }}
{{- $msg := hasFailed .TaskRun -}}
{{-  if ne $msg "" }}

{{decorate "underline bold" "Message"}}

{{ $msg }}
{{- end }}

{{decorate "inputresources" ""}}{{decorate "underline bold" "Input Resources\n"}}

{{- $l := len .TaskRun.Spec.Inputs.Resources }}{{ if eq $l 0 }}
No resources
{{- else }}
 NAME	RESOURCE REF
{{- range $i, $r := .TaskRun.Spec.Inputs.Resources }}
{{- $rRefName := taskResourceRefExists $r }}{{- if ne $rRefName "" }}
 {{decorate "bullet" $r.Name }}	{{ $r.ResourceRef.Name }}
{{- else }}
 {{decorate "bullet" $r.Name }}	{{ "" }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "outputresources" ""}}{{decorate "underline bold" "Output Resources\n"}}

{{- $l := len .TaskRun.Spec.Outputs.Resources }}{{ if eq $l 0 }}
No resources
{{- else }}
 NAME	RESOURCE REF
{{- range $i, $r := .TaskRun.Spec.Outputs.Resources }}
{{- $rRefName := taskResourceRefExists $r }}{{- if ne $rRefName "" }}
 {{decorate "bullet" $r.Name }}	{{ $r.ResourceRef.Name }}
{{- else }}
 {{decorate "bullet" $r.Name }}	{{ "" }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}

{{- $l := len .TaskRun.Spec.Inputs.Params }}{{ if eq $l 0 }}
No params
{{- else }}
 NAME	VALUE
{{- range $i, $p := .TaskRun.Spec.Inputs.Params }}
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
 NAME
{{- range $sidecar := $sidecars }}
 {{decorate "bullet" $sidecar.Name }}
{{- end }}
{{- end }}
`

func sortStepStatesByStartTime(steps []v1alpha1.StepState) []v1alpha1.StepState {
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
		return fmt.Errorf("failed to create tekton client")
	}

	tr, err := cs.Tekton.TektonV1alpha1().TaskRuns(p.Namespace()).Get(trName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to find taskrun %q", trName)
	}

	var data = struct {
		TaskRun *v1alpha1.TaskRun
		Params  cli.Params
	}{
		TaskRun: tr,
		Params:  p,
	}

	funcMap := template.FuncMap{
		"formatAge":             formatted.Age,
		"formatDuration":        formatted.Duration,
		"formatCondition":       formatted.Condition,
		"hasFailed":             hasFailed,
		"taskRefExists":         validate.TaskRefExists,
		"taskResourceRefExists": validate.TaskResourceRefExists,
		"stepReasonExists":      validate.StepReasonExists,
		"decorate":              formatted.DecorateAttr,
		"sortStepStates":        sortStepStatesByStartTime,
		"getTimeout":            getTimeoutValue,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Describe taskrun").Funcs(funcMap).Parse(templ))

	err = t.Execute(w, data)
	if err != nil {
		fmt.Fprintf(s.Err, "Failed to execute template")
		return err
	}
	return w.Flush()
}

func hasFailed(tr *v1alpha1.TaskRun) string {
	if len(tr.Status.Conditions) == 0 {
		return ""
	}

	if tr.Status.Conditions[0].Status == corev1.ConditionFalse {
		return tr.Status.Conditions[0].Message
	}

	return ""
}

func getTimeoutValue(tr *v1alpha1.TaskRun) string {
	if tr.Spec.Timeout != nil {
		return tr.Spec.Timeout.Duration.String()
	}
	return ""
}
