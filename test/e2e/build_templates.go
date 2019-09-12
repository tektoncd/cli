package e2e

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"sort"
	"testing"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jonboulle/clockwork"

	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const describeTemplate = `Name:	{{ .PipelineName }}

Resources
{{- $rl := len .Pipeline.Spec.Resources }}{{ if eq $rl 0 }}
No resources
{{- else }}
NAME	TYPE
{{- range $i, $r := .Pipeline.Spec.Resources }}
{{$r.Name }}	{{ $r.Type }}
{{- end }}
{{- end }}

Tasks
{{- $tl := len .Pipeline.Spec.Tasks }}{{ if eq $tl 0 }}
No tasks
{{- else }}
NAME	TASKREF	RUNAFTER
{{- range $i, $t := .Pipeline.Spec.Tasks }}
{{ $t.Name }}	{{ $t.TaskRef.Name }}	{{ $t.RunAfter }}
{{- end }}
{{- end }}

Runs
{{- $rl := len .PipelineRuns.Items }}{{ if eq $rl 0 }}
No runs
{{- else }}
NAME	STARTED	DURATION	STATUS
{{- range $i, $pr := .PipelineRuns.Items }}
{{ $pr.Name }}	{{ formatAge $pr.Status.StartTime $.Params }}	{{ formatDuration $pr.Status.StartTime $pr.Status.CompletionTime }}	{{ index $pr.Status.Conditions | formatCondition }}
{{- end }}
{{- end }}
`

func DescribeTemplateForPipelines(cs *Clients, pname string) string {
	log.Printf("validating Pipeline : %s describe command\n", pname)
	clock := clockwork.NewFakeClockAt(time.Now())

	pipeline := GetPipeline(cs, pname)
	pipelineRuns := GetPipelineRunListWithName(cs, pname)

	var data = struct {
		Pipeline     *v1alpha1.Pipeline
		PipelineRuns *v1alpha1.PipelineRunList
		PipelineName string
		Params       clockwork.Clock
	}{
		Pipeline:     pipeline,
		PipelineRuns: pipelineRuns,
		PipelineName: pname,
		Params:       clock,
	}

	funcMap := template.FuncMap{
		"formatAge":       formatted.Age,
		"formatDuration":  formatted.Duration,
		"formatCondition": formatted.Condition,
	}

	var tmplBytes bytes.Buffer

	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	t := template.Must(template.New("Describe Pipeline").Funcs(funcMap).Parse(describeTemplate))

	err1 := t.Execute(w, data)
	if err1 != nil {
		panic(err1)
	}

	w.Flush()
	return tmplBytes.String()

}

const describeTemplateForPipelinesRun = `Name:	{{ .PipelineRun.Name }}
Namespace:	{{ .PipelineRun.Namespace }}
Pipeline Ref:	{{ .PipelineRun.Spec.PipelineRef.Name }}
{{- if ne .PipelineRun.Spec.ServiceAccount "" }}
Service Account:	{{ .PipelineRun.Spec.ServiceAccount }}
{{- end }}

Status
STARTED	DURATION	STATUS
{{ formatAge .PipelineRun.Status.StartTime  .Params }}	{{ formatDuration .PipelineRun.Status.StartTime .PipelineRun.Status.CompletionTime }}	{{ formatCondition .PipelineRun.Status.Conditions }}
{{- $msg := hasFailed .PipelineRun -}}
{{-  if ne $msg "" }}

Message
{{ $msg }}
{{- end }}

Resources
{{- $l := len .PipelineRun.Spec.Resources }}{{ if eq $l 0 }}
No resources
{{- else }}
NAME	RESOURCE REF
{{- range $i, $r := .PipelineRun.Spec.Resources }}
{{$r.Name }}	{{ $r.ResourceRef.Name }}
{{- end }}
{{- end }}

Params
{{- $l := len .PipelineRun.Spec.Params }}{{ if eq $l 0 }}
No params
{{- else }}
NAME	VALUE
{{- range $i, $p := .PipelineRun.Spec.Params }}
{{ $p.Name }}	{{ $p.Value }}
{{- end }}
{{- end }}

Taskruns
{{- $l := len .TaskrunList }}{{ if eq $l 0 }}
No taskruns
{{- else }}
NAME	TASK NAME	STARTED	DURATION	STATUS
{{- range $taskrun := .TaskrunList }}
{{ $taskrun.TaskrunName }}	{{ $taskrun.PipelineTaskName }}	{{ formatAge $taskrun.Status.StartTime $.Params }}	{{ formatDuration $taskrun.Status.StartTime $taskrun.Status.CompletionTime }}	{{ formatCondition $taskrun.Status.Conditions }}
{{- end }}
{{- end }}
`

func DescribeTemplateForPipelineRun(cs *Clients, prname string) string {
	log.Printf("validating  PipelineRun : %s describe command\n", prname)
	clock := clockwork.NewFakeClockAt(time.Now())
	pipelinerun := GetPipelineRun(cs, prname)

	var trl taskrunList

	if len(pipelinerun.Status.TaskRuns) != 0 {
		trl = newTaskrunListFromMap(pipelinerun.Status.TaskRuns)
		sort.Sort(trl)
	}

	var data = struct {
		PipelineRun *v1alpha1.PipelineRun
		Params      clockwork.Clock
		TaskrunList taskrunList
	}{
		PipelineRun: pipelinerun,
		Params:      clock,
		TaskrunList: trl,
	}

	funcMap := template.FuncMap{
		"formatAge":       formatted.Age,
		"formatDuration":  formatted.Duration,
		"formatCondition": formatted.Condition,
		"hasFailed":       hasFailed,
	}

	t := template.Must(template.New("Describe PipelineRun").Funcs(funcMap).Parse(describeTemplateForPipelinesRun))

	var tmplBytes bytes.Buffer

	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	err1 := t.Execute(w, data)
	if err1 != nil {
		panic(err1)
	}

	w.Flush()

	return tmplBytes.String()
}

