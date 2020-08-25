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

package builder

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"testing"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/test/framework"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetTask(c *framework.Clients, name string) *v1alpha1.Task {

	task, err := c.TaskClient.Get(name, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Couldn't get expected task  %s", err)
	}

	return task
}

func GetClusterTask(c *framework.Clients, name string) *v1alpha1.ClusterTask {
	clustertask, err := c.ClusterTaskClient.Get(name, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Couldn't get expected clustertask  %s", err)
	}
	return clustertask
}

func GetTaskList(c *framework.Clients) *v1alpha1.TaskList {

	tasklist, err := c.TaskClient.List(metav1.ListOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected tasklist  %s", err)
	}

	return tasklist
}

func GetClusterTaskList(c *framework.Clients) *v1alpha1.ClusterTaskList {
	clustertasklist, err := c.ClusterTaskClient.List(metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Couldn't get expected clustertasklist  %s", err)
	}
	return clustertasklist
}

func GetTaskRun(c *framework.Clients, name string) *v1alpha1.TaskRun {

	taskRun, err := c.TaskRunClient.Get(name, metav1.GetOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected taskRun  %s", err)
	}

	return taskRun
}

func GetTaskRunList(c *framework.Clients) *v1alpha1.TaskRunList {
	taskRunlist, err := c.TaskRunClient.List(metav1.ListOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected taskRunlist  %s", err)
	}

	return taskRunlist
}

func GetPipelineResource(c *framework.Clients, name string) *v1alpha1.PipelineResource {

	pipelineResource, err := c.PipelineResourceClient.Get(name, metav1.GetOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected pipelineResource  %s", err)
	}

	return pipelineResource
}

func GetPipelineResourceList(c *framework.Clients) *v1alpha1.PipelineResourceList {

	pipelineResourceList, err := c.PipelineResourceClient.List(metav1.ListOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected pipelineResourceList  %s", err)
	}

	return pipelineResourceList
}

func GetPipeline(c *framework.Clients, name string) *v1alpha1.Pipeline {

	pipeline, err := c.PipelineClient.Get(name, metav1.GetOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected pipeline  %s", err)
	}

	return pipeline
}

func GetPipelineList(c *framework.Clients) *v1alpha1.PipelineList {

	pipelineList, err := c.PipelineClient.List(metav1.ListOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected pipelineList  %s", err)
	}

	return pipelineList
}

func GetPipelineRun(c *framework.Clients, name string) *v1alpha1.PipelineRun {

	pipelineRun, err := c.PipelineRunClient.Get(name, metav1.GetOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected pipelineRun  %s", err)
	}

	return pipelineRun
}

func GetPipelineRunListWithName(c *framework.Clients, pname string, sortByStartTime bool) *v1alpha1.PipelineRunList {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/pipeline=%s", pname),
	}
	pipelineRunList, err := c.PipelineRunClient.List(opts)

	if err != nil {
		log.Fatalf("Couldn't get expected pipelineRunList  %s", err)
	}

	if err == nil && sortByStartTime {
		SortByStartTimePipelineRun(pipelineRunList.Items)
	}

	return pipelineRunList
}

func GetTaskRunListWithName(c *framework.Clients, tname string, sortByStartTime bool) *v1alpha1.TaskRunList {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/task=%s", tname),
	}

	time.Sleep(1 * time.Second)
	taskRunList, err := c.TaskRunClient.List(opts)

	if err != nil {
		log.Fatalf("Couldn't get expected taskRunList  %s", err)
	}

	if err == nil && sortByStartTime {
		SortByStartTimeTaskRun(taskRunList.Items)
	}

	return taskRunList
}

func GetPipelineRunList(c *framework.Clients) *v1alpha1.PipelineRunList {

	pipelineRunList, err := c.PipelineRunClient.List(metav1.ListOptions{})

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
		SortByStartTimeTaskRun(obj.Items)

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

func ListAllTasksOutput(t *testing.T, cs *framework.Clients, td map[int]interface{}) string {
	t.Helper()
	const (
		emptyMsg = "No tasks found"
		header   = "NAME\tDESCRIPTION\tAGE"
		body     = "%s\t%s\t%s\n"
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
			formatted.FormatDesc(task.Spec.Description),
			formatted.Age(&task.CreationTimestamp, clock),
		)
	}
	w.Flush()
	return tmplBytes.String()
}

