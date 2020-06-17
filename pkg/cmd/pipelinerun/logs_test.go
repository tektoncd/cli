// Copyright © 2019 The Tekton Authors.
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
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/pods/fake"
	"github.com/tektoncd/cli/pkg/pods/stream"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

const (
	versionA1 = "v1alpha1"
	versionB1 = "v1beta1"
)

func TestLog_invalid_namespace(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client()
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipelinerun"})
	c := Command(p)

	out, err := test.ExecuteCommand(c, "logs", "pipelinerun", "-n", "invalid")

	if err == nil {
		t.Errorf("Expecting error for invalid namespace")
	}

	expected := "Error: pipelineruns.tekton.dev \"pipelinerun\" not found\n"
	test.AssertOutput(t, expected, out)
}

func TestLog_invalid_limit(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}
	c := Command(p)

	limit := "0"
	_, err := test.ExecuteCommand(c, "logs", "-n", "ns", "--limit", limit)

	if err == nil {
		t.Errorf("Expecting error for --limit option being <=0")
	}

	expected := "limit was " + limit + " but must be a positive number"
	test.AssertOutput(t, expected, err.Error())
}

func TestLog_no_pipelinerun_argument(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}
	c := Command(p)

	_, err := test.ExecuteCommand(c, "logs", "-n", "ns")

	if err == nil {
		t.Error("Expecting an error but it's empty")
	}
}

func TestLog_run_found(t *testing.T) {
	clock := clockwork.NewFakeClock()
	pdata := []*v1alpha1.Pipeline{
		tb.Pipeline("pipeline",
			tb.PipelineNamespace("ns"),
			cb.PipelineCreationTimestamp(clock.Now().Add(-15*time.Minute)),
		),
	}
	prdata := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipelinerun-1",
			tb.PipelineRunNamespace("ns"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines:    pdata,
		PipelineRuns: prdata,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(pdata[0], versionA1),
		cb.UnstructuredPR(prdata[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	c := Command(p)
	_, err = test.ExecuteCommand(c, "logs", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
}

func TestLog_run_not_found(t *testing.T) {
	pr := []*v1alpha1.PipelineRun{
		tb.PipelineRun("output-pipeline-1",
			tb.PipelineRunNamespace("ns"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "output-pipeline-1"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionFalse,
					Reason: resources.ReasonFailed,
				}),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: pr, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(pr[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	c := Command(p)
	_, err = test.ExecuteCommand(c, "logs", "output-pipeline-2", "-n", "ns")
	expected := "pipelineruns.tekton.dev \"output-pipeline-2\" not found"
	test.AssertOutput(t, expected, err.Error())
}

func TestPipelinerunLogs(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		prstart      = clockwork.NewFakeClock()
		ns           = "namespace"

		task1Name    = "output-task"
		tr1Name      = "output-task-1"
		tr1StartTime = prstart.Now().Add(20 * time.Second)
		tr1Pod       = "output-task-pod-123456"
		tr1Step1Name = "writefile-step"
		tr1InitStep1 = "credential-initializer-mdzbr"
		tr1InitStep2 = "place-tools"

		task2Name    = "read-task"
		tr2Name      = "read-task-1"
		tr2StartTime = prstart.Now().Add(2 * time.Minute)
		tr2Pod       = "read-task-pod-123456"
		tr2Step1Name = "readfile-step"

		nopStep = "nop"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun(tr1Name,
			tb.TaskRunNamespace(ns),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(task1Name),
			),
			tb.TaskRunStatus(
				tb.PodName(tr1Pod),
				tb.TaskRunStartTime(tr1StartTime),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName(tr1Step1Name),
					tb.StateTerminated(0),
				),
				tb.StepState(
					cb.StepName(nopStep),
					tb.StateTerminated(0),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(task1Name),
			),
		),
		tb.TaskRun(tr2Name,
			tb.TaskRunNamespace(ns),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(task2Name),
			),
			tb.TaskRunStatus(
				tb.PodName(tr2Pod),
				tb.TaskRunStartTime(tr2StartTime),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName(tr2Step1Name),
					tb.StateTerminated(0),
				),
				tb.StepState(
					cb.StepName(nopStep),
					tb.StateTerminated(0),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(task2Name),
			),
		),
	}

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunLabel("tekton.dev/pipeline", prName),
			tb.PipelineRunSpec(pipelineName),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.PipelineRunTaskRunsStatus(tr1Name, &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: task1Name,
					Status:           &trs[0].Status,
				}),
				tb.PipelineRunTaskRunsStatus(tr2Name, &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: task2Name,
					Status:           &trs[1].Status,
				}),
			),
		),
	}
	pps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName,
			tb.PipelineNamespace(ns),
			tb.PipelineSpec(
				tb.PipelineTask(task1Name, task1Name),
				tb.PipelineTask(task2Name, task2Name),
			),
		),
	}

	p := []*corev1.Pod{
		tb.Pod(tr1Pod,
			tb.PodNamespace(ns),
			tb.PodLabel("tekton.dev/task", pipelineName),
			tb.PodSpec(
				tb.PodInitContainer(tr1InitStep1, "override-with-creds:latest"),
				tb.PodInitContainer(tr1InitStep2, "override-with-tools:latest"),
				tb.PodContainer(tr1Step1Name, tr1Step1Name+":latest"),
				tb.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodInitContainerStatus(tr1InitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(tr1InitStep2, "override-with-tools:latest"),
			),
		),
		tb.Pod(tr2Pod,
			tb.PodNamespace(ns),
			tb.PodLabel("tekton.dev/task", pipelineName),
			tb.PodSpec(
				tb.PodContainer(tr2Step1Name, tr1Step1Name+":latest"),
				tb.PodContainer(nopStep, "override-with-nop:latest"),
			),
		),
	}

	fakeLogs := fake.Logs(
		fake.Task(tr1Pod,
			fake.Step(tr1InitStep1, "initialized the credentials"),
			fake.Step(tr1InitStep2, "place tools log"),
			fake.Step(tr1Step1Name, "written a file"),
			fake.Step(nopStep, "Build successful"),
		),
		fake.Task(tr2Pod,
			fake.Step(tr2Step1Name, "able to read a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	scenarios := []struct {
		name         string
		allSteps     bool
		tasks        []string
		expectedLogs []string
	}{
		{
			name:     "for all tasks",
			allSteps: false,
			expectedLogs: []string{
				"[output-task : writefile-step] written a file\n",
				"[output-task : nop] Build successful\n",
				"[read-task : readfile-step] able to read a file\n",
				"[read-task : nop] Build successful\n",
			},
		}, {
			name:     "for task1 only",
			allSteps: false,
			tasks:    []string{task1Name},
			expectedLogs: []string{
				"[output-task : writefile-step] written a file\n",
				"[output-task : nop] Build successful\n",
			},
		}, {
			name:     "including init steps",
			allSteps: true,
			expectedLogs: []string{
				"[output-task : credential-initializer-mdzbr] initialized the credentials\n",
				"[output-task : place-tools] place tools log\n",
				"[output-task : writefile-step] written a file\n",
				"[output-task : nop] Build successful\n",
				"[read-task : readfile-step] able to read a file\n",
				"[read-task : nop] Build successful\n",
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: pps, TaskRuns: trs, Pods: p, Namespaces: nsList})
			cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"task", "taskrun", "pipeline", "pipelinerun"})
			tdc := testDynamic.Options{}
			dc, err := tdc.Client(
				cb.UnstructuredP(pps[0], versionA1),
				cb.UnstructuredPR(prs[0], versionA1),
				cb.UnstructuredTR(trs[0], versionA1),
				cb.UnstructuredTR(trs[1], versionA1),
			)
			if err != nil {
				t.Errorf("unable to create dynamic client: %v", err)
			}
			prlo := logOptsv1aplha1(prName, ns, cs, dc, fake.Streamer(fakeLogs), s.allSteps, false, s.tasks...)
			output, _ := fetchLogs(prlo)

			expected := strings.Join(s.expectedLogs, "\n") + "\n"

			test.AssertOutput(t, expected, output)
		})
	}
}