func hasFailed(pr *v1alpha1.PipelineRun) string {
	if pr.Status.Conditions[0].Status == corev1.ConditionFalse {
		return pr.Status.Conditions[0].Message
	}
	return ""
}

type taskrunList []tkr

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

func (s taskrunList) Len() int      { return len(s) }
func (s taskrunList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s taskrunList) Less(i, j int) bool {
	return s[j].Status.StartTime.Before(s[i].Status.StartTime)
}

func DescribeTemplateForTaskList(cs *Clients) string {

	const (
		emptyMsg = "No tasks found"
		header   = "NAME\tAGE"
		body     = "%s\t%s\n"
	)

	log.Print("validating Task List command\n")
	clock := clockwork.NewFakeClockAt(time.Now())
	task := GetTaskList(cs)
	var tmplBytes bytes.Buffer
	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	if len(task.Items) == 0 {
		fmt.Fprintln(w, emptyMsg)
		w.Flush()
		return tmplBytes.String()
	}
	fmt.Fprintln(w, header)
	for _, task := range task.Items {
		fmt.Fprintf(w, body,
			task.Name,
			formatted.Age(&task.CreationTimestamp, clock),
		)
	}
	w.Flush()
	return tmplBytes.String()
}

// const describeTemplateForTaskRunList = `{{- $l := len .Taskrun.Items }}{{ if eq $l 0 }}
// No taskruns found
// {{- else }}
// NAME	STARTED      DURATION     STATUS
// {{- range $i, $tr := .Taskrun.Items }}
// {{$tr.Name }}	{{ formatAge $tr.Status.StartTime  $.Params }}	{{ formatDuration $tr.Status.StartTime $tr.Status.CompletionTime }}	{{ index $tr.Status.Conditions | formatCondition }}
// {{- end }}
// {{- end }}
// `

func DescribeTemplateForTaskRunList(cs *Clients) string {

	const (
		emptyMsg = "No taskruns found"
		header   = "NAME\tSTARTED\tDURATION\tSTATUS\t"
		body     = "%s\t%s\t%s\t%s\t\n"
	)

	clock := clockwork.NewFakeClockAt(time.Now())
	taskrun := GetTaskRunList(cs)
	sort.Sort(byStartTime(taskrun.Items))

	var tmplBytes bytes.Buffer
	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	if len(taskrun.Items) == 0 {
		fmt.Fprintln(w, emptyMsg)
		w.Flush()
		return tmplBytes.String()
	}
	fmt.Fprintln(w, header)
	for _, tr := range taskrun.Items {
		fmt.Fprintf(w, body,
			tr.Name,
			formatted.Age(tr.Status.StartTime, clock),
			formatted.Duration(tr.Status.StartTime, tr.Status.CompletionTime),
			formatted.Condition(tr.Status.Conditions),
		)
	}
	w.Flush()
	return tmplBytes.String()
}

type pipelineruns map[string]v1alpha1.PipelineRun

const pipelineslistTemplate = `{{- $pl := len .Pipelines.Items }}{{ if eq $pl 0 -}}
No pipelines
{{- else -}}
NAME	AGE	LAST RUN	STARTED	DURATION	STATUS
{{- range $_, $p := .Pipelines.Items }}
{{- $pr := accessMap $.PipelineRuns $p.Name }}
{{- if $pr }}
{{ $p.Name }}	{{ formatAge $p.CreationTimestamp $.Params }}	{{ $pr.Name }}	{{ formatAge $pr.Status.StartTime $.Params }}	{{ formatDuration $pr.Status.StartTime $pr.Status.CompletionTime }}	{{ index $pr.Status.Conditions | formatCondition }}
{{- else }}
{{ $p.Name }}	{{ formatAge $p.CreationTimestamp $.Params }}	---	---	---	---
{{- end }}
{{- end }}
{{- end }}
`

func DescribeTemplateForPipelineList(cs *Clients) string {
	log.Print("validating Pipelines List command\n")
	clock := clockwork.NewFakeClockAt(time.Now())
	ps, prs, err := listPipelineDetails(cs)
	if err != nil {
		log.Println("Failed to list pipelines")
	}
	var data = struct {
		Pipelines    *v1alpha1.PipelineList
		PipelineRuns pipelineruns
		Params       clockwork.Clock
	}{
		Pipelines:    ps,
		PipelineRuns: prs,
		Params:       clock,
	}

	funcMap := template.FuncMap{
		"accessMap": func(prs pipelineruns, name string) *v1alpha1.PipelineRun {
			if pr, ok := prs[name]; ok {
				return &pr
			}

			return nil
		},
		"formatAge":       formatted.Age,
		"formatDuration":  formatted.Duration,
		"formatCondition": formatted.Condition,
	}

	t := template.Must(template.New("Pipelines List").Funcs(funcMap).Parse(pipelineslistTemplate))

	var tmplBytes bytes.Buffer

	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	err1 := t.Execute(w, data)
	if err1 != nil {
		panic(err1)
	}

	w.Flush()
	return tmplBytes.String()
}