func ListAllClusterTasksOutput(t *testing.T, cs *framework.Clients, td map[int]interface{}) string {
	t.Helper()
	const (
		emptyMsg = "No clustertasks found"
		header   = "NAME\tDESCRIPTION\tAGE"
		body     = "%s\t%s\t%s\n"
	)

	clock := clockwork.NewFakeClockAt(time.Now())

	clustertask := GetClusterTaskListWithTestData(t, cs, td)

	var tmplBytes bytes.Buffer
	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	if len(clustertask.Items) == 0 {
		fmt.Fprintln(w, emptyMsg)
		w.Flush()
		return tmplBytes.String()
	}
	fmt.Fprintln(w, header)

	for _, clustertask := range clustertask.Items {
		fmt.Fprintf(w, body,
			clustertask.Name,
			formatted.FormatDesc(clustertask.Spec.Description),
			formatted.Age(&clustertask.CreationTimestamp, clock),
		)
	}
	w.Flush()
	return tmplBytes.String()
}

func GetTaskListWithTestData(t *testing.T, c *framework.Clients, td map[int]interface{}) *v1alpha1.TaskList {
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
		t.Logf("Changes occurred while performing diff operation %+v", changelog)
	}
	return tasklist
}

func GetClusterTaskListWithTestData(t *testing.T, c *framework.Clients, td map[int]interface{}) *v1alpha1.ClusterTaskList {
	t.Helper()

	clustertasklist := GetClusterTaskList(c)

	if len(clustertasklist.Items) != len(td) {
		t.Errorf("Length of task list and Testdata provided not matching")
	}
	if len(clustertasklist.Items) == 0 {
		return clustertasklist
	}
	for i, clustertask := range td {
		switch clustertask := clustertask.(type) {
		case *TaskData:
			clustertasklist.Items[i].Name = clustertask.Name
		default:
			t.Error("Test Data Format Didn't Match please do check Test Data which you passing")
		}
	}

	if changelog := cmp.Diff(clustertasklist, GetClusterTaskList(c)); changelog != "" {
		t.Logf("Changes occurred while performing diff operation %+v", changelog)
	}
	return clustertasklist
}

type TaskRunData struct {
	Name   string
	Status string
}

func ListAllTaskRunsOutput(t *testing.T, cs *framework.Clients, allnamespaces bool, td map[int]interface{}) string {

	const listTemplate = `{{- $trl := len .TaskRuns.Items }}{{ if eq $trl 0 -}}
No TaskRuns found
{{- else -}}
NAME	STARTED	DURATION	STATUS{{- if $.AllNamespaces }}	NAMESPACE{{- end }}
{{- range $_, $tr := .TaskRuns.Items }}
{{- if $tr }}
{{ $tr.Name }}	{{ formatAge $tr.Status.StartTime $.Time }}	{{ formatDuration $tr.Status.StartTime $tr.Status.CompletionTime }}	{{ formatCondition $tr.Status.Conditions }}{{- if $.AllNamespaces }}	{{ $tr.Namespace }}{{- end }}
{{- end }}
{{- end }}
{{- end }}
`
	clock := clockwork.NewFakeClockAt(time.Now())
	taskrun := GetTaskRunListWithTestData(t, cs, td)
	trslen := len(taskrun.Items)

	if trslen != 0 {
		SortByStartTimeTaskRun(taskrun.Items)
	}
	var data = struct {
		TaskRuns      *v1alpha1.TaskRunList
		Time          clockwork.Clock
		AllNamespaces bool
	}{
		TaskRuns:      taskrun,
		Time:          clock,
		AllNamespaces: allnamespaces,
	}

	funcMap := template.FuncMap{
		"formatAge":       formatted.Age,
		"formatDuration":  formatted.Duration,
		"formatCondition": formatted.Condition,
	}

	var tmplBytes bytes.Buffer
	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)
	tmp := template.Must(template.New("List TaskRuns").Funcs(funcMap).Parse(listTemplate))

	err := tmp.Execute(w, data)
	if err != nil {
		t.Errorf("Error: while parsing template %+v", err)
	}

	w.Flush()
	return tmplBytes.String()
}