// scenario, print logs for 1 completed taskruns out of 4 pipeline tasks
func TestPipelinerunLog_completed_taskrun_only(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		prstart      = clockwork.NewFakeClock()
		ns           = "namespace"

		task1Name    = "output-task"
		tr1Name      = "output-task-1"
		tr1StartTime = prstart.Now().Add(20 * time.Second)
		tr1Pod       = "output-task-pod-123456"
		tr1Step1Name = "writefile-step"

		// these are pipeline tasks for which pipeline has not
		// scheduled any taskrun
		task2Name = "read-task"

		task3Name = "notify"

		task4Name = "teardown"
	)

	trdata := []*v1alpha1.TaskRun{
		tb.TaskRun(tr1Name,
			tb.TaskRunNamespace(ns),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(task1Name),
			),
			tb.TaskRunStatus(
				tb.PodName(tr1Pod),
				tb.TaskRunStartTime(tr1StartTime),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName(tr1Step1Name),
					tb.StateTerminated(0),
				),
				tb.StepState(
					cb.StepName("nop"),
					tb.StateTerminated(0),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(task1Name),
			),
		),
	}
	pdata := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName,
			tb.PipelineNamespace(ns),
			tb.PipelineSpec(
				tb.PipelineTask(task1Name, task1Name),
				tb.PipelineTask(task2Name, task2Name),
				tb.PipelineTask(task3Name, task3Name),
				tb.PipelineTask(task4Name, task4Name),
			),
		),
	}
	prdata := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunLabel("tekton.dev/pipeline", prName),
			tb.PipelineRunSpec(pipelineName),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonRunning,
				}),
				tb.PipelineRunTaskRunsStatus(tr1Name, &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: task1Name,
				}),
			),
		),
	}
	// define pipeline in pipelineRef
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns:     trdata,
		Pipelines:    pdata,
		PipelineRuns: prdata,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
		Pods: []*corev1.Pod{
			tb.Pod(tr1Pod,
				tb.PodNamespace(ns),
				tb.PodLabel("tekton.dev/task", pipelineName),
				tb.PodSpec(
					tb.PodContainer(tr1Step1Name, tr1Step1Name+":latest"),
					tb.PodContainer("nop", "override-with-nop:latest"),
				),
			),
		},
	})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"task", "taskrun", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trdata[0], versionA1),
		cb.UnstructuredP(pdata[0], versionA1),
		cb.UnstructuredPR(prdata[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	tdata2 := []*v1alpha1.Task{
		tb.Task("output-task2", tb.TaskNamespace("ns"), cb.TaskCreationTime(clockwork.NewFakeClock().Now())),
	}
	trdata2 := []*v1alpha1.TaskRun{
		tb.TaskRun("output-taskrun2",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("output-task2")),
			tb.TaskRunStatus(
				tb.PodName("output-task-pod-embedded"),
				tb.TaskRunStartTime(clockwork.NewFakeClock().Now()),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.StepState(
					cb.StepName("test-step"),
					tb.StateTerminated(0),
				),
			),
		),
	}
	prdata2 := []*v1alpha1.PipelineRun{
		tb.PipelineRun("embedded-pipeline-1",
			tb.PipelineRunNamespace("ns"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "embedded-pipeline-1"),
			tb.PipelineRunSpec("", tb.PipelineRunPipelineSpec(
				tb.PipelineTask("output-task2", "output-task2"),
			)),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonRunning,
				}),
				tb.PipelineRunTaskRunsStatus("output-taskrun2", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "output-task2",
				}),
			),
		),
	}
	// define embedded pipeline
	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks:        tdata2,
		TaskRuns:     trdata2,
		PipelineRuns: prdata2,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
		Pods: []*corev1.Pod{
			tb.Pod("output-task-pod-embedded",
				tb.PodNamespace("ns"),
				tb.PodSpec(
					tb.PodContainer("test-step", "test-step1:latest"),
					tb.PodContainer("nop2", "override-with-nop:latest"),
				),
				cb.PodStatus(
					cb.PodPhase(corev1.PodSucceeded),
				),
			),
		},
	})
	cs2.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"task", "taskrun", "pipeline", "pipelinerun"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredT(tdata2[0], versionA1),
		cb.UnstructuredTR(trdata2[0], versionA1),
		cb.UnstructuredPR(prdata2[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	for _, tt := range []struct {
		name            string
		pipelineRunName string
		namespace       string
		dynamic         dynamic.Interface
		input           pipelinetest.Clients
		logs            []fake.Log
		want            []string
	}{
		{
			name:            "Test PipelineRef",
			pipelineRunName: prName,
			namespace:       ns,
			dynamic:         dc,
			input:           cs,
			logs: fake.Logs(
				fake.Task(tr1Pod,
					fake.Step(tr1Step1Name, "wrote a file"),
					fake.Step("nop", "Build successful"),
				),
			),
			want: []string{
				"[output-task : writefile-step] wrote a file\n",
				"[output-task : nop] Build successful\n",
				"",
			},
		},
		{
			name:            "Test embedded Pipeline",
			pipelineRunName: "embedded-pipeline-1",
			namespace:       "ns",
			dynamic:         dc2,
			input:           cs2,
			logs: fake.Logs(
				fake.Task("output-task-pod-embedded",
					fake.Step("test-step", "test embedded"),
					fake.Step("nop2", "Test successful"),
				),
			),
			want: []string{
				"[output-task2 : test-step] test embedded\n",
				"[output-task2 : nop2] Test successful\n",
				"",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prlo := logOptsv1aplha1(tt.pipelineRunName, tt.namespace, tt.input, tt.dynamic, fake.Streamer(tt.logs), false, false)
			output, _ := fetchLogs(prlo)
			test.AssertOutput(t, strings.Join(tt.want, "\n"), output)
		})
	}
}

func TestPipelinerunLog_follow_mode(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		prstart      = clockwork.NewFakeClock()
		ns           = "namespace"

		task1Name    = "output-task"
		tr1Name      = "output-task-1"
		tr1StartTime = prstart.Now().Add(20 * time.Second)
		tr1Pod       = "output-task-pod-123456"
		tr1Step1Name = "writefile-step"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun(tr1Name,
			tb.TaskRunNamespace(ns),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(task1Name),
			),
			tb.TaskRunStatus(
				tb.PodName(tr1Pod),
				tb.TaskRunStartTime(tr1StartTime),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName(tr1Step1Name),
					tb.StateTerminated(0),
				),
				tb.StepState(
					cb.StepName("nop"),
					tb.StateTerminated(0),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(task1Name),
			),
		),
	}

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunLabel("tekton.dev/pipeline", prName),
			tb.PipelineRunSpec(pipelineName),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonRunning,
				}),
				tb.PipelineRunTaskRunsStatus(tr1Name, &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: task1Name,
					Status:           &trs[0].Status,
				}),
			),
		),
	}

	pps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName,
			tb.PipelineNamespace(ns),
			tb.PipelineSpec(
				tb.PipelineTask(task1Name, task1Name),
			),
		),
	}

	p := []*corev1.Pod{
		tb.Pod(tr1Pod,
			tb.PodNamespace(ns),
			tb.PodLabel("tekton.dev/task", pipelineName),
			tb.PodSpec(
				tb.PodContainer(tr1Step1Name, tr1Step1Name+":latest"),
				tb.PodContainer("nop", "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
			),
		),
	}

	fakeLogStream := fake.Logs(
		fake.Task(tr1Pod,
			fake.Step(tr1Step1Name,
				"wrote a file1",
				"wrote a file2",
				"wrote a file3",
				"wrote a file4",
			),
			fake.Step("nop", "Build successful"),
		),
	)

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: pps, TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"task", "taskrun", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trs[0], versionA1),
		cb.UnstructuredPR(prs[0], versionA1),
		cb.UnstructuredP(pps[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOptsv1aplha1(prName, ns, cs, dc, fake.Streamer(fakeLogStream), false, true)
	output, _ := fetchLogs(prlo)

	expectedLogs := []string{
		"[output-task : writefile-step] wrote a file1",
		"[output-task : writefile-step] wrote a file2",
		"[output-task : writefile-step] wrote a file3",
		"[output-task : writefile-step] wrote a file4\n",
		"[output-task : nop] Build successful\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"

	test.AssertOutput(t, expected, output)
}

func TestLogs_error_log(t *testing.T) {
	var (
		pipelineName = "errlogs-pipeline"
		prName       = "errlogs-run"
		ns           = "namespace"
		taskName     = "errlogs-task"
		errMsg       = "Pipeline tektoncd/errlog-pipeline can't be Run; it contains Tasks that don't exist: Couldn't retrieve Task errlog-tasks: task.tekton.dev errlog-tasks not found"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	ts := []*v1alpha1.Task{
		tb.Task(taskName,
			tb.TaskNamespace(ns),
			tb.TaskSpec()),
	}

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunLabel("tekton.dev/pipeline", prName),
			tb.PipelineRunSpec(pipelineName),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status:  corev1.ConditionFalse,
					Message: errMsg,
				}),
			),
		),
	}

	ps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName,
			tb.PipelineNamespace(ns),
			tb.PipelineSpec(
				tb.PipelineTask(taskName, taskName),
			),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: ps, Tasks: ts, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"task", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredT(ts[0], versionA1),
		cb.UnstructuredP(ps[0], versionA1),
		cb.UnstructuredPR(prs[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOptsv1aplha1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false)
	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, errMsg+"\n", output)
}