func listPipelineDetails(cs *Clients) (*v1alpha1.PipelineList, pipelineruns, error) {
	ps := GetPipelineList(cs)

	if len(ps.Items) == 0 {
		return ps, pipelineruns{}, nil
	}

	runs := GetPipelineRunList(cs)

	latestRuns := pipelineruns{}

	for _, run := range runs.Items {
		pipelineName := run.Spec.PipelineRef.Name
		latest, ok := latestRuns[pipelineName]
		if !ok {
			latestRuns[pipelineName] = run
			continue
		}
		if run.CreationTimestamp.After(latest.CreationTimestamp.Time) {
			latestRuns[pipelineName] = run
		}
	}

	return ps, latestRuns, nil
}

func DescribeTemplateForPipelineResourceList(cs *Clients) string {

	const (
		emptyMsg = "No pipelineresources found."
		header   = "NAME\tTYPE\tDETAILS"
		body     = "%s\t%s\t%s\n"
	)

	log.Print("validating Pipeline Resources List command\n")

	pipelineResourcelist := GetPipelineResourceList(cs)

	var tmplBytes bytes.Buffer
	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	if len(pipelineResourcelist.Items) == 0 {
		fmt.Fprintln(w, emptyMsg)
		w.Flush()
		return tmplBytes.String()
	}
	fmt.Fprintln(w, header)
	for _, pre := range pipelineResourcelist.Items {
		fmt.Fprintf(w, body,
			pre.Name,
			pre.Spec.Type,
			details(pre),
		)
	}
	w.Flush()

	return tmplBytes.String()
}

const describeTemplateForPipelinesResources = `Name:	{{ .PipelineResource.Name }}
Namespace:	{{ .PipelineResource.Namespace }}
PipelineResource Type:	{{ .PipelineResource.Spec.Type }}

Params
{{- $l := len .PipelineResource.Spec.Params }}{{ if eq $l 0 }}
No params
{{- else }}
NAME	VALUE
{{- range $i, $p := .PipelineResource.Spec.Params }}
{{ $p.Name }}	{{ $p.Value }}
{{- end }}
{{- end }}

Secret Params
{{- $l := len .PipelineResource.Spec.SecretParams }}{{ if eq $l 0 }}
No secret params
{{- else }}
FIELDNAME	SECRETNAME
{{- range $i, $p := .PipelineResource.Spec.SecretParams }}
{{ $p.FieldName }}	{{ $p.SecretName }}
{{- end }}
{{- end }}
`

func DescribeTemplateForPipelinesResources(cs *Clients, prname string) string {
	log.Printf("validating  PipelineResource: %s describe command\n", prname)
	clock := clockwork.NewFakeClockAt(time.Now())
	pipelineResource := GetPipelineResource(cs, prname)
	var data = struct {
		PipelineResource *v1alpha1.PipelineResource
		Params           clockwork.Clock
	}{
		PipelineResource: pipelineResource,
		Params:           clock,
	}

	funcMap := template.FuncMap{}

	t := template.Must(template.New("Describe PipelineResource").Funcs(funcMap).Parse(describeTemplateForPipelinesResources))

	var tmplBytes bytes.Buffer

	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	err1 := t.Execute(w, data)
	if err1 != nil {
		panic(err1)
	}

	w.Flush()
	return tmplBytes.String()
}

func DescribeTemplateForPipelineRunList(cs *Clients) string {

	const (
		emptyMsg = "No pipelineruns found"
		header   = "NAME\tSTARTED\tDURATION\tSTATUS\t"
		body     = "%s\t%s\t%s\t%s\t\n"
	)

	log.Print("validating PipelineRun List command\n")
	clock := clockwork.NewFakeClockAt(time.Now())
	pipelinerunlst := GetPipelineRunList(cs)

	var tmplBytes bytes.Buffer
	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	if len(pipelinerunlst.Items) == 0 {
		fmt.Fprintln(w, emptyMsg)
		w.Flush()
		return tmplBytes.String()
	}
	fmt.Fprintln(w, header)
	for _, pr := range pipelinerunlst.Items {
		fmt.Fprintf(w, body,
			pr.Name,
			formatted.Age(pr.Status.StartTime, clock),
			formatted.Duration(pr.Status.StartTime, pr.Status.CompletionTime),
			formatted.Condition(pr.Status.Conditions),
		)
	}
	w.Flush()
	return tmplBytes.String()
}

func CreateTemplateResourcesForOutputpath(obj interface{}) string {
	const (
		emptyMsg = ""
		body     = "%s\n"
	)
	var tmplBytes bytes.Buffer
	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	switch obj.(type) {
	case *v1alpha1.TaskList:

		if len(obj.(*v1alpha1.TaskList).Items) == 0 {

			return emptyMsg
		}

		for _, r := range obj.(*v1alpha1.TaskList).Items {
			fmt.Fprintf(w, body,
				r.Name,
			)
		}
		w.Flush()
		return tmplBytes.String()
	case *v1alpha1.TaskRunList:
		//sort by start Time
		sort.Sort(byStartTime(obj.(*v1alpha1.TaskRunList).Items))
		if len(obj.(*v1alpha1.TaskRunList).Items) == 0 {

			return emptyMsg
		}

		for _, r := range obj.(*v1alpha1.TaskRunList).Items {
			fmt.Fprintf(w, body,
				r.Name,
			)
		}
		w.Flush()
		return tmplBytes.String()
	case *v1alpha1.PipelineList:
		if len(obj.(*v1alpha1.PipelineList).Items) == 0 {
			return emptyMsg
		}

		for _, r := range obj.(*v1alpha1.PipelineList).Items {
			fmt.Fprintf(w, body,
				r.Name,
			)
		}
		w.Flush()
		return tmplBytes.String()

	case *v1alpha1.PipelineRunList:
		if len(obj.(*v1alpha1.PipelineRunList).Items) == 0 {
			return emptyMsg
		}

		for _, r := range obj.(*v1alpha1.PipelineRunList).Items {
			fmt.Fprintf(w, body,
				r.Name,
			)
		}
		w.Flush()
		return tmplBytes.String()
	case *v1alpha1.PipelineResourceList:
		if len(obj.(*v1alpha1.PipelineResourceList).Items) == 0 {
			return emptyMsg
		}

		for _, r := range obj.(*v1alpha1.PipelineResourceList).Items {
			fmt.Fprintf(w, body,
				r.Name,
			)
		}
		w.Flush()
		return tmplBytes.String()
	case *v1alpha1.ClusterTaskList:
		if len(obj.(*v1alpha1.ClusterTaskList).Items) == 0 {
			return emptyMsg
		}

		for _, r := range obj.(*v1alpha1.ClusterTaskList).Items {
			fmt.Fprintf(w, body,
				r.Name,
			)
		}
		w.Flush()
		return tmplBytes.String()
	}

	return ""

}

