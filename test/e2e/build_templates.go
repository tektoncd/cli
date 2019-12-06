package e2e

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"testing"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jonboulle/clockwork"

	"github.com/tektoncd/cli/pkg/formatted"
	trhsort "github.com/tektoncd/cli/pkg/helper/taskrun/sort"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetTask(c *Clients, name string) *v1alpha1.Task {

	task, err := c.TaskClient.Get(name, metav1.GetOptions{})
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

func ListResourceNamesForJSONPath(obj interface{}) string {
	const (
		emptyMsg = ""
		body     = "%s\n"
	)
	var tmplBytes bytes.Buffer
	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	switch obj := obj.(type) {
	case *v1alpha1.TaskList:

		if len(obj.Items) == 0 {

			return emptyMsg
		}

		for _, r := range obj.Items {
			fmt.Fprintf(w, body,
				r.Name,
			)
		}
		w.Flush()
		return tmplBytes.String()
	case *v1alpha1.TaskRunList:
		if len(obj.Items) == 0 {

			return emptyMsg
		}
		//sort by start Time
		trslen := len(obj.Items)
		if trslen != 0 {
			obj.Items = trhsort.SortTaskRunsByStartTime(obj.Items)
		}

		for _, r := range obj.Items {
			fmt.Fprintf(w, body,
				r.Name,
			)
		}
		w.Flush()
		return tmplBytes.String()
	case *v1alpha1.PipelineList:
		if len(obj.Items) == 0 {
			return emptyMsg
		}

		for _, r := range obj.Items {
			fmt.Fprintf(w, body,
				r.Name,
			)
		}
		w.Flush()
		return tmplBytes.String()

	case *v1alpha1.PipelineRunList:
		if len(obj.Items) == 0 {
			return emptyMsg
		}

		for _, r := range obj.Items {
			fmt.Fprintf(w, body,
				r.Name,
			)
		}
		w.Flush()
		return tmplBytes.String()
	case *v1alpha1.PipelineResourceList:
		if len(obj.Items) == 0 {
			return emptyMsg
		}

		for _, r := range obj.Items {
			fmt.Fprintf(w, body,
				r.Name,
			)
		}
		w.Flush()
		return tmplBytes.String()
	case *v1alpha1.ClusterTaskList:
		if len(obj.Items) == 0 {
			return emptyMsg
		}

		for _, r := range obj.Items {
			fmt.Fprintf(w, body,
				r.Name,
			)
		}
		w.Flush()
		return tmplBytes.String()
	}

	return ""

}

type TaskData struct {
	Name string
}

func ListAllTasksOutput(t *testing.T, cs *Clients, td map[int]interface{}) string {
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
		t.Errorf("Length of task list and Testdata provided not matching")
	}
	if len(tasklist.Items) == 0 {
		return tasklist
	}
	for i, task := range td {
		switch task := task.(type) {
		case *TaskData:
			tasklist.Items[i].Name = task.Name
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

func ListAllTaskRunsOutput(t *testing.T, cs *Clients, td map[int]interface{}) string {

	const (
		emptyMsg = "No taskruns found"
		header   = "NAME\tSTARTED\tDURATION\tSTATUS\t"
		body     = "%s\t%s\t%s\t%s\t\n"
	)

	clock := clockwork.NewFakeClockAt(time.Now())
	taskrun := GetTaskRunListWithTestData(t, cs, td)

	trslen := len(taskrun.Items)
	if trslen != 0 {
		taskrun.Items = trhsort.SortTaskRunsByStartTime(taskrun.Items)
	}

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

func GetTaskRunListWithTestData(t *testing.T, c *Clients, td map[int]interface{}) *v1alpha1.TaskRunList {
	taskRunlist := GetTaskRunList(c)
	if len(taskRunlist.Items) != len(td) {
		t.Errorf("Length of taskrun list and Testdata provided not matching")
	}
	if len(taskRunlist.Items) == 0 {
		return taskRunlist
	}
	for i, tr := range td {
		switch tr := tr.(type) {
		case *TaskRunData:
			match, _ := regexp.Compile(tr.Name + ".*")
			if match.MatchString(taskRunlist.Items[i].Name) {
				taskRunlist.Items[i].Status.Conditions[0].Reason = tr.Status
			} else {
				t.Errorf("TaskRun Name didnt match , Expected %s Got %s", tr.Name, taskRunlist.Items[i].Name)
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

type pipelineruns map[string]v1alpha1.PipelineRun

type PipelinesListData struct {
	Name   string
	Status string
}

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

func ListAllPipelinesOutput(t *testing.T, cs *Clients, td map[int]interface{}) string {
	t.Helper()
	t.Log("validating Pipelines List command\n")
	clock := clockwork.NewFakeClockAt(time.Now())
	ps, prs, err := listPipelineDetailsWithTestData(t, cs, td)
	if err != nil {
		t.Error("Failed to list pipelines")
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

	tmp := template.Must(template.New("Pipelines List").Funcs(funcMap).Parse(pipelineslistTemplate))

	var tmplBytes bytes.Buffer

	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	err1 := tmp.Execute(w, data)
	if err1 != nil {
		panic(err1)
	}

	w.Flush()
	return tmplBytes.String()
}

func listPipelineDetailsWithTestData(t *testing.T, cs *Clients, td map[int]interface{}) (*v1alpha1.PipelineList, pipelineruns, error) {
	t.Helper()
	ps := GetPipelineListWithTestData(t, cs, td)
	runs := GetPipelineRunList(cs)
	latestRuns := pipelineruns{}
	for _, p := range td {
		if _, ok := p.(*PipelinesListData); ok {
			for _, run := range runs.Items {
				pipelineName := p.(*PipelinesListData).Name
				latest, ok := latestRuns[pipelineName]
				if !ok {
					run.Status.Conditions[0].Reason = p.(*PipelinesListData).Status
					latestRuns[pipelineName] = run
					continue
				}
				if run.CreationTimestamp.After(latest.CreationTimestamp.Time) {
					run.Status.Conditions[0].Reason = p.(*PipelinesListData).Status
					latestRuns[pipelineName] = run
				}
			}
		}
	}

	return ps, latestRuns, nil
}

func GetPipelineListWithTestData(t *testing.T, c *Clients, td map[int]interface{}) *v1alpha1.PipelineList {
	t.Helper()
	ps := GetPipelineList(c)

	if len(ps.Items) == 0 {
		return ps
	}

	if len(ps.Items) != len(td) {
		t.Error("Lenght of pipeline list and Testdata provided not matching")
	}

	for i, p := range td {
		switch p := p.(type) {
		case *PipelinesListData:
			ps.Items[i].Name = p.Name
		default:
			t.Error("Test Data Format Didn't Match please do check Test Data which you passing")
		}
	}

	if changelog := cmp.Diff(ps, GetPipelineList(c)); changelog != "" {
		t.Logf("Changes occured while performing diff operation %+v", changelog)
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

Params
{{- $l := len .Pipeline.Spec.Params }}{{ if eq $l 0 }}
No params
{{- else }}
NAME	TYPE	DEFAULT VALUE
{{- range $i, $p := .Pipeline.Spec.Params }}
{{- if not $p.Default }}
{{ $p.Name }}	{{ $p.Type }}	{{ "" }}
{{- else }}
{{- if eq $p.Type "string" }}
{{ $p.Name }}	{{ $p.Type }}	{{ $p.Default.StringVal }}
{{- else }}
{{ $p.Name }}	{{ $p.Type }}	{{ $p.Default.ArrayVal }}
{{- end }}
{{- end }}
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

Pipelineruns
{{- $rl := len .PipelineRuns.Items }}{{ if eq $rl 0 }}
No pipelineruns
{{- else }}
NAME	STARTED	DURATION	STATUS
{{- range $i, $pr := .PipelineRuns.Items }}
{{ $pr.Name }}	{{ formatAge $pr.Status.StartTime $.Params }}	{{ formatDuration $pr.Status.StartTime $pr.Status.CompletionTime }}	{{ formatCondition $pr.Status.Conditions }}
{{- end }}
{{- end }}
`

func GetPipelineDescribeOutput(t *testing.T, cs *Clients, pname string, td map[int]interface{}) string {

	t.Helper()
	clock := clockwork.NewFakeClockAt(time.Now())

	pipeline := GetPipelineWithTestData(t, cs, pname, td)
	if len(pipeline.Spec.Resources) > 0 {
		pipeline.Spec.Resources = SortResourcesByTypeAndName(pipeline.Spec.Resources)
	}
	pipelineRuns := GetPipelineRunListWithNameAndTestData(t, cs, pname, td)

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

	tmp := template.Must(template.New("Describe Pipeline").Funcs(funcMap).Parse(describeTemplate))

	err1 := tmp.Execute(w, data)
	if err1 != nil {
		panic(err1)
	}

	w.Flush()
	return tmplBytes.String()

}

func GetPipelineWithTestData(t *testing.T, c *Clients, name string, td map[int]interface{}) *v1alpha1.Pipeline {
	t.Helper()
	pipeline := GetPipeline(c, name)

	for _, p := range td {
		switch p := p.(type) {
		case *PipelineDescribeData:
			pipeline.Name = p.Name
			if len(pipeline.Spec.Resources) == len(p.Resources) {
				count := 0
				for k, v := range p.Resources {
					pipeline.Spec.Resources[count].Name = k
					switch v {
					case "git":
						pipeline.Spec.Resources[count].Type = v1alpha1.PipelineResourceTypeGit
					case "storage":
						pipeline.Spec.Resources[count].Type = v1alpha1.PipelineResourceTypeStorage
					case "image":
						pipeline.Spec.Resources[count].Type = v1alpha1.PipelineResourceTypeImage
					case "cluster":
						pipeline.Spec.Resources[count].Type = v1alpha1.PipelineResourceTypeCluster
					case "pullRequest":
						pipeline.Spec.Resources[count].Type = v1alpha1.PipelineResourceTypePullRequest
					case "build-gcs":
						pipeline.Spec.Resources[count].Type = v1alpha1.PipelineResourceTypeBuildGCS
					case "gcs":
						pipeline.Spec.Resources[count].Type = v1alpha1.PipelineResourceTypeGCS
					default:
						t.Errorf("Provided PipelineResourcesData is not Valid Type : Need to Provide (%s, %s, %s, %s, %s)", v1alpha1.PipelineResourceTypeGit, v1alpha1.PipelineResourceTypeImage, v1alpha1.PipelineResourceTypePullRequest, v1alpha1.PipelineResourceTypeBuildGCS, v1alpha1.PipelineResourceTypeCluster)
					}

					count++
				}
			} else {
				t.Errorf("length of Resources didn't match with testdata for pipeline %s", p.Name)
			}

			if len(pipeline.Spec.Tasks) == len(p.Task) {

				for i, tref := range p.Task {
					switch tref := tref.(type) {
					case *TaskRefData:
						pipeline.Spec.Tasks[i].Name = tref.TaskName
						pipeline.Spec.Tasks[i].TaskRef.Name = tref.TaskRef
						pipeline.Spec.Tasks[i].RunAfter = tref.RunAfter
					default:
						t.Errorf(" TaskRef Data type doesnt match with Expected Type Recheck your Test Data for pipeline %s", p.Name)
					}
				}
			} else {
				t.Errorf("length of Task didn't match with testdata for pipeline %s", p.Name)
			}
		default:
			t.Errorf(" Pipeline Describe Data type doesnt match with Expected Type Recheck your Test Data for pipeline %s", p.(*PipelineDescribeData).Name)
		}
	}

	if changelog := cmp.Diff(pipeline, GetPipeline(c, name)); changelog != "" {
		t.Logf("Changes occured while performing diff operation %+v", changelog)
	}

	return pipeline
}

func GetPipelineRunListWithNameAndTestData(t *testing.T, c *Clients, pname string, td map[int]interface{}) *v1alpha1.PipelineRunList {
	t.Helper()
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/pipeline=%s", pname),
	}
	pipelineRunList, err := c.PipelineRunClient.List(opts)
	if err != nil {
		t.Errorf("Couldn't get expected pipelineRunList  %s", err)
	}
	if len(pipelineRunList.Items) == 0 {
		return pipelineRunList
	}

	for _, p := range td {
		switch p := p.(type) {
		case *PipelineDescribeData:

			if len(pipelineRunList.Items) == len(p.Runs) {
				count := 0
				for k, v := range p.Runs {
					pipelineRunList.Items[count].Name = k
					pipelineRunList.Items[count].Status.Conditions[0].Reason = v
					count++
				}
			} else {
				t.Errorf("length of PipelineRuns didn't match with testdata for pipeline %s", p.Name)
			}

		default:
			t.Errorf(" Pipeline Describe Data type doesnt match with Expected Type Recheck your Test Data for pipeline %s", p.(*PipelineDescribeData).Name)
		}
	}

	if changelog := cmp.Diff(pipelineRunList, GetPipelineRunListWithName(c, pname)); changelog != "" {
		t.Logf("Changes occured while performing diff operation %+v", changelog)
	}

	return pipelineRunList
}