func TestLogs_nologs(t *testing.T) {
	var (
		pipelineName = "nologs-pipeline"
		prName       = "nologs-run"
		ns           = "namespace"
		taskName     = "nologs-task"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunLabel("tekton.dev/pipeline", prName),
			tb.PipelineRunSpec(pipelineName),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status:  corev1.ConditionUnknown,
					Message: "Running",
				}),
			),
		),
	}

	ps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName,
			tb.PipelineNamespace(ns),
			tb.PipelineSpec(
				tb.PipelineTask(taskName, taskName),
			),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], versionA1),
		cb.UnstructuredPR(prs[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOptsv1aplha1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false)
	output, err := fetchLogs(prlo)
	if err == nil {
		t.Errorf("Unexpected output: %v", output)
	}
	test.AssertOutput(t, "pipelinerun has not started yet", err.Error())
}

func TestLog_run_failed_with_and_without_follow(t *testing.T) {
	var (
		pipelineName = "fail-pipeline"
		prName       = "fail-run"
		ns           = "namespace"
		taskName     = "fail-task"
		failMessage  = "Failed because I wanted"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunLabel("tekton.dev/pipeline", prName),
			tb.PipelineRunSpec(pipelineName),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionFalse,
					Message: failMessage,
				}),
			),
		),
	}

	ps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName,
			tb.PipelineNamespace(ns),
			tb.PipelineSpec(
				tb.PipelineTask(taskName, taskName),
			),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], versionA1),
		cb.UnstructuredPR(prs[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	// follow mode disabled
	prlo := logOptsv1aplha1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false)
	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, failMessage+"\n", output)

	// follow mode enabled
	prlo = logOptsv1aplha1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, true)
	output, err = fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, failMessage+"\n", output)
}