type byStartTime []v1alpha1.TaskRun

func (s byStartTime) Len() int           { return len(s) }
func (s byStartTime) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byStartTime) Less(i, j int) bool { return s[j].Status.StartTime.Before(s[i].Status.StartTime) }

func details(pre v1alpha1.PipelineResource) string {
	var key = "url"
	if pre.Spec.Type == v1alpha1.PipelineResourceTypeStorage {
		key = "Location"
	}

	for _, p := range pre.Spec.Params {
		if p.Name == key {
			return p.Name + ": " + p.Value
		}
	}

	return "---"
}

func GetTask(c *Clients, name string) *v1alpha1.Task {

	task, err := c.TaskClient.Get(name, metav1.GetOptions{})
	// 	require.Nil(t, err)
	if err != nil {
		log.Fatalf("Couldn't get expected task  %s", err)
	}

	return task
}

func GetTaskList(c *Clients) *v1alpha1.TaskList {

	tasklist, err := c.TaskClient.List(metav1.ListOptions{})
	// 	require.Nil(t, err)
	if err != nil {
		log.Fatalf("Couldn't get expected tasklist  %s", err)
	}

	return tasklist
}

func GetTaskRun(c *Clients, name string) *v1alpha1.TaskRun {

	taskRun, err := c.TaskRunClient.Get(name, metav1.GetOptions{})
	// 	require.Nil(t, err)
	if err != nil {
		log.Fatalf("Couldn't get expected taskRun  %s", err)
	}

	return taskRun
}

func GetTaskRunList(c *Clients) *v1alpha1.TaskRunList {

	taskRunlist, err := c.TaskRunClient.List(metav1.ListOptions{})
	// 	require.Nil(t, err)
	if err != nil {
		log.Fatalf("Couldn't get expected taskRunlist  %s", err)
	}

	return taskRunlist
}

func GetPipelineResource(c *Clients, name string) *v1alpha1.PipelineResource {

	pipelineResource, err := c.PipelineResourceClient.Get(name, metav1.GetOptions{})
	// 	require.Nil(t, err)
	if err != nil {
		log.Fatalf("Couldn't get expected pipelineResource  %s", err)
	}

	return pipelineResource
}

func GetPipelineResourceList(c *Clients) *v1alpha1.PipelineResourceList {

	pipelineResourceList, err := c.PipelineResourceClient.List(metav1.ListOptions{})
	// 	require.Nil(t, err)
	if err != nil {
		log.Fatalf("Couldn't get expected pipelineResourceList  %s", err)
	}

	return pipelineResourceList
}

func GetPipeline(c *Clients, name string) *v1alpha1.Pipeline {

	pipeline, err := c.PipelineClient.Get(name, metav1.GetOptions{})
	// 	require.Nil(t, err)
	if err != nil {
		log.Fatalf("Couldn't get expected pipeline  %s", err)
	}

	return pipeline
}

func GetPipelineList(c *Clients) *v1alpha1.PipelineList {

	pipelineList, err := c.PipelineClient.List(metav1.ListOptions{})
	// 	require.Nil(t, err)
	if err != nil {
		log.Fatalf("Couldn't get expected pipelineList  %s", err)
	}

	return pipelineList
}

func GetPipelineRun(c *Clients, name string) *v1alpha1.PipelineRun {

	pipelineRun, err := c.PipelineRunClient.Get(name, metav1.GetOptions{})
	// 	require.Nil(t, err)
	if err != nil {
		log.Fatalf("Couldn't get expected pipelineRun  %s", err)
	}

	return pipelineRun
}

func GetPipelineRunListWithName(c *Clients, pname string) *v1alpha1.PipelineRunList {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/pipeline=%s", pname),
	}
	pipelineRunList, err := c.PipelineRunClient.List(opts)
	// 	require.Nil(t, err)
	if err != nil {
		log.Fatalf("Couldn't get expected pipelineRunList  %s", err)
	}

	return pipelineRunList
}

func GetPipelineRunList(c *Clients) *v1alpha1.PipelineRunList {

	pipelineRunList, err := c.PipelineRunClient.List(metav1.ListOptions{})
	// 	require.Nil(t, err)
	if err != nil {
		log.Fatalf("Couldn't get expected pipelineRunList  %s", err)
	}

	return pipelineRunList
}

////////////////////TestData//////////////////////
type TaskData struct {
	Name string
}