func GetTaskRunListWithTestData(t *testing.T, c *framework.Clients, td map[int]interface{}) *v1alpha1.TaskRunList {
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
		t.Logf("Changes occurred while performing diff operation %+v", changelog)
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

func ListAllPipelinesOutput(t *testing.T, cs *framework.Clients, td map[int]interface{}) string {
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

func listPipelineDetailsWithTestData(t *testing.T, cs *framework.Clients, td map[int]interface{}) (*v1alpha1.PipelineList, pipelineruns, error) {
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

func GetPipelineListWithTestData(t *testing.T, c *framework.Clients, td map[int]interface{}) *v1alpha1.PipelineList {
	t.Helper()
	ps := GetPipelineList(c)

	if len(ps.Items) == 0 {
		return ps
	}

	if len(ps.Items) != len(td) {
		t.Error("Length of pipeline list and Testdata provided not matching")
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
		t.Logf("Changes occurred while performing diff operation %+v", changelog)
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

const describeTemplate = `{{decorate "bold" "Name"}}:	{{ .PipelineName }}
{{decorate "bold" "Namespace"}}:	{{ .Pipeline.Namespace }}
{{- if ne .Pipeline.Spec.Description "" }}
{{decorate "bold" "Description"}}:	{{ .Pipeline.Spec.Description }}
{{- end }}

{{decorate "resources" ""}}{{decorate "underline bold" "Resources\n"}}
{{- $rl := len .Pipeline.Spec.Resources }}{{ if eq $rl 0 }}
 No resources
{{- else }}
 NAME	TYPE
{{- range $i, $r := .Pipeline.Spec.Resources }}
 {{decorate "bullet" $r.Name }}	{{ $r.Type }}
{{- end }}
{{- end }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}
{{- $l := len .Pipeline.Spec.Params }}{{ if eq $l 0 }}
 No params
{{- else }}
 NAME	TYPE	DESCRIPTION	DEFAULT VALUE
{{- range $i, $p := .Pipeline.Spec.Params }}
{{- if not $p.Default }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ $p.Description }}	{{ "---" }}
{{- else }}
{{- if eq $p.Type "string" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ $p.Description }}	{{ $p.Default.StringVal }}
{{- else }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ $p.Description }}	{{ $p.Default.ArrayVal }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "results" ""}}{{decorate "underline bold" "Results\n"}}
{{- if eq (len .Pipeline.Spec.Results) 0 }}
 No results
{{- else }}
 NAME	DESCRIPTION
{{- range $result := .Pipeline.Spec.Results }}
 {{ decorate "bullet" $result.Name }}	{{ $result.Description }}
{{- end }}
{{- end }}

{{decorate "workspaces" ""}}{{decorate "underline bold" "Workspaces\n"}}
{{- if eq (len .Pipeline.Spec.Workspaces) 0 }}
 No workspaces
{{- else }}
 NAME	DESCRIPTION
{{- range $workspace := .Pipeline.Spec.Workspaces }}
 {{ decorate "bullet" $workspace.Name }}	{{ formatDesc $workspace.Description }}
{{- end }}
{{- end }}

{{decorate "tasks" ""}}{{decorate "underline bold" "Tasks\n"}}
{{- $tl := len .Pipeline.Spec.Tasks }}{{ if eq $tl 0 }}
 No tasks
{{- else }}
 NAME	TASKREF	RUNAFTER	TIMEOUT	CONDITIONS	PARAMS
{{- range $i, $t := .Pipeline.Spec.Tasks }}
 {{decorate "bullet" $t.Name }}	{{ $t.TaskRef.Name }}	{{ join $t.RunAfter ", " }}	{{ formatTimeout $t.Timeout }}	{{ formatTaskConditions $t.Conditions }}	{{ formatParam $t.Params $.Pipeline.Spec.Params }}
{{- end }}
{{- end }}

{{decorate "pipelineruns" ""}}{{decorate "underline bold" "PipelineRuns\n"}}
{{- $rl := len .PipelineRuns.Items }}{{ if eq $rl 0 }}
 No pipelineruns
{{- else }}
 NAME	STARTED	DURATION	STATUS
{{- range $i, $pr := .PipelineRuns.Items }}
 {{decorate "bullet" $pr.Name }}	{{ formatAge $pr.Status.StartTime $.Params }}	{{ formatDuration $pr.Status.StartTime $pr.Status.CompletionTime }}	{{ formatCondition $pr.Status.Conditions }}
{{- end }}
{{- end }}
`

func GetPipelineDescribeOutput(t *testing.T, cs *framework.Clients, pname string, td map[int]interface{}) string {

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
		"formatAge":            formatted.Age,
		"formatDuration":       formatted.Duration,
		"formatCondition":      formatted.Condition,
		"formatTimeout":        formatted.Timeout,
		"formatDesc":           formatted.FormatDesc,
		"formatParam":          formatted.Param,
		"decorate":             formatted.DecorateAttr,
		"join":                 strings.Join,
		"formatTaskConditions": formatted.TaskConditions,
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

func GetPipelineWithTestData(t *testing.T, c *framework.Clients, name string, td map[int]interface{}) *v1alpha1.Pipeline {
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
		t.Logf("Changes occurred while performing diff operation %+v", changelog)
	}

	return pipeline
}

func GetPipelineRunListWithNameAndTestData(t *testing.T, c *framework.Clients, pname string, td map[int]interface{}) *v1alpha1.PipelineRunList {
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

	if changelog := cmp.Diff(pipelineRunList, GetPipelineRunListWithName(c, pname, false)); changelog != "" {
		t.Logf("Changes occurred while performing diff operation %+v", changelog)
	}

	return pipelineRunList
}