func TestLog_pipelinerun_still_running(t *testing.T) {
	var (
		pipelineName = "inprogress-pipeline"
		prName       = "inprogress-run"
		ns           = "namespace"
		taskName     = "inprogress-task"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	initialPRs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunLabel("tekton.dev/pipeline", prName),
			tb.PipelineRunSpec(pipelineName),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Message: "Running",
				}),
			),
		),
	}

	finalPRs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Message: "Running",
				}),
			),
		),

		tb.PipelineRun(prName,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionTrue,
					Message: "Running",
				}),
			),
		),
	}

	ps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName,
			tb.PipelineNamespace(ns),
			tb.PipelineSpec(
				tb.PipelineTask(taskName, taskName),
			),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: initialPRs, Pipelines: ps, Namespaces: nsList})
	watcher := watch.NewFake()
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"taskrun", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{WatchResource: "pipelineruns", Watcher: watcher}
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], versionA1),
		cb.UnstructuredPR(initialPRs[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOptsv1aplha1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false)

	updatePRv1alpha1(finalPRs, watcher)

	output, err := fetchLogs(prlo)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, "Pipeline still running ..."+"\n", output)
}

func TestLog_pipelinerun_status_done(t *testing.T) {
	var (
		pipelineName = "done-pipeline"
		prName       = "done-run"
		ns           = "namespace"
		taskName     = "done-task"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunLabel("tekton.dev/pipeline", prName),
			tb.PipelineRunSpec(pipelineName),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Message: "Running",
				}),
			),
		),
	}

	ps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName,
			tb.PipelineNamespace(ns),
			tb.PipelineSpec(
				tb.PipelineTask(taskName, taskName),
			),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"pipeline", "pipelinerun"})
	watcher := watch.NewFake()
	tdc := testDynamic.Options{WatchResource: "pipelineruns", Watcher: watcher}
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], versionA1),
		cb.UnstructuredPR(prs[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOptsv1aplha1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false)

	go func() {
		time.Sleep(time.Second * 1)
		for _, pr := range prs {
			pr.Status.Conditions[0].Status = corev1.ConditionTrue
			pr.Status.Conditions[0].Message = "completed"
			watcher.Modify(pr)
		}
	}()

	start := time.Now()
	output, err := fetchLogs(prlo)
	elapsed := time.Since(start).Seconds()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if elapsed > 10 {
		t.Errorf("Timed out")
	}
	test.AssertOutput(t, "", output)
}

