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
	"context"
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
	pipelineCmd "github.com/tektoncd/cli/pkg/cmd/pipeline"
	taskrunCmd "github.com/tektoncd/cli/pkg/cmd/taskrun"
	"github.com/tektoncd/cli/pkg/formatted"
	pipelinerunpkg "github.com/tektoncd/cli/pkg/pipelinerun/sort"
	taskrunpkg "github.com/tektoncd/cli/pkg/taskrun/sort"
	"github.com/tektoncd/cli/test/framework"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetTask(c *framework.Clients, name string) *v1.Task {

	task, err := c.TaskClient.Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Couldn't get expected task  %s", err)
	}

	return task
}

func GetClusterTask(c *framework.Clients, name string) *v1beta1.ClusterTask {
	clustertask, err := c.ClusterTaskClient.Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Couldn't get expected clustertask  %s", err)
	}
	return clustertask
}

func GetTaskList(c *framework.Clients) *v1.TaskList {

	tasklist, err := c.TaskClient.List(context.Background(), metav1.ListOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected tasklist  %s", err)
	}

	return tasklist
}

func GetClusterTaskList(c *framework.Clients) *v1beta1.ClusterTaskList {
	clustertasklist, err := c.ClusterTaskClient.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Couldn't get expected clustertasklist  %s", err)
	}
	return clustertasklist
}

func GetTaskRun(c *framework.Clients, name string) *v1.TaskRun {

	taskRun, err := c.TaskRunClient.Get(context.Background(), name, metav1.GetOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected taskRun  %s", err)
	}

	return taskRun
}

func GetTaskRunList(c *framework.Clients) *v1.TaskRunList {
	taskRunlist, err := c.TaskRunClient.List(context.Background(), metav1.ListOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected taskRunlist  %s", err)
	}

	return taskRunlist
}

func GetPipeline(c *framework.Clients, name string) *v1.Pipeline {

	pipeline, err := c.PipelineClient.Get(context.Background(), name, metav1.GetOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected pipeline  %s", err)
	}

	return pipeline
}

func GetPipelineList(c *framework.Clients) *v1.PipelineList {

	pipelineList, err := c.PipelineClient.List(context.Background(), metav1.ListOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected pipelineList  %s", err)
	}

	return pipelineList
}

func GetPipelineRun(c *framework.Clients, name string) *v1.PipelineRun {

	pipelineRun, err := c.PipelineRunClient.Get(context.Background(), name, metav1.GetOptions{})

	if err != nil {
		log.Fatalf("Couldn't get expected pipelineRun  %s", err)
	}

	return pipelineRun
}

func GetPipelineRunListWithName(c *framework.Clients, pname string, sortByStartTime bool) *v1.PipelineRunList {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/pipeline=%s", pname),
	}
	pipelineRunList, err := c.PipelineRunClient.List(context.Background(), opts)

	if err != nil {
		log.Fatalf("Couldn't get expected pipelineRunList  %s", err)
	}

	if err == nil && sortByStartTime {
		pipelinerunpkg.SortByStartTime(pipelineRunList.Items)
	}

	return pipelineRunList
}

func GetTaskRunListByLabel(c *framework.Clients, sortByStartTime bool, label string) *v1.TaskRunList {
	opts := metav1.ListOptions{
		LabelSelector: label,
	}

	time.Sleep(1 * time.Second)
	taskRunList, err := c.TaskRunClient.List(context.Background(), opts)

	if err != nil {
		log.Fatalf("Couldn't get expected taskRunList  %s", err)
	}

	if err == nil && sortByStartTime {
		taskrunpkg.SortByStartTime(taskRunList.Items)
	}

	return taskRunList
}

func GetTaskRunListWithTaskName(c *framework.Clients, tname string, sortByStartTime bool) *v1.TaskRunList {
	label := fmt.Sprintf("tekton.dev/task=%s", tname)
	return GetTaskRunListByLabel(c, sortByStartTime, label)
}

func GetTaskRunListWithClusterTaskName(c *framework.Clients, ctname string, sortByStartTime bool) *v1.TaskRunList {
	label := fmt.Sprintf("tekton.dev/clusterTask=%s", ctname)
	return GetTaskRunListByLabel(c, sortByStartTime, label)
}