func CreateTemplateForTaskListWithTestData(t *testing.T, cs *Clients, td map[int]interface{}) string {
	t.Helper()
	const (
		emptyMsg = "No tasks found"
		header   = "NAME\tAGE"
		body     = "%s\t%s\n"
	)

	clock := clockwork.NewFakeClockAt(time.Now())

	task := GetTaskListWithTestData(t, cs, td)

	var tmplBytes bytes.Buffer
	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	if len(task.Items) == 0 {
		fmt.Fprintln(w, emptyMsg)
		w.Flush()
		return tmplBytes.String()
	}
	fmt.Fprintln(w, header)

	for _, task := range task.Items {
		fmt.Fprintf(w, body,
			task.Name,
			formatted.Age(&task.CreationTimestamp, clock),
		)
	}
	w.Flush()
	return tmplBytes.String()
}

func GetTaskListWithTestData(t *testing.T, c *Clients, td map[int]interface{}) *v1alpha1.TaskList {
	t.Helper()

	tasklist := GetTaskList(c)

	if len(tasklist.Items) != len(td) {
		t.Errorf("Lenght of task list and Testdata provided not matching")
	}
	if len(tasklist.Items) == 0 {
		return tasklist
	}
	for i, task := range td {
		switch task.(type) {
		case *TaskData:
			tasklist.Items[i].Name = task.(*TaskData).Name
		default:
			t.Error("Test Data Format Didn't Match please do check Test Data which you passing")
		}
	}

	if changelog := cmp.Diff(tasklist, GetTaskList(c)); changelog != "" {
		t.Logf("Changes occured while performing diff operation %+v", changelog)
	}
	return tasklist
}

type TaskRunData struct {
	Name   string
	Status string
}

func CreateTemplateForTaskRunListWithMockData(t *testing.T, cs *Clients, td map[int]interface{}) string {

	const (
		emptyMsg = "No taskruns found"
		header   = "NAME\tSTARTED\tDURATION\tSTATUS\t"
		body     = "%s\t%s\t%s\t%s\t\n"
	)

	clock := clockwork.NewFakeClockAt(time.Now())
	taskrun := GetTaskRunListWithMockData(t, cs, td)

	sort.Sort(byStartTime(taskrun.Items))

	var tmplBytes bytes.Buffer
	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	if len(taskrun.Items) == 0 {
		fmt.Fprintln(w, emptyMsg)
		w.Flush()
		return tmplBytes.String()
	}

	fmt.Fprintln(w, header)
	for _, tr := range taskrun.Items {
		fmt.Fprintf(w, body,
			tr.Name,
			formatted.Age(tr.Status.StartTime, clock),
			formatted.Duration(tr.Status.StartTime, tr.Status.CompletionTime),
			formatted.Condition(tr.Status.Conditions),
		)
	}
	w.Flush()
	return tmplBytes.String()
}

func GetTaskRunListWithMockData(t *testing.T, c *Clients, td map[int]interface{}) *v1alpha1.TaskRunList {
	taskRunlist := GetTaskRunList(c) //c.TaskRunClient.List(metav1.ListOptions{})
	if len(taskRunlist.Items) != len(td) {
		t.Errorf("Lenght of taskrun list and Testdata provided not matching")
	}
	if len(taskRunlist.Items) == 0 {
		return taskRunlist
	}
	for i, tr := range td {
		switch tr.(type) {
		case *TaskRunData:
			match, _ := regexp.Compile(tr.(*TaskRunData).Name + ".*")
			if match.MatchString(taskRunlist.Items[i].Name) {
				taskRunlist.Items[i].Status.Conditions[0].Reason = tr.(*TaskRunData).Status
			} else {
				t.Errorf("TaskRun Name didnt match , Expected %s Got %s", tr.(*TaskRunData).Name, taskRunlist.Items[i].Name)
			}
		default:
			t.Errorf("Test Data Format Didn't Match please do check Test Data which you passing")
		}

	}

	if changelog := cmp.Diff(taskRunlist, GetTaskRunList(c)); changelog != "" {
		t.Logf("Changes occured while performing diff operation %+v", changelog)
	}
	return taskRunlist
}

//----------------Pipeline Resources -----------------------------

type PipelineResourcesData struct {
	Name    string
	Type    string
	Details string
}

func DescribeTemplateForPipelineResourceListWithMockData(cs *Clients, td map[int]interface{}) string {

	const (
		emptyMsg = "No pipelineresources found."
		header   = "NAME\tTYPE\tDETAILS"
		body     = "%s\t%s\t%s\n"
	)

	pipelineResourcelist := GetPipelineResourceListWithMockData(cs, td)

	var tmplBytes bytes.Buffer
	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	if len(pipelineResourcelist.Items) == 0 {
		fmt.Fprintln(w, emptyMsg)
		w.Flush()
		return tmplBytes.String()
	}
	fmt.Fprintln(w, header)
	for _, pre := range pipelineResourcelist.Items {
		fmt.Fprintf(w, body,
			pre.Name,
			pre.Spec.Type,
			details(pre),
		)
	}
	w.Flush()

	return tmplBytes.String()
}