func TestLog_pipelinerun_last(t *testing.T) {
	version := "v1alpha1"
	var (
		pipelineName = "pipeline1"
		prName       = "pr1"
		prName2      = "pr2"
		ns           = "namespaces"
		taskName     = "task1"
	)

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	pipelines := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName,
			tb.PipelineNamespace(ns),
			tb.PipelineSpec(
				tb.PipelineTask(taskName, taskName),
			),
		),
	}

	pipelineruns := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName2,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
			tb.PipelineRunSpec(pipelineName),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Message: "Running",
				}),
			),
		),
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
			tb.PipelineRunSpec(pipelineName),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Message: "Running",
				}),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: pipelineruns, Pipelines: pipelines, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(pipelineruns[0], version),
		cb.UnstructuredPR(pipelineruns[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := test.Params{
		Kube:    cs.Kube,
		Tekton:  cs.Pipeline,
		Dynamic: dc,
	}
	p.SetNamespace(ns)
	lopt := options.LogOptions{
		Params: &p,
		Last:   true,
		Limit:  len(pipelineruns),
	}
	err = askRunName(&lopt)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, prName2, lopt.PipelineRunName)
}

func TestLog_pipelinerun_only_one(t *testing.T) {
	version := "v1alpha1"
	var (
		pipelineName = "pipeline1"
		prName       = "pr1"
		ns           = "namespaces"
		taskName     = "task1"
	)

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	pipelines := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName,
			tb.PipelineNamespace(ns),
			tb.PipelineSpec(
				tb.PipelineTask(taskName, taskName),
			),
		),
	}

	pipelineruns := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
			tb.PipelineRunSpec(pipelineName),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Message: "Running",
				}),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: pipelineruns, Pipelines: pipelines, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(pipelineruns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := test.Params{
		Kube:    cs.Kube,
		Tekton:  cs.Pipeline,
		Dynamic: dc,
	}
	p.SetNamespace(ns)
	lopt := options.LogOptions{
		Params: &p,
		// This code https://git.io/JvCMV seems buggy so have to set the upper
		// Limit.. but I guess that's another fight for another day.
		Limit: len(pipelineruns),
	}
	err = askRunName(&lopt)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, prName, lopt.PipelineRunName)
}

func updatePRv1alpha1(finalRuns []*v1alpha1.PipelineRun, watcher *watch.FakeWatcher) {
	go func() {
		for _, pr := range finalRuns {
			time.Sleep(time.Second * 1)
			watcher.Modify(pr)
		}
	}()
}

func updatePRv1beta1(finalRuns []*v1beta1.PipelineRun, watcher *watch.FakeWatcher) {
	go func() {
		for _, pr := range finalRuns {
			time.Sleep(time.Second * 1)
			watcher.Modify(pr)
		}
	}()
}

