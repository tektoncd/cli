package description

import (
	"fmt"
	"html/template"
	"sort"
	"text/tabwriter"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/helper/validate"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
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
{{- if ne $timeout "" }}
{{decorate "bold" "Timeout"}}:	{{ .PipelineRun.Spec.Timeout.Duration.String }}
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

{{decorate "taskruns" ""}}{{decorate "underline bold" "Taskruns\n"}}
{{- $l := len .TaskrunList }}{{ if eq $l 0 }}
 No taskruns
{{- else }}
 NAME	TASK NAME	STARTED	DURATION	STATUS
{{- range $taskrun := .TaskrunList }}
 {{decorate "bullet" $taskrun.TaskrunName }}	{{ $taskrun.PipelineTaskName }}	{{ formatAge $taskrun.Status.StartTime $.Params.Time }}	{{ formatDuration $taskrun.Status.StartTime $taskrun.Status.CompletionTime }}	{{ formatCondition $taskrun.Status.Conditions }}
{{- end }}
{{- end }}
`

type taskrunList []tkr

func (trs taskrunList) Len() int      { return len(trs) }
func (trs taskrunList) Swap(i, j int) { trs[i], trs[j] = trs[j], trs[i] }
func (trs taskrunList) Less(i, j int) bool {
	if trs[j].Status.StartTime == nil {
		return false
	}

	if trs[i].Status.StartTime == nil {
		return true
	}

	return trs[j].Status.StartTime.Before(trs[i].Status.StartTime)
}

type tkr struct {
	TaskrunName string
	*v1alpha1.PipelineRunTaskRunStatus
}

func newTaskrunListFromMap(statusMap map[string]*v1alpha1.PipelineRunTaskRunStatus) taskrunList {

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
		return fmt.Errorf("failed to create tekton client")
	}

	pr, err := cs.Tekton.TektonV1alpha1().PipelineRuns(p.Namespace()).Get(prName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to find pipelinerun %q", prName)
	}

	var trl taskrunList

	if len(pr.Status.TaskRuns) != 0 {
		trl = newTaskrunListFromMap(pr.Status.TaskRuns)
		sort.Sort(trl)
	}

	var data = struct {
		PipelineRun *v1alpha1.PipelineRun
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
		"hasFailed":                 hasFailed,
		"pipelineRefExists":         validate.PipelineRefExists,
		"pipelineResourceRefExists": validate.PipelineResourceRefExists,
		"decorate":                  formatted.DecorateAttr,
		"getTimeout":                getTimeoutValue,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Describe Pipelinerun").Funcs(funcMap).Parse(templ))

	if err = t.Execute(w, data); err != nil {
		fmt.Fprintf(s.Err, "Failed to execute template")
		return err
	}
	return w.Flush()
}

func hasFailed(pr *v1alpha1.PipelineRun) string {
	if len(pr.Status.Conditions) == 0 {
		return ""
	}

	if pr.Status.Conditions[0].Status == corev1.ConditionFalse {
		for _, taskrunStatus := range pr.Status.TaskRuns {
			if len(taskrunStatus.Status.Conditions) == 0 {
				continue
			}
			if taskrunStatus.Status.Conditions[0].Status == corev1.ConditionFalse {
				return fmt.Sprintf("%s (%s)", pr.Status.Conditions[0].Message,
					taskrunStatus.Status.Conditions[0].Message)
			}
		}
		return pr.Status.Conditions[0].Message
	}
	return ""
}

func getTimeoutValue(pr *v1alpha1.PipelineRun) string {
	if pr.Spec.Timeout != nil {
		return pr.Spec.Timeout.Duration.String()
	}
	return ""
}