func GetPipelineResourceListWithMockData(c *Clients, td map[int]interface{}) *v1alpha1.PipelineResourceList {

	pipelineResourceList, err := c.PipelineResourceClient.List(metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Couldn't get expected PipelineResourcelist  %s", err)
	}
	if len(pipelineResourceList.Items) != len(td) {
		log.Panic("Lenght of PipelineResources list and Testdata provided not matching")
	}
	if len(pipelineResourceList.Items) == 0 {
		return pipelineResourceList
	}

	var key = "url"
	for i, pr := range td {
		switch pr.(type) {
		case *PipelineResourcesData:

			//Mock Name
			pipelineResourceList.Items[i].Name = pr.(*PipelineResourcesData).Name

			//Mock Resource Type
			if pr.(*PipelineResourcesData).Type == "git" {
				pipelineResourceList.Items[i].Spec.Type = v1alpha1.PipelineResourceTypeGit
			} else if pr.(*PipelineResourcesData).Type == "storage" {
				pipelineResourceList.Items[i].Spec.Type = v1alpha1.PipelineResourceTypeStorage
				key = "Location"
			} else if pr.(*PipelineResourcesData).Type == "image" {
				pipelineResourceList.Items[i].Spec.Type = v1alpha1.PipelineResourceTypeImage
			} else if pr.(*PipelineResourcesData).Type == "cluster" {
				pipelineResourceList.Items[i].Spec.Type = v1alpha1.PipelineResourceTypeCluster
			} else if pr.(*PipelineResourcesData).Type == "pullRequest" {
				pipelineResourceList.Items[i].Spec.Type = v1alpha1.PipelineResourceTypePullRequest
			} else if pr.(*PipelineResourcesData).Type == "build-gcs" {
				pipelineResourceList.Items[i].Spec.Type = v1alpha1.PipelineResourceTypeBuildGCS
			} else if pr.(*PipelineResourcesData).Type == "gcs" {
				pipelineResourceList.Items[i].Spec.Type = v1alpha1.PipelineResourceTypeGCS
			} else {
				log.Panicf("Provided PipelineResourcesData is not Valid Type : Need to Provide (%s, %s, %s, %s, %s)", v1alpha1.PipelineResourceTypeGit, v1alpha1.PipelineResourceTypeImage, v1alpha1.PipelineResourceTypePullRequest, v1alpha1.PipelineResourceTypeBuildGCS, v1alpha1.PipelineResourceTypeCluster)
			}

			for i, p := range pipelineResourceList.Items[i].Spec.Params {
				if p.Name == key {
					pipelineResourceList.Items[i].Spec.Params[i].Value = pr.(*PipelineResourcesData).Details
					break
				}
			}

		default:
			log.Panicf("Test Data Format Didn't Match please do check Test Data which you passing")
		}

	}

	if changelog := cmp.Diff(pipelineResourceList, GetPipelineResourceList(c)); changelog != "" {
		log.Printf("Changes occured while performing diff operation %+v", changelog)
	}
	return pipelineResourceList
}

type PipelineResourcesDescribeData struct {
	Name                  string
	Namespace             string
	PipelineResource_Type string
	Params                map[string]string
	SecretParams          map[string]string
}

func DescribeTemplateForPipelinesResourcesWithMockData(cs *Clients, prname string, td map[int]interface{}) string {

	clock := clockwork.NewFakeClockAt(time.Now())
	pipelineResource := GetPipelineResourceWithMockData(cs, prname, td)
	var data = struct {
		PipelineResource *v1alpha1.PipelineResource
		Params           clockwork.Clock
	}{
		PipelineResource: pipelineResource,
		Params:           clock,
	}

	funcMap := template.FuncMap{}

	t := template.Must(template.New("Describe PipelineResource").Funcs(funcMap).Parse(describeTemplateForPipelinesResources))

	var tmplBytes bytes.Buffer

	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	err1 := t.Execute(w, data)
	if err1 != nil {
		panic(err1)
	}

	w.Flush()
	return tmplBytes.String()
}

func GetPipelineResourceWithMockData(c *Clients, name string, td map[int]interface{}) *v1alpha1.PipelineResource {

	pipelineResource, err := c.PipelineResourceClient.Get(name, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Couldn't get expected PipelineResourcelist  %s", err)
	}
	for _, pr := range td {

		switch pr.(type) {
		case *PipelineResourcesDescribeData:
			if pr.(*PipelineResourcesDescribeData).Name == name {
				pipelineResource.Name = pr.(*PipelineResourcesDescribeData).Name

				pipelineResource.Namespace = pr.(*PipelineResourcesDescribeData).Namespace

				if pr.(*PipelineResourcesDescribeData).PipelineResource_Type == "git" {
					pipelineResource.Spec.Type = v1alpha1.PipelineResourceTypeGit
				} else if pr.(*PipelineResourcesDescribeData).PipelineResource_Type == "storage" {
					pipelineResource.Spec.Type = v1alpha1.PipelineResourceTypeStorage
				} else if pr.(*PipelineResourcesDescribeData).PipelineResource_Type == "image" {
					pipelineResource.Spec.Type = v1alpha1.PipelineResourceTypeImage
				} else if pr.(*PipelineResourcesDescribeData).PipelineResource_Type == "cluster" {
					pipelineResource.Spec.Type = v1alpha1.PipelineResourceTypeCluster
				} else if pr.(*PipelineResourcesDescribeData).PipelineResource_Type == "pullRequest" {
					pipelineResource.Spec.Type = v1alpha1.PipelineResourceTypePullRequest
				} else if pr.(*PipelineResourcesDescribeData).PipelineResource_Type == "build-gcs" {
					pipelineResource.Spec.Type = v1alpha1.PipelineResourceTypeBuildGCS
				} else if pr.(*PipelineResourcesDescribeData).PipelineResource_Type == "gcs" {
					pipelineResource.Spec.Type = v1alpha1.PipelineResourceTypeGCS
				} else {
					log.Panicf("Provided PipelineResourcesData is not Valid Type : Need to Provide (%s, %s, %s, %s, %s)", v1alpha1.PipelineResourceTypeGit, v1alpha1.PipelineResourceTypeImage, v1alpha1.PipelineResourceTypePullRequest, v1alpha1.PipelineResourceTypeBuildGCS, v1alpha1.PipelineResourceTypeCluster)
				}

				if len(pr.(*PipelineResourcesDescribeData).Params) == len(pipelineResource.Spec.Params) {
					for i, _ := range pipelineResource.Spec.Params {
						pipelineResource.Spec.Params[i].Value = pr.(*PipelineResourcesDescribeData).Params[pipelineResource.Spec.Params[i].Name]
					}
				} else {
					log.Panicf("Pipeline Resources Params lenght didnt match...")
				}

				if len(pr.(*PipelineResourcesDescribeData).SecretParams) == len(pipelineResource.Spec.SecretParams) {

					for i, _ := range pipelineResource.Spec.SecretParams {
						pipelineResource.Spec.SecretParams[i].SecretName = pr.(*PipelineResourcesDescribeData).SecretParams[pipelineResource.Spec.SecretParams[i].FieldName]
					}
				} else {
					log.Panicf("Pipeline Resources Params lenght didnt match...")
				}
			} else {
				continue
			}

		default:
			log.Panicf("Test Data Format Didn't Match please do check Test Data which you passing")
		}

	}

	if changelog := cmp.Diff(pipelineResource, GetPipelineResource(c, name)); changelog != "" {
		log.Printf("Changes occured while performing diff operation %+v", changelog)
	}

	return pipelineResource
}