func TestPipelinerunLog_completed_taskrun_only_v1bea1(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		prstart      = clockwork.NewFakeClock()
		ns           = "namespace"

		task1Name    = "output-task"
		tr1Name      = "output-task-1"
		tr1StartTime = prstart.Now().Add(20 * time.Second)
		tr1Pod       = "output-task-pod-123456"
		tr1Step1Name = "writefile-step"

		// these are pipeline tasks for which pipeline has not
		// scheduled any taskrun
		task2Name = "read-task"

		task3Name = "notify"

		task4Name = "teardown"
	)

	trdata := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      tr1Name,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: task1Name,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: tr1StartTime},
					PodName:   tr1Pod,
					Steps: []v1beta1.StepState{
						{
							Name: tr1Step1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: "nop",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
					},
				},
			},
		},
	}

	pdata := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: task1Name,
						TaskRef: &v1beta1.TaskRef{
							Name: task1Name,
						},
					},
					{
						Name: task2Name,
						TaskRef: &v1beta1.TaskRef{
							Name: task2Name,
						},
					},
					{
						Name: task3Name,
						TaskRef: &v1beta1.TaskRef{
							Name: task3Name,
						},
					},
					{
						Name: task4Name,
						TaskRef: &v1beta1.TaskRef{
							Name: task4Name,
						},
					},
				},
			},
		},
	}

	prdata := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonRunning,
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					TaskRuns: map[string]*v1beta1.PipelineRunTaskRunStatus{
						tr1Name: {
							PipelineTaskName: task1Name,
						},
					},
				},
			},
		},
	}

	// define pipeline in pipelineRef
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		TaskRuns:     trdata,
		Pipelines:    pdata,
		PipelineRuns: prdata,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
		Pods: []*corev1.Pod{
			tb.Pod(tr1Pod,
				tb.PodNamespace(ns),
				tb.PodLabel("tekton.dev/task", pipelineName),
				tb.PodSpec(
					tb.PodContainer(tr1Step1Name, tr1Step1Name+":latest"),
					tb.PodContainer("nop", "override-with-nop:latest"),
				),
			),
		},
	})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trdata[0], versionB1),
		cb.UnstructuredV1beta1P(pdata[0], versionB1),
		cb.UnstructuredV1beta1PR(prdata[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	tdata2 := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "output-task2",
				CreationTimestamp: metav1.Time{Time: clockwork.NewFakeClock().Now()},
			},
		},
	}

	trdata2 := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "output-taskrun2",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "output-task2",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: clockwork.NewFakeClock().Now()},
					PodName:   "output-task-pod-embedded",
					Steps: []v1beta1.StepState{
						{
							Name: "test-step",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
					},
				},
			},
		},
	}

	prdata2 := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "embedded-pipeline-1",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "embedded-pipeline-1"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineSpec: &v1beta1.PipelineSpec{
					Tasks: []v1beta1.PipelineTask{
						{
							Name: "output-task2",
							TaskRef: &v1beta1.TaskRef{
								Name: "output-task2",
							},
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonRunning,
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					TaskRuns: map[string]*v1beta1.PipelineRunTaskRunStatus{
						"output-taskrun2": {
							PipelineTaskName: "output-task2",
						},
					},
				},
			},
		},
	}
	// define embedded pipeline
	cs2, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Tasks:        tdata2,
		TaskRuns:     trdata2,
		PipelineRuns: prdata2,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
		Pods: []*corev1.Pod{
			tb.Pod("output-task-pod-embedded",
				tb.PodNamespace("ns"),
				tb.PodSpec(
					tb.PodContainer("test-step", "test-step1:latest"),
					tb.PodContainer("nop2", "override-with-nop:latest"),
				),
				cb.PodStatus(
					cb.PodPhase(corev1.PodSucceeded),
				),
			),
		},
	})
	cs2.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun", "pipeline", "pipelinerun"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1T(tdata2[0], versionB1),
		cb.UnstructuredV1beta1TR(trdata2[0], versionB1),
		cb.UnstructuredV1beta1PR(prdata2[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	for _, tt := range []struct {
		name            string
		pipelineRunName string
		namespace       string
		dynamic         dynamic.Interface
		input           pipelinev1beta1test.Clients
		logs            []fake.Log
		want            []string
	}{
		{
			name:            "Test PipelineRef",
			pipelineRunName: prName,
			namespace:       ns,
			dynamic:         dc,
			input:           cs,
			logs: fake.Logs(
				fake.Task(tr1Pod,
					fake.Step(tr1Step1Name, "wrote a file"),
					fake.Step("nop", "Build successful"),
				),
			),
			want: []string{
				"[output-task : writefile-step] wrote a file\n",
				"[output-task : nop] Build successful\n",
				"",
			},
		},
		{
			name:            "Test embedded Pipeline",
			pipelineRunName: "embedded-pipeline-1",
			namespace:       "ns",
			dynamic:         dc2,
			input:           cs2,
			logs: fake.Logs(
				fake.Task("output-task-pod-embedded",
					fake.Step("test-step", "test embedded"),
					fake.Step("nop2", "Test successful"),
				),
			),
			want: []string{
				"[output-task2 : test-step] test embedded\n",
				"[output-task2 : nop2] Test successful\n",
				"",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prlo := logOptsv1beta1(tt.pipelineRunName, tt.namespace, tt.input, tt.dynamic, fake.Streamer(tt.logs), false, false)
			output, _ := fetchLogs(prlo)
			test.AssertOutput(t, strings.Join(tt.want, "\n"), output)
		})
	}
}

func TestPipelinerunLog_follow_mode_v1beta1(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		prstart      = clockwork.NewFakeClock()
		ns           = "namespace"

		task1Name    = "output-task"
		tr1Name      = "output-task-1"
		tr1StartTime = prstart.Now().Add(20 * time.Second)
		tr1Pod       = "output-task-pod-123456"
		tr1Step1Name = "writefile-step"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun(tr1Name,
			tb.TaskRunNamespace(ns),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(task1Name),
			),
			tb.TaskRunStatus(
				tb.PodName(tr1Pod),
				tb.TaskRunStartTime(tr1StartTime),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName(tr1Step1Name),
					tb.StateTerminated(0),
				),
				tb.StepState(
					cb.StepName("nop"),
					tb.StateTerminated(0),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(task1Name),
			),
		),
	}

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunLabel("tekton.dev/pipeline", prName),
			tb.PipelineRunSpec(pipelineName),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonRunning,
				}),
				tb.PipelineRunTaskRunsStatus(tr1Name, &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: task1Name,
					Status:           &trs[0].Status,
				}),
			),
		),
	}

	pps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName,
			tb.PipelineNamespace(ns),
			tb.PipelineSpec(
				tb.PipelineTask(task1Name, task1Name),
			),
		),
	}

	p := []*corev1.Pod{
		tb.Pod(tr1Pod,
			tb.PodNamespace(ns),
			tb.PodLabel("tekton.dev/task", pipelineName),
			tb.PodSpec(
				tb.PodContainer(tr1Step1Name, tr1Step1Name+":latest"),
				tb.PodContainer("nop", "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
			),
		),
	}

	fakeLogStream := fake.Logs(
		fake.Task(tr1Pod,
			fake.Step(tr1Step1Name,
				"wrote a file1",
				"wrote a file2",
				"wrote a file3",
				"wrote a file4",
			),
			fake.Step("nop", "Build successful"),
		),
	)

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: pps, TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trs[0], versionB1),
		cb.UnstructuredPR(prs[0], versionB1),
		cb.UnstructuredP(pps[0], versionB1),
	)

	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	prlo := logOptsv1aplha1(prName, ns, cs, dc, fake.Streamer(fakeLogStream), false, true)
	output, _ := fetchLogs(prlo)

	expectedLogs := []string{
		"[output-task : writefile-step] wrote a file1",
		"[output-task : writefile-step] wrote a file2",
		"[output-task : writefile-step] wrote a file3",
		"[output-task : writefile-step] wrote a file4\n",
		"[output-task : nop] Build successful\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"
	test.AssertOutput(t, expected, output)
}