func GetPipelineRunList(c *framework.Clients) *v1.PipelineRunList {

	pipelineRunList, err := c.PipelineRunClient.List(context.Background(), metav1.ListOptions{})

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
	case *v1.TaskList:

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
	case *v1.TaskRunList:
		if len(obj.Items) == 0 {

			return emptyMsg
		}
		// sort by start Time
		taskrunpkg.SortByStartTime(obj.Items)

		for _, r := range obj.Items {
			fmt.Fprintf(w, body,
				r.Name,
			)
		}
		w.Flush()
		return tmplBytes.String()
	case *v1.PipelineList:
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

	case *v1.PipelineRunList:
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
	case *v1beta1.ClusterTaskList:
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

	tasks := task.Items

	for idx := range tasks {
		fmt.Fprintf(w, body,
			tasks[idx].Name,
			formatted.FormatDesc(tasks[idx].Spec.Description),
			formatted.Age(&tasks[idx].CreationTimestamp, clock),
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

	clusterTasks := clustertask.Items

	for idx := range clusterTasks {
		fmt.Fprintf(w, body,
			clusterTasks[idx].Name,
			formatted.FormatDesc(clusterTasks[idx].Spec.Description),
			formatted.Age(&clusterTasks[idx].CreationTimestamp, clock),
		)
	}
	w.Flush()
	return tmplBytes.String()
}

func GetTaskListWithTestData(t *testing.T, c *framework.Clients, td map[int]interface{}) *v1.TaskList {
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

func GetClusterTaskListWithTestData(t *testing.T, c *framework.Clients, td map[int]interface{}) *v1beta1.ClusterTaskList {
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

func ListAllTaskRunsOutput(t *testing.T, cs *framework.Clients, allnamespaces, noheaders bool, td map[int]interface{}) string {
	clock := clockwork.NewFakeClockAt(time.Now())
	taskrun := GetTaskRunListWithTestData(t, cs, td)
	trslen := len(taskrun.Items)

	if trslen != 0 {
		taskrunpkg.SortByStartTime(taskrun.Items)
	}
	var data = struct {
		TaskRuns      *v1.TaskRunList
		Time          clockwork.Clock
		AllNamespaces bool
		NoHeaders     bool
	}{
		TaskRuns:      taskrun,
		Time:          clock,
		AllNamespaces: allnamespaces,
		NoHeaders:     noheaders,
	}

	funcMap := template.FuncMap{
		"formatAge":       formatted.Age,
		"formatDuration":  formatted.Duration,
		"formatCondition": formatted.Condition,
	}

	var tmplBytes bytes.Buffer
	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)
	tmp := template.Must(template.New("List TaskRuns").Funcs(funcMap).Parse(taskrunCmd.ListTemplate))

	err := tmp.Execute(w, data)
	if err != nil {
		t.Errorf("Error: while parsing template %+v", err)
	}

	w.Flush()
	return tmplBytes.String()
}

func GetTaskRunListWithTestData(t *testing.T, c *framework.Clients, td map[int]interface{}) *v1.TaskRunList {
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

type pipelineruns map[string]v1.PipelineRun

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
		Pipelines    *v1.PipelineList
		PipelineRuns pipelineruns
		Params       clockwork.Clock
	}{
		Pipelines:    ps,
		PipelineRuns: prs,
		Params:       clock,
	}

	funcMap := template.FuncMap{
		"accessMap": func(prs pipelineruns, name string) *v1.PipelineRun {
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

func listPipelineDetailsWithTestData(t *testing.T, cs *framework.Clients, td map[int]interface{}) (*v1.PipelineList, pipelineruns, error) {
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

func GetPipelineListWithTestData(t *testing.T, c *framework.Clients, td map[int]interface{}) *v1.PipelineList {
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

func GetPipelineDescribeOutput(t *testing.T, cs *framework.Clients, pname string, td map[int]interface{}) string {

	t.Helper()
	clock := clockwork.NewFakeClockAt(time.Now())

	pipeline := GetPipelineWithTestData(t, cs, pname, td)
	pipelineRuns := GetPipelineRunListWithNameAndTestData(t, cs, pname, td)

	var data = struct {
		Pipeline     *v1.Pipeline
		PipelineRuns *v1.PipelineRunList
		PipelineName string
		Time         clockwork.Clock
	}{
		Pipeline:     pipeline,
		PipelineRuns: pipelineRuns,
		PipelineName: pname,
		Time:         clock,
	}

	funcMap := template.FuncMap{
		"formatAge":               formatted.Age,
		"formatDuration":          formatted.Duration,
		"formatCondition":         formatted.Condition,
		"formatTimeout":           formatted.Timeout,
		"formatDesc":              formatted.FormatDesc,
		"formatParam":             formatted.Param,
		"decorate":                formatted.DecorateAttr,
		"join":                    strings.Join,
		"getTaskRefName":          formatted.GetTaskRefName,
		"removeLastAppliedConfig": formatted.RemoveLastAppliedConfig,
	}

	var tmplBytes bytes.Buffer

	w := tabwriter.NewWriter(&tmplBytes, 0, 5, 3, ' ', tabwriter.TabIndent)

	tmp := template.Must(template.New("Describe Pipeline").Funcs(funcMap).Parse(pipelineCmd.DescribeTemplate))

	err1 := tmp.Execute(w, data)
	if err1 != nil {
		panic(err1)
	}

	w.Flush()
	return tmplBytes.String()

}

func GetPipelineWithTestData(t *testing.T, c *framework.Clients, name string, td map[int]interface{}) *v1.Pipeline {
	t.Helper()
	pipeline := GetPipeline(c, name)

	for _, p := range td {
		switch p := p.(type) {
		case *PipelineDescribeData:
			pipeline.Name = p.Name
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

func GetPipelineRunListWithNameAndTestData(t *testing.T, c *framework.Clients, pname string, td map[int]interface{}) *v1.PipelineRunList {
	t.Helper()
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/pipeline=%s", pname),
	}
	pipelineRunList, err := c.PipelineRunClient.List(context.Background(), opts)
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