type PipelinesListData struct {
	Name   string
	Status []string
}

func DescribeTemplateForPipelineListWithMockData(cs *Clients, td map[int]interface{}) string {
	log.Print("validating Pipelines List command\n")
	clock := clockwork.NewFakeClockAt(time.Now())
	ps, prs, err := listPipelineDetailsWithMockData(cs, td)
	if err != nil {
		log.Println("Failed to list pipelines")
	}
	var data = struct {
		Pipelines    *v1alpha1.PipelineList
		PipelineRuns pipelineruns
		Params       clockwork.Clock
	}{
		Pipelines:    ps,
		PipelineRuns: prs,
		Params:       clock,
	}

	funcMap := template.FuncMap{
		"accessMap": func(prs pipelineruns, name string) *v1alpha1.PipelineRun {
			if pr, ok := prs[name]; ok {
				return &pr
			}

			return nil
		},
		"formatAge":       formatted.Age,
		"formatDuration":  formatted.Duration,
		"formatCondition": formatted.Condition,
	}

	t := template.Must(template.New("Pipelines List").Funcs(funcMap).Parse(pipelineslistTemplate))

	var tmplBytes bytes.Buffer

	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	err1 := t.Execute(w, data)
	if err1 != nil {
		panic(err1)
	}

	w.Flush()
	return tmplBytes.String()
}

func listPipelineDetailsWithMockData(cs *Clients, td map[int]interface{}) (*v1alpha1.PipelineList, pipelineruns, error) {
	ps := GetPipelineListWithMockData(cs, td)

	runs := GetPipelineRunList(cs)
	latestRuns := pipelineruns{}
	for _, p := range td {
		switch p.(type) {
		case *PipelinesListData:
			for _, run := range runs.Items {
				pipelineName := p.(*PipelinesListData).Name
				latest, ok := latestRuns[pipelineName]
				if !ok {
					run.Status.Conditions[0].Reason = p.(*PipelinesListData).Status[0]
					latestRuns[pipelineName] = run
					continue
				}
				if run.CreationTimestamp.After(latest.CreationTimestamp.Time) {
					run.Status.Conditions[0].Reason = p.(*PipelinesListData).Status[0]
					latestRuns[pipelineName] = run
				}
			}
		}
	}

	return ps, latestRuns, nil
}

func GetPipelineListWithMockData(c *Clients, td map[int]interface{}) *v1alpha1.PipelineList {

	ps, err := c.PipelineClient.List(metav1.ListOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected pipelineList  %s", err)
	}

	if len(ps.Items) == 0 {
		return ps
	}

	if len(ps.Items) != len(td) {
		log.Panic("Lenght of taskrun list and Testdata provided not matching")
	}

	for i, p := range td {
		switch p.(type) {
		case *PipelinesListData:
			ps.Items[i].Name = p.(*PipelinesListData).Name
		default:
			log.Panicf("Test Data Format Didn't Match please do check Test Data which you passing")
		}
	}

	if changelog := cmp.Diff(ps, GetPipelineList(c)); changelog != "" {
		log.Printf("Changes occured while performing diff operation %+v", changelog)
	}

	return ps
}

type PipelineDescribeData struct {
	Name      string
	Resources map[string]string
	Task      map[int]interface{}
	Runs      map[string]string
}

type TaskRefData struct {
	TaskName string
	TaskRef  string
	RunAfter []string
}

func DescribeTemplateForPipelinesWithMockData(cs *Clients, pname string, td map[int]interface{}) string {
	log.Printf("validating Pipeline : %s describe command\n", pname)
	clock := clockwork.NewFakeClockAt(time.Now())

	pipeline := GetPipelineWithMockData(cs, pname, td)
	pipelineRuns := GetPipelineRunListWithNameAndMockData(cs, pname, td)

	var data = struct {
		Pipeline     *v1alpha1.Pipeline
		PipelineRuns *v1alpha1.PipelineRunList
		PipelineName string
		Params       clockwork.Clock
	}{
		Pipeline:     pipeline,
		PipelineRuns: pipelineRuns,
		PipelineName: pname,
		Params:       clock,
	}

	funcMap := template.FuncMap{
		"formatAge":       formatted.Age,
		"formatDuration":  formatted.Duration,
		"formatCondition": formatted.Condition,
	}

	var tmplBytes bytes.Buffer

	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	t := template.Must(template.New("Describe Pipeline").Funcs(funcMap).Parse(describeTemplate))

	err1 := t.Execute(w, data)
	if err1 != nil {
		panic(err1)
	}

	w.Flush()
	return tmplBytes.String()

}