func TestLogs_error_log_v1beta1(t *testing.T) {
	var (
		pipelineName = "errlogs-pipeline"
		prName       = "errlogs-run"
		ns           = "namespace"
		taskName     = "errlogs-task"
		errMsg       = "Pipeline tektoncd/errlog-pipeline can't be Run; it contains Tasks that don't exist: Couldn't retrieve Task errlog-tasks: task.tekton.dev errlog-tasks not found"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	ts := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      taskName,
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Message: errMsg,
						},
					},
				},
			},
		},
	}

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1beta1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{PipelineRuns: prs, Pipelines: ps, Tasks: ts, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1T(ts[0], versionB1),
		cb.UnstructuredV1beta1P(ps[0], versionB1),
		cb.UnstructuredV1beta1PR(prs[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOptsv1beta1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false)
	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, errMsg+"\n", output)
}

func TestLogs_nologs_v1beta1(t *testing.T) {
	var (
		pipelineName = "nologs-pipeline"
		prName       = "nologs-run"
		ns           = "namespace"
		taskName     = "nologs-task"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1beta1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{PipelineRuns: prs, Pipelines: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], versionB1),
		cb.UnstructuredV1beta1PR(prs[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOptsv1beta1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false)
	output, err := fetchLogs(prlo)
	if err == nil {
		t.Errorf("Unexpected output: %v", output)
	}
	test.AssertOutput(t, "pipelinerun has not started yet", err.Error())
}

func TestLog_run_failed_with_and_without_follow_v1beta1(t *testing.T) {
	var (
		pipelineName = "fail-pipeline"
		prName       = "fail-run"
		ns           = "namespace"
		taskName     = "fail-task"
		failMessage  = "Failed because I wanted"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace(ns),
			tb.PipelineRunLabel("tekton.dev/pipeline", prName),
			tb.PipelineRunSpec(pipelineName),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionFalse,
					Message: failMessage,
				}),
			),
		),
	}

	ps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName,
			tb.PipelineNamespace(ns),
			tb.PipelineSpec(
				tb.PipelineTask(taskName, taskName),
			),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], versionB1),
		cb.UnstructuredPR(prs[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	// follow mode disabled
	prlo := logOptsv1aplha1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false)
	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, failMessage+"\n", output)

	// follow mode enabled
	prlo = logOptsv1aplha1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, true)
	output, err = fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, failMessage+"\n", output)
}

func TestLog_pipelinerun_still_running_v1beta1(t *testing.T) {
	var (
		pipelineName = "inprogress-pipeline"
		prName       = "inprogress-run"
		ns           = "namespace"
		taskName     = "inprogress-task"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	initialPRs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status:  corev1.ConditionUnknown,
							Type:    apis.ConditionSucceeded,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	finalPRs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status:  corev1.ConditionUnknown,
							Type:    apis.ConditionSucceeded,
							Message: "Running",
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Type:    apis.ConditionSucceeded,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1beta1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{PipelineRuns: initialPRs, Pipelines: ps, Namespaces: nsList})
	watcher := watch.NewFake()
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"taskrun", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{WatchResource: "pipelineruns", Watcher: watcher}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], versionB1),
		cb.UnstructuredV1beta1PR(initialPRs[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOptsv1beta1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false)

	updatePRv1beta1(finalPRs, watcher)

	output, err := fetchLogs(prlo)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, "Pipeline still running ..."+"\n", output)
}

func TestLog_pipelinerun_status_done_v1beta1(t *testing.T) {
	var (
		pipelineName = "done-pipeline"
		prName       = "done-run"
		ns           = "namespace"
		taskName     = "done-task"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1beta1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{PipelineRuns: prs, Pipelines: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"pipeline", "pipelinerun"})
	watcher := watch.NewFake()
	tdc := testDynamic.Options{WatchResource: "pipelineruns", Watcher: watcher}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], versionB1),
		cb.UnstructuredV1beta1PR(prs[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOptsv1beta1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false)

	go func() {
		time.Sleep(time.Second * 1)
		for _, pr := range prs {
			pr.Status.Conditions[0].Status = corev1.ConditionTrue
			pr.Status.Conditions[0].Message = "completed"
			watcher.Modify(pr)
		}
	}()

	start := time.Now()
	output, err := fetchLogs(prlo)
	elapsed := time.Since(start).Seconds()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if elapsed > 10 {
		t.Errorf("Timed out")
	}
	test.AssertOutput(t, "", output)
}

func TestLog_pipelinerun_last_v1beta1(t *testing.T) {
	var (
		pipelineName = "pipeline1"
		prName       = "pr1"
		prName2      = "pr2"
		ns           = "namespaces"
		taskName     = "task1"
	)

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}
	pipelines := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1beta1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	pipelineruns := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName2,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{PipelineRuns: pipelineruns, Pipelines: pipelines, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(pipelineruns[0], versionB1),
		cb.UnstructuredV1beta1PR(pipelineruns[1], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := test.Params{
		Kube:    cs.Kube,
		Tekton:  cs.Pipeline,
		Dynamic: dc,
	}
	p.SetNamespace(ns)
	lopt := options.LogOptions{
		Params: &p,
		Last:   true,
		Limit:  len(pipelineruns),
	}
	err = askRunName(&lopt)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, prName2, lopt.PipelineRunName)
}

func TestLog_pipelinerun_only_one_v1beta1(t *testing.T) {
	var (
		pipelineName = "pipeline1"
		prName       = "pr1"
		ns           = "namespaces"
		taskName     = "task1"
	)

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	pipelines := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1beta1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	pipelineruns := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{PipelineRuns: pipelineruns, Pipelines: pipelines, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(pipelineruns[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := test.Params{
		Kube:    cs.Kube,
		Tekton:  cs.Pipeline,
		Dynamic: dc,
	}
	p.SetNamespace(ns)
	lopt := options.LogOptions{
		Params: &p,
		// This code https://git.io/JvCMV seems buggy so have to set the upper
		// Limit.. but I guess that's another fight for another day.
		Limit: len(pipelineruns),
	}
	err = askRunName(&lopt)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, prName, lopt.PipelineRunName)
}

func logOptsv1aplha1(name string, ns string, cs pipelinetest.Clients, dc dynamic.Interface, streamer stream.NewStreamerFunc, allSteps bool, follow bool, tasks ...string) *options.LogOptions {
	p := test.Params{
		Kube:    cs.Kube,
		Tekton:  cs.Pipeline,
		Dynamic: dc,
	}
	p.SetNamespace(ns)

	logOptions := options.LogOptions{
		PipelineRunName: name,
		Tasks:           tasks,
		AllSteps:        allSteps,
		Follow:          follow,
		Params:          &p,
		Streamer:        streamer,
	}

	return &logOptions
}
func logOptsv1beta1(name string, ns string, cs pipelinev1beta1test.Clients, dc dynamic.Interface, streamer stream.NewStreamerFunc, allSteps bool, follow bool, tasks ...string) *options.LogOptions {
	p := test.Params{
		Kube:    cs.Kube,
		Tekton:  cs.Pipeline,
		Dynamic: dc,
	}
	p.SetNamespace(ns)

	logOptions := options.LogOptions{
		PipelineRunName: name,
		Tasks:           tasks,
		AllSteps:        allSteps,
		Follow:          follow,
		Params:          &p,
		Streamer:        streamer,
	}

	return &logOptions
}

func fetchLogs(lo *options.LogOptions) (string, error) {
	out := new(bytes.Buffer)
	lo.Stream = &cli.Stream{Out: out, Err: out}
	err := Run(lo)
	return out.String(), err
}