func GetPipelineWithMockData(c *Clients, name string, td map[int]interface{}) *v1alpha1.Pipeline {

	pipeline, err := c.PipelineClient.Get(name, metav1.GetOptions{})
	// 	require.Nil(t, err)
	if err != nil {
		log.Fatalf("Couldn't get expected pipeline  %s", err)
	}

	for _, p := range td {
		switch p.(type) {
		case *PipelineDescribeData:
			pipeline.Name = p.(*PipelineDescribeData).Name
			if len(pipeline.Spec.Resources) == len(p.(*PipelineDescribeData).Resources) {
				count := 0
				for k, v := range p.(*PipelineDescribeData).Resources {
					pipeline.Spec.Resources[count].Name = k
					if v == "git" {
						pipeline.Spec.Resources[count].Type = v1alpha1.PipelineResourceTypeGit
					} else if v == "storage" {
						pipeline.Spec.Resources[count].Type = v1alpha1.PipelineResourceTypeStorage

					} else if v == "image" {
						pipeline.Spec.Resources[count].Type = v1alpha1.PipelineResourceTypeImage
					} else if v == "cluster" {
						pipeline.Spec.Resources[count].Type = v1alpha1.PipelineResourceTypeCluster
					} else if v == "pullRequest" {
						pipeline.Spec.Resources[count].Type = v1alpha1.PipelineResourceTypePullRequest
					} else if v == "build-gcs" {
						pipeline.Spec.Resources[count].Type = v1alpha1.PipelineResourceTypeBuildGCS
					} else if v == "gcs" {
						pipeline.Spec.Resources[count].Type = v1alpha1.PipelineResourceTypeGCS
					} else {
						log.Panicf("Provided PipelineResourcesData is not Valid Type : Need to Provide (%s, %s, %s, %s, %s)", v1alpha1.PipelineResourceTypeGit, v1alpha1.PipelineResourceTypeImage, v1alpha1.PipelineResourceTypePullRequest, v1alpha1.PipelineResourceTypeBuildGCS, v1alpha1.PipelineResourceTypeCluster)
					}
					count++
				}
			} else {
				log.Fatalf("length of Resources didn't match with testdata for pipeline %s", p.(*PipelineDescribeData).Name)
			}

			if len(pipeline.Spec.Tasks) == len(p.(*PipelineDescribeData).Task) {

				for i, t := range p.(*PipelineDescribeData).Task {
					switch t.(type) {
					case *TaskRefData:
						pipeline.Spec.Tasks[i].Name = t.(*TaskRefData).TaskName
						pipeline.Spec.Tasks[i].TaskRef.Name = t.(*TaskRefData).TaskRef
						pipeline.Spec.Tasks[i].RunAfter = t.(*TaskRefData).RunAfter
					default:
						log.Panicf(" TaskRef Data type doesnt match with Expected Type Recheck your Test Data for pipeline %s", p.(*PipelineDescribeData).Name)
					}

				}
			} else {
				log.Fatalf("length of Task didn't match with testdata for pipeline %s", p.(*PipelineDescribeData).Name)
			}
		default:
			log.Panicf(" Pipeline Describe Data type doesnt match with Expected Type Recheck your Test Data for pipeline %s", p.(*PipelineDescribeData).Name)
		}
	}

	if changelog := cmp.Diff(pipeline, GetPipeline(c, name)); changelog != "" {
		log.Printf("Changes occured while performing diff operation %+v", changelog)
	}

	if err != nil {
		log.Fatalf("Couldn't get expected pipelineList  %s", err)
	}

	return pipeline
}

func GetPipelineRunListWithNameAndMockData(c *Clients, pname string, td map[int]interface{}) *v1alpha1.PipelineRunList {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/pipeline=%s", pname),
	}
	pipelineRunList, err := c.PipelineRunClient.List(opts)
	// 	require.Nil(t, err)
	if err != nil {
		log.Fatalf("Couldn't get expected pipelineRunList  %s", err)
	}
	if len(pipelineRunList.Items) == 0 {
		return pipelineRunList
	}

	for _, p := range td {
		switch p.(type) {
		case *PipelineDescribeData:

			if len(pipelineRunList.Items) == len(p.(*PipelineDescribeData).Runs) {
				count := 0
				for k, v := range p.(*PipelineDescribeData).Runs {
					pipelineRunList.Items[count].Name = k
					pipelineRunList.Items[count].Status.Conditions[0].Reason = v
					count++
				}
			} else {
				log.Fatalf("length of PipelineRuns didn't match with testdata for pipeline %s", p.(*PipelineDescribeData).Name)
			}

		default:
			log.Panicf(" Pipeline Describe Data type doesnt match with Expected Type Recheck your Test Data for pipeline %s", p.(*PipelineDescribeData).Name)
		}
	}

	if changelog := cmp.Diff(pipelineRunList, GetPipelineRunListWithName(c, pname)); changelog != "" {
		log.Printf("Changes occured while performing diff operation %+v", changelog)
	}

	return pipelineRunList
}
