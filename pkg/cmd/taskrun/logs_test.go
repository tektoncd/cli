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

package taskrun

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	tb "github.com/tektoncd/cli/internal/builder/v1alpha1"
	tbv1beta1 "github.com/tektoncd/cli/internal/builder/v1beta1"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/log"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/pods/fake"
	"github.com/tektoncd/cli/pkg/pods/stream"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	k8stest "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

const (
	versionA1 = "v1alpha1"
	versionB1 = "v1beta1"
)

func TestLog_invalid_namespace(t *testing.T) {
	tr := []*v1alpha1.TaskRun{
		tb.TaskRun("output-taskrun-1", tb.TaskRunNamespace("ns")),
	}

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: tr, Namespaces: nsList})
	watcher := watch.NewFake()
	cs.Kube.PrependWatchReactor("pods", k8stest.DefaultWatchReactor(watcher, nil))
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client()
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	c := Command(p)
	got, _ := test.ExecuteCommand(c, "logs", "output-taskrun-2", "-n", "invalid")
	expected := "Error: Unable to get Taskrun: taskruns.tekton.dev \"output-taskrun-2\" not found\n"
	test.AssertOutput(t, expected, got)
}

func TestLog_no_taskrun_arg(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc1, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	task2 := []*v1alpha1.Task{
		tb.Task("task", tb.TaskNamespace("ns"), cb.TaskCreationTime(clockwork.NewFakeClock().Now())),
	}
	taskrun2 := []*v1alpha1.TaskRun{
		tb.TaskRun("taskrun1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task")),
			tb.TaskRunStatus(
				tb.PodName("pod"),
				tb.TaskRunStartTime(clockwork.NewFakeClock().Now()),
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.TaskRunReasonSuccessful.String(),
				}),
				tb.StepState(
					cb.StepName("step1"),
					tb.StateTerminated(0),
				),
			),
		),
	}
	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks:    task2,
		TaskRuns: taskrun2,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
		Pods: []*corev1.Pod{
			tbv1beta1.Pod("pod", tbv1beta1.PodNamespace("ns"),
				tbv1beta1.PodSpec(
					tbv1beta1.PodContainer("step1", "step1:latest"),
				),
				cb.PodStatus(
					cb.PodPhase(corev1.PodSucceeded),
				),
			),
		},
	})
	cs2.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"taskrun"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredTR(taskrun2[0], "v1alpha1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	testParams := []struct {
		name      string
		dc        dynamic.Interface
		input     pipelinetest.Clients
		wantError bool
	}{
		{
			name:      "found no data",
			input:     cs,
			dc:        dc1,
			wantError: true,
		},
		{
			name:      "found single data",
			input:     cs2,
			dc:        dc2,
			wantError: false,
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			trlo := logoptsV1alpha1("", "ns", tp.input, fake.Streamer(fake.Logs()), false, false, []string{}, tp.dc)
			_, err := fetchLogs(trlo)
			if tp.wantError {
				if err == nil {
					t.Error("Expecting an error but it's empty")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
			}
		})
	}
}

func TestLog_missing_taskrun(t *testing.T) {
	tr := []*v1alpha1.TaskRun{
		tb.TaskRun("output-taskrun-1", tb.TaskRunNamespace("ns")),
	}

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(tr[0], "v1alpha1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: tr, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"taskrun"})
	watcher := watch.NewFake()
	cs.Kube.PrependWatchReactor("pods", k8stest.DefaultWatchReactor(watcher, nil))
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	c := Command(p)
	got, _ := test.ExecuteCommand(c, "logs", "output-taskrun-2", "-n", "ns")
	expected := "Error: " + log.MsgTRNotFoundErr + ": taskruns.tekton.dev \"output-taskrun-2\" not found\n"
	test.AssertOutput(t, expected, got)
}

func TestLog_invalid_flags(t *testing.T) {
	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: nsList})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	c := Command(p)
	got, _ := test.ExecuteCommand(c, "logs", "output-taskrun-2", "-n", "ns", "-a", "--step", "test")
	expected := "Error: option --all and option --step are not compatible\n"
	test.AssertOutput(t, expected, got)
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

func TestLog_taskrun_logs(t *testing.T) {
	var (
		ns          = "namespace"
		taskName    = "output-task"
		trName      = "output-task-1"
		trStartTime = clockwork.NewFakeClock().Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trStep1Name = "writefile-step"
		nopStep     = "nop"
	)

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun(trName, tb.TaskRunNamespace(ns),
			tb.TaskRunStatus(
				tb.PodName(trPod),
				tb.TaskRunStartTime(trStartTime),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName(trStep1Name),
					tb.StateTerminated(0),
				),
				tb.StepState(
					cb.StepName(nopStep),
					tb.StateTerminated(0),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(taskName),
			),
		),
	}

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	ps := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod,
			fake.Step(trStep1Name, "wrote a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Pods: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trs[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	trlo := logoptsV1alpha1(trName, ns, cs, fake.Streamer(logs), false, false, []string{}, dc)
	output, _ := fetchLogs(trlo)

	expectedLogs := []string{
		"[writefile-step] wrote a file\n",
		"[nop] Build successful\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"

	test.AssertOutput(t, expected, output)
}

func TestLog_taskrun_logs_v1beta1(t *testing.T) {
	var (
		ns          = "namespace"
		taskName    = "output-task"
		trName      = "output-task-1"
		trStartTime = clockwork.NewFakeClock().Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trStep1Name = "writefile-step"
		nopStep     = "nop"
	)

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionTrue,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					PodName:   trPod,
					StartTime: &metav1.Time{Time: trStartTime},
					Steps: []v1beta1.StepState{
						{
							Name: trStep1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: nopStep,
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

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	ps := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod,
			fake.Step(trStep1Name, "wrote a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Pods: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	trlo := logoptsV1beta1(trName, ns, cs, fake.Streamer(logs), false, false, []string{}, dc)
	output, _ := fetchLogs(trlo)

	expectedLogs := []string{
		"[writefile-step] wrote a file\n",
		"[nop] Build successful\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"

	test.AssertOutput(t, expected, output)
}

func TestLog_taskrun_logs_no_pod_name(t *testing.T) {
	var (
		ns          = "namespace"
		taskName    = "output-task"
		trName      = "output-task-1"
		trStartTime = clockwork.NewFakeClock().Now().Add(20 * time.Second)
		trStep1Name = "writefile-step"
		nopStep     = "nop"
	)

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionTrue,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: trStartTime},
					Steps: []v1beta1.StepState{
						{
							Name: trStep1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: nopStep,
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

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	ps := []*corev1.Pod{}

	logs := fake.Logs()

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Pods: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	trlo := logoptsV1beta1(trName, ns, cs, fake.Streamer(logs), false, false, []string{}, dc)
	_, err = fetchLogs(trlo)

	if err == nil {
		t.Error("Expecting an error but it's empty")
	}

	expected := "pod for taskrun output-task-1 not available yet"
	test.AssertOutput(t, expected, err.Error())
}

func TestLog_taskrun_logs_no_pod_name_v1beta1(t *testing.T) {
	var (
		ns          = "namespace"
		taskName    = "output-task"
		trName      = "output-task-1"
		trStartTime = clockwork.NewFakeClock().Now().Add(20 * time.Second)
		trStep1Name = "writefile-step"
		nopStep     = "nop"
	)

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionTrue,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: trStartTime},
					Steps: []v1beta1.StepState{
						{
							Name: trStep1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: nopStep,
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

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	ps := []*corev1.Pod{}

	logs := fake.Logs()

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Pods: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	trlo := logoptsV1beta1(trName, ns, cs, fake.Streamer(logs), false, false, []string{}, dc)
	_, err = fetchLogs(trlo)

	if err == nil {
		t.Error("Expecting an error but it's empty")
	}

	expected := "pod for taskrun output-task-1 not available yet"
	test.AssertOutput(t, expected, err.Error())
}

func TestLog_taskrun_all_steps(t *testing.T) {
	var (
		prstart  = clockwork.NewFakeClock()
		ns       = "namespace"
		taskName = "output-task"

		trName      = "output-task-run"
		trStartTime = prstart.Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trInitStep1 = "credential-initializer-mdzbr"
		trInitStep2 = "place-tools"
		trStep1Name = "writefile-step"
		nopStep     = "nop"
	)

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionTrue,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					PodName:   trPod,
					StartTime: &metav1.Time{Time: trStartTime},
					Steps: []v1beta1.StepState{
						{
							Name: trStep1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: nopStep,
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

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	p := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodInitContainer(trInitStep1, "override-with-creds:latest"),
				tbv1beta1.PodInitContainer(trInitStep2, "override-with-tools:latest"),
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
				cb.PodInitContainerStatus(trInitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(trInitStep2, "override-with-tools:latest"),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod,
			fake.Step(trInitStep1, "initialized the credentials"),
			fake.Step(trInitStep2, "place tools log"),
			fake.Step(trStep1Name, "written a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	trl := logoptsV1beta1(trName, ns, cs, fake.Streamer(logs), true, false, []string{}, dc)
	output, _ := fetchLogs(trl)

	expectedLogs := []string{
		"[credential-initializer-mdzbr] initialized the credentials\n",
		"[place-tools] place tools log\n",
		"[writefile-step] written a file\n",
		"[nop] Build successful\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"

	test.AssertOutput(t, expected, output)
}

func TestLog_taskrun_all_steps_v1beta1(t *testing.T) {
	var (
		prstart  = clockwork.NewFakeClock()
		ns       = "namespace"
		taskName = "output-task"

		trName      = "output-task-run"
		trStartTime = prstart.Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trInitStep1 = "credential-initializer-mdzbr"
		trInitStep2 = "place-tools"
		trStep1Name = "writefile-step"
		nopStep     = "nop"
	)

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionTrue,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					PodName:   trPod,
					StartTime: &metav1.Time{Time: trStartTime},
					Steps: []v1beta1.StepState{
						{
							Name: trStep1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: nopStep,
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

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	p := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodInitContainer(trInitStep1, "override-with-creds:latest"),
				tbv1beta1.PodInitContainer(trInitStep2, "override-with-tools:latest"),
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
				cb.PodInitContainerStatus(trInitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(trInitStep2, "override-with-tools:latest"),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod,
			fake.Step(trInitStep1, "initialized the credentials"),
			fake.Step(trInitStep2, "place tools log"),
			fake.Step(trStep1Name, "written a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	trl := logoptsV1beta1(trName, ns, cs, fake.Streamer(logs), true, false, []string{}, dc)
	output, _ := fetchLogs(trl)

	expectedLogs := []string{
		"[credential-initializer-mdzbr] initialized the credentials\n",
		"[place-tools] place tools log\n",
		"[writefile-step] written a file\n",
		"[nop] Build successful\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"

	test.AssertOutput(t, expected, output)
}

func TestLog_taskrun_given_steps(t *testing.T) {
	var (
		ns       = "namespace"
		taskName = "output-task"

		trName      = "output-task-run"
		trStartTime = clockwork.NewFakeClock().Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trInitStep1 = "credential-initializer-mdzbr"
		trInitStep2 = "place-tools"
		trStep1Name = "writefile-step"
		nopStep     = "nop"
	)

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun(trName, tb.TaskRunNamespace(ns),
			tb.TaskRunStatus(
				tb.PodName(trPod),
				tb.TaskRunStartTime(trStartTime),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName(trStep1Name),
					tb.StateTerminated(0),
				),
				tb.StepState(
					cb.StepName(nopStep),
					tb.StateTerminated(0),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(taskName),
			),
		),
	}

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	p := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodInitContainer(trInitStep1, "override-with-creds:latest"),
				tbv1beta1.PodInitContainer(trInitStep2, "override-with-tools:latest"),
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
				cb.PodInitContainerStatus(trInitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(trInitStep2, "override-with-tools:latest"),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod,
			fake.Step(trInitStep1, "initialized the credentials"),
			fake.Step(trInitStep2, "place tools log"),
			fake.Step(trStep1Name, "written a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trs[0], versionA1))
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	trl := logoptsV1alpha1(trName, ns, cs, fake.Streamer(logs), false, false, []string{trStep1Name}, dc)
	output, _ := fetchLogs(trl)

	expectedLogs := []string{
		"[writefile-step] written a file\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"

	test.AssertOutput(t, expected, output)
}

func TestLog_taskrun_given_steps_v1beta1(t *testing.T) {
	var (
		ns       = "namespace"
		taskName = "output-task"

		trName      = "output-task-run"
		trStartTime = clockwork.NewFakeClock().Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trInitStep1 = "credential-initializer-mdzbr"
		trInitStep2 = "place-tools"
		trStep1Name = "writefile-step"
		nopStep     = "nop"
	)

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionTrue,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					PodName:   trPod,
					StartTime: &metav1.Time{Time: trStartTime},
					Steps: []v1beta1.StepState{
						{
							Name: trStep1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: nopStep,
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

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	p := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodInitContainer(trInitStep1, "override-with-creds:latest"),
				tbv1beta1.PodInitContainer(trInitStep2, "override-with-tools:latest"),
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
				cb.PodInitContainerStatus(trInitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(trInitStep2, "override-with-tools:latest"),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod,
			fake.Step(trInitStep1, "initialized the credentials"),
			fake.Step(trInitStep2, "place tools log"),
			fake.Step(trStep1Name, "written a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionB1))
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	trl := logoptsV1beta1(trName, ns, cs, fake.Streamer(logs), false, false, []string{trStep1Name}, dc)
	output, _ := fetchLogs(trl)

	expectedLogs := []string{
		"[writefile-step] written a file\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"

	test.AssertOutput(t, expected, output)
}

func TestLog_taskrun_follow_mode(t *testing.T) {
	var (
		prstart     = clockwork.NewFakeClock()
		ns          = "namespace"
		taskName    = "output-task"
		trName      = "output-task-run"
		trStartTime = prstart.Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trStep1Name = "writefile-step"
		trInitStep1 = "credential-initializer-mdzbr"
		trInitStep2 = "place-tools"
		nopStep     = "nop"
	)

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun(trName, tb.TaskRunNamespace(ns),
			tb.TaskRunStatus(
				tb.PodName(trPod),
				tb.TaskRunStartTime(trStartTime),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName(trStep1Name),
					tb.StateTerminated(0),
				),
				tb.StepState(
					cb.StepName(nopStep),
					tb.StateTerminated(0),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(taskName),
			),
		),
	}

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	p := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodInitContainer(trInitStep1, "override-with-creds:latest"),
				tbv1beta1.PodInitContainer(trInitStep2, "override-with-tools:latest"),
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
				cb.PodInitContainerStatus(trInitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(trInitStep2, "override-with-tools:latest"),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod,
			fake.Step(trInitStep1, "initialized the credentials"),
			fake.Step(trInitStep2, "place tools log"),
			fake.Step(trStep1Name, "wrote a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trs[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	trlo := logoptsV1alpha1(trName, ns, cs, fake.Streamer(logs), false, true, []string{}, dc)
	output, _ := fetchLogs(trlo)

	expectedLogs := []string{
		"[writefile-step] wrote a file\n",
		"[nop] Build successful\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"

	test.AssertOutput(t, expected, output)
}

func TestLog_taskrun_follow_mode_v1beta1(t *testing.T) {
	var (
		prstart     = clockwork.NewFakeClock()
		ns          = "namespace"
		taskName    = "output-task"
		trName      = "output-task-run"
		trStartTime = prstart.Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trStep1Name = "writefile-step"
		trInitStep1 = "credential-initializer-mdzbr"
		trInitStep2 = "place-tools"
		nopStep     = "nop"
	)

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionTrue,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					PodName:   trPod,
					StartTime: &metav1.Time{Time: trStartTime},
					Steps: []v1beta1.StepState{
						{
							Name: trStep1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: nopStep,
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

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	p := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodInitContainer(trInitStep1, "override-with-creds:latest"),
				tbv1beta1.PodInitContainer(trInitStep2, "override-with-tools:latest"),
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
				cb.PodInitContainerStatus(trInitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(trInitStep2, "override-with-tools:latest"),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod,
			fake.Step(trInitStep1, "initialized the credentials"),
			fake.Step(trInitStep2, "place tools log"),
			fake.Step(trStep1Name, "wrote a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	trlo := logoptsV1beta1(trName, ns, cs, fake.Streamer(logs), false, true, []string{}, dc)
	output, _ := fetchLogs(trlo)

	expectedLogs := []string{
		"[writefile-step] wrote a file\n",
		"[nop] Build successful\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"

	test.AssertOutput(t, expected, output)
}

func TestLog_taskrun_last(t *testing.T) {
	var (
		ns           = "namespaces"
		trPod        = "pod"
		taskName     = "task"
		trName1      = "taskrun1"
		trName2      = "taskrun2"
		prstart      = clockwork.NewFakeClock()
		tr1StartTime = prstart.Now().Add(30 * time.Second)
		tr2StartTime = prstart.Now().Add(20 * time.Second)
		trStepName   = "writefile-step"
		nopStep      = "nop"
	)

	taskruns := []*v1alpha1.TaskRun{
		tb.TaskRun(trName1, tb.TaskRunNamespace(ns),
			tb.TaskRunStatus(
				tb.PodName(trPod),
				tb.TaskRunStartTime(tr1StartTime),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName(trStepName),
					tb.StateTerminated(0),
				),
				tb.StepState(
					cb.StepName(nopStep),
					tb.StateTerminated(0),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(taskName),
			),
		),
		tb.TaskRun(trName2, tb.TaskRunNamespace(ns),
			tb.TaskRunStatus(
				tb.PodName(trPod),
				tb.TaskRunStartTime(tr2StartTime),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName(trStepName),
					tb.StateTerminated(0),
				),
				tb.StepState(
					cb.StepName(nopStep),
					tb.StateTerminated(0),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(taskName),
			),
		),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: taskruns, Namespaces: namespaces})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(taskruns[0], versionA1),
		cb.UnstructuredTR(taskruns[1], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"taskrun"})
	p := test.Params{
		Kube:    cs.Kube,
		Tekton:  cs.Pipeline,
		Dynamic: dc,
	}
	p.SetNamespace(ns)
	lopt := options.LogOptions{
		Params: &p,
		Last:   true,
		Limit:  len(taskruns),
	}
	err = askRunName(&lopt)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, trName1, lopt.TaskrunName)
}

func TestLog_taskrun_last_v1beta1(t *testing.T) {
	var (
		ns           = "namespaces"
		trPod        = "pod"
		taskName     = "task"
		trName1      = "taskrun1"
		trName2      = "taskrun2"
		prstart      = clockwork.NewFakeClock()
		tr1StartTime = prstart.Now().Add(30 * time.Second)
		tr2StartTime = prstart.Now().Add(20 * time.Second)
		trStepName   = "writefile-step"
		nopStep      = "nop"
	)

	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName1,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionTrue,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					PodName:   trPod,
					StartTime: &metav1.Time{Time: tr1StartTime},
					Steps: []v1beta1.StepState{
						{
							Name: trStepName,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: nopStep,
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName2,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionTrue,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					PodName:   trPod,
					StartTime: &metav1.Time{Time: tr2StartTime},
					Steps: []v1beta1.StepState{
						{
							Name: trStepName,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: nopStep,
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

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: taskruns, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(taskruns[0], versionB1),
		cb.UnstructuredV1beta1TR(taskruns[1], versionB1),
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
		Limit:  len(taskruns),
	}
	err = askRunName(&lopt)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, trName1, lopt.TaskrunName)
}

func TestLog_taskrun_follow_mode_no_pod_name(t *testing.T) {
	var (
		prstart     = clockwork.NewFakeClock()
		ns          = "namespace"
		taskName    = "output-task"
		trName      = "output-task-run"
		trStartTime = prstart.Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trStep1Name = "writefile-step"
		trInitStep1 = "credential-initializer-mdzbr"
		trInitStep2 = "place-tools"
		nopStep     = "nop"
	)

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun(trName, tb.TaskRunNamespace(ns),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(trStartTime),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName(trStep1Name),
					tb.StateTerminated(0),
				),
				tb.StepState(
					cb.StepName(nopStep),
					tb.StateTerminated(0),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(taskName),
			),
		),
	}

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	p := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodInitContainer(trInitStep1, "override-with-creds:latest"),
				tbv1beta1.PodInitContainer(trInitStep2, "override-with-tools:latest"),
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
				cb.PodInitContainerStatus(trInitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(trInitStep2, "override-with-tools:latest"),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod,
			fake.Step(trInitStep1, "initialized the credentials"),
			fake.Step(trInitStep2, "place tools log"),
			fake.Step(trStep1Name, "wrote a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trs[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	trlo := logoptsV1alpha1(trName, ns, cs, fake.Streamer(logs), false, true, []string{}, dc)
	_, err = fetchLogs(trlo)

	if err == nil {
		t.Error("Expecting an error but it's empty")
	}

	expected := "task output-task create has not started yet or pod for task not yet available"
	test.AssertOutput(t, expected, err.Error())
}

func TestLog_taskrun_follow_mode_no_pod_name_v1beta1(t *testing.T) {
	var (
		prstart     = clockwork.NewFakeClock()
		ns          = "namespace"
		taskName    = "output-task"
		trName      = "output-task-run"
		trStartTime = prstart.Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trStep1Name = "writefile-step"
		trInitStep1 = "credential-initializer-mdzbr"
		trInitStep2 = "place-tools"
		nopStep     = "nop"
	)

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionTrue,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: trStartTime},
					Steps: []v1beta1.StepState{
						{
							Name: trStep1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: nopStep,
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

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	p := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodInitContainer(trInitStep1, "override-with-creds:latest"),
				tbv1beta1.PodInitContainer(trInitStep2, "override-with-tools:latest"),
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
				cb.PodInitContainerStatus(trInitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(trInitStep2, "override-with-tools:latest"),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod,
			fake.Step(trInitStep1, "initialized the credentials"),
			fake.Step(trInitStep2, "place tools log"),
			fake.Step(trStep1Name, "wrote a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	trlo := logoptsV1beta1(trName, ns, cs, fake.Streamer(logs), false, true, []string{}, dc)
	_, err = fetchLogs(trlo)

	if err == nil {
		t.Error("Expecting an error but it's empty")
	}

	expected := "task output-task create has not started yet or pod for task not yet available"
	test.AssertOutput(t, expected, err.Error())
}

func TestLog_taskrun_follow_mode_update_pod_name(t *testing.T) {
	var (
		prstart     = clockwork.NewFakeClock()
		ns          = "namespace"
		taskName    = "output-task"
		trName      = "output-task-run"
		trStartTime = prstart.Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trStep1Name = "writefile-step"
		trInitStep1 = "credential-initializer-mdzbr"
		trInitStep2 = "place-tools"
		nopStep     = "nop"
	)

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun(trName, tb.TaskRunNamespace(ns),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(trStartTime),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName(trStep1Name),
					tb.StateTerminated(0),
				),
				tb.StepState(
					cb.StepName(nopStep),
					tb.StateTerminated(0),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(taskName),
			),
		),
	}

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	p := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodInitContainer(trInitStep1, "override-with-creds:latest"),
				tbv1beta1.PodInitContainer(trInitStep2, "override-with-tools:latest"),
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
				cb.PodInitContainerStatus(trInitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(trInitStep2, "override-with-tools:latest"),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod,
			fake.Step(trInitStep1, "initialized the credentials"),
			fake.Step(trInitStep2, "place tools log"),
			fake.Step(trStep1Name, "wrote a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"taskrun"})
	watcher := watch.NewRaceFreeFake()
	tdc := testDynamic.Options{WatchResource: "taskruns", Watcher: watcher}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trs[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	trlo := logoptsV1alpha1(trName, ns, cs, fake.Streamer(logs), false, true, []string{}, dc)

	go func() {
		time.Sleep(time.Second * 1)
		trs[0].Status.PodName = trPod
		watcher.Modify(trs[0])
	}()

	output, err := fetchLogs(trlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedLogs := []string{
		"[writefile-step] wrote a file\n",
		"[nop] Build successful\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"
	test.AssertOutput(t, expected, output)
}

func TestLog_taskrun_follow_mode_update_pod_name_v1beta1(t *testing.T) {
	var (
		prstart     = clockwork.NewFakeClock()
		ns          = "namespace"
		taskName    = "output-task"
		trName      = "output-task-run"
		trStartTime = prstart.Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trStep1Name = "writefile-step"
		trInitStep1 = "credential-initializer-mdzbr"
		trInitStep2 = "place-tools"
		nopStep     = "nop"
	)

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionTrue,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: trStartTime},
					Steps: []v1beta1.StepState{
						{
							Name: trStep1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: nopStep,
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

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	p := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodInitContainer(trInitStep1, "override-with-creds:latest"),
				tbv1beta1.PodInitContainer(trInitStep2, "override-with-tools:latest"),
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
				cb.PodInitContainerStatus(trInitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(trInitStep2, "override-with-tools:latest"),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod,
			fake.Step(trInitStep1, "initialized the credentials"),
			fake.Step(trInitStep2, "place tools log"),
			fake.Step(trStep1Name, "wrote a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"taskrun"})
	watcher := watch.NewRaceFreeFake()
	tdc := testDynamic.Options{WatchResource: "taskruns", Watcher: watcher}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	trlo := logoptsV1beta1(trName, ns, cs, fake.Streamer(logs), false, true, []string{}, dc)

	go func() {
		time.Sleep(time.Second * 1)
		trs[0].Status.PodName = trPod
		watcher.Modify(trs[0])
	}()

	output, err := fetchLogs(trlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedLogs := []string{
		"[writefile-step] wrote a file\n",
		"[nop] Build successful\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"
	test.AssertOutput(t, expected, output)
}

func TestLog_taskrun_follow_mode_update_timeout(t *testing.T) {
	var (
		prstart     = clockwork.NewFakeClock()
		ns          = "namespace"
		taskName    = "output-task"
		trName      = "output-task-run"
		trStartTime = prstart.Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trStep1Name = "writefile-step"
		trInitStep1 = "credential-initializer-mdzbr"
		trInitStep2 = "place-tools"
		nopStep     = "nop"
	)

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun(trName, tb.TaskRunNamespace(ns),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(trStartTime),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName(trStep1Name),
					tb.StateTerminated(0),
				),
				tb.StepState(
					cb.StepName(nopStep),
					tb.StateTerminated(0),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(taskName),
			),
		),
	}

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	p := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodInitContainer(trInitStep1, "override-with-creds:latest"),
				tbv1beta1.PodInitContainer(trInitStep2, "override-with-tools:latest"),
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
				cb.PodInitContainerStatus(trInitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(trInitStep2, "override-with-tools:latest"),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod,
			fake.Step(trInitStep1, "initialized the credentials"),
			fake.Step(trInitStep2, "place tools log"),
			fake.Step(trStep1Name, "wrote a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"taskrun"})
	watcher := watch.NewRaceFreeFake()
	tdc := testDynamic.Options{WatchResource: "taskruns", Watcher: watcher}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trs[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	trlo := logoptsV1alpha1(trName, ns, cs, fake.Streamer(logs), false, true, []string{}, dc)

	go func() {
		time.Sleep(time.Second * 1)
		watcher.Action("MODIFIED", trs[0])
	}()

	output, err := fetchLogs(trlo)
	if err == nil {
		t.Error("Expecting an error but it's empty")
	}

	expectedOut := ""
	test.AssertOutput(t, expectedOut, output)

	expectedErr := "task output-task create has not started yet or pod for task not yet available"
	test.AssertOutput(t, expectedErr, err.Error())
}

func TestLog_taskrun_follow_mode_update_timeout_v1beta1(t *testing.T) {
	var (
		prstart     = clockwork.NewFakeClock()
		ns          = "namespace"
		taskName    = "output-task"
		trName      = "output-task-run"
		trStartTime = prstart.Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trStep1Name = "writefile-step"
		trInitStep1 = "credential-initializer-mdzbr"
		trInitStep2 = "place-tools"
		nopStep     = "nop"
	)

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionTrue,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: trStartTime},
					Steps: []v1beta1.StepState{
						{
							Name: trStep1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: nopStep,
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

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	p := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodInitContainer(trInitStep1, "override-with-creds:latest"),
				tbv1beta1.PodInitContainer(trInitStep2, "override-with-tools:latest"),
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
				cb.PodInitContainerStatus(trInitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(trInitStep2, "override-with-tools:latest"),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod,
			fake.Step(trInitStep1, "initialized the credentials"),
			fake.Step(trInitStep2, "place tools log"),
			fake.Step(trStep1Name, "wrote a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"taskrun"})
	watcher := watch.NewRaceFreeFake()
	tdc := testDynamic.Options{WatchResource: "taskruns", Watcher: watcher}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	trlo := logoptsV1beta1(trName, ns, cs, fake.Streamer(logs), false, true, []string{}, dc)

	go func() {
		time.Sleep(time.Second * 1)
		watcher.Action("MODIFIED", trs[0])
	}()

	output, err := fetchLogs(trlo)
	if err == nil {
		t.Error("Expecting an error but it's empty")
	}

	expectedOut := ""
	test.AssertOutput(t, expectedOut, output)

	expectedErr := "task output-task create has not started yet or pod for task not yet available"
	test.AssertOutput(t, expectedErr, err.Error())
}

func TestLog_taskrun_follow_mode_no_output_provided(t *testing.T) {
	var (
		prstart     = clockwork.NewFakeClock()
		ns          = "namespace"
		taskName    = "output-task"
		trName      = "output-task-run"
		trStartTime = prstart.Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trStep1Name = "writefile-step"
		trInitStep1 = "credential-initializer-mdzbr"
		trInitStep2 = "place-tools"
		nopStep     = "nop"
	)

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun(trName, tb.TaskRunNamespace(ns),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(trStartTime),
				tb.StatusCondition(apis.Condition{
					Status:  corev1.ConditionFalse,
					Reason:  v1beta1.TaskRunReasonFailed.String(),
					Message: "invalid output resources: TaskRun's declared resources didn't match usage in Task: Didn't provide required values: [builtImage]",
				}),
				tb.StepState(
					cb.StepName(trStep1Name),
					tb.StateTerminated(0),
				),
				tb.StepState(
					cb.StepName(nopStep),
					tb.StateTerminated(0),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(taskName),
			),
		),
	}

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	p := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodInitContainer(trInitStep1, "override-with-creds:latest"),
				tbv1beta1.PodInitContainer(trInitStep2, "override-with-tools:latest"),
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
				cb.PodInitContainerStatus(trInitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(trInitStep2, "override-with-tools:latest"),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod),
	)

	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"taskrun"})
	watcher := watch.NewRaceFreeFake()
	tdc := testDynamic.Options{WatchResource: "taskruns", Watcher: watcher}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trs[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	trlo := logoptsV1alpha1(trName, ns, cs, fake.Streamer(logs), false, true, []string{}, dc)

	_, err = fetchLogs(trlo)
	if err == nil {
		t.Errorf("Expected error but no error occurred ")
	}

	expected := "task output-task has failed: invalid output resources: TaskRun's declared resources didn't match usage in Task: Didn't provide required values: [builtImage]"
	test.AssertOutput(t, expected, err.Error())
}

func TestLog_taskrun_follow_mode_no_output_provided_v1beta1(t *testing.T) {
	var (
		prstart     = clockwork.NewFakeClock()
		ns          = "namespace"
		taskName    = "output-task"
		trName      = "output-task-run"
		trStartTime = prstart.Now().Add(20 * time.Second)
		trPod       = "output-task-pod-123456"
		trStep1Name = "writefile-step"
		trInitStep1 = "credential-initializer-mdzbr"
		trInitStep2 = "place-tools"
		nopStep     = "nop"
	)

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  v1beta1.TaskRunReasonFailed.String(),
							Message: "invalid output resources: TaskRun's declared resources didn't match usage in Task: Didn't provide required values: [builtImage]",
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: trStartTime},
					Steps: []v1beta1.StepState{
						{
							Name: trStep1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: nopStep,
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

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	p := []*corev1.Pod{
		tbv1beta1.Pod(trPod, tbv1beta1.PodNamespace(ns),
			tbv1beta1.PodSpec(
				tbv1beta1.PodInitContainer(trInitStep1, "override-with-creds:latest"),
				tbv1beta1.PodInitContainer(trInitStep2, "override-with-tools:latest"),
				tbv1beta1.PodContainer(trStep1Name, trStep1Name+":latest"),
				tbv1beta1.PodContainer(nopStep, "override-with-nop:latest"),
			),
			cb.PodStatus(
				cb.PodPhase(corev1.PodSucceeded),
				cb.PodInitContainerStatus(trInitStep1, "override-with-creds:latest"),
				cb.PodInitContainerStatus(trInitStep2, "override-with-tools:latest"),
			),
		),
	}

	logs := fake.Logs(
		fake.Task(trPod),
	)

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"taskrun"})
	watcher := watch.NewRaceFreeFake()
	tdc := testDynamic.Options{WatchResource: "taskruns", Watcher: watcher}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	trlo := logoptsV1beta1(trName, ns, cs, fake.Streamer(logs), false, true, []string{}, dc)

	_, err = fetchLogs(trlo)
	if err == nil {
		t.Errorf("Expected error but no error occurred ")
	}

	expected := "task output-task has failed: invalid output resources: TaskRun's declared resources didn't match usage in Task: Didn't provide required values: [builtImage]"
	test.AssertOutput(t, expected, err.Error())
}

func logoptsV1alpha1(run, ns string, cs pipelinetest.Clients, streamer stream.NewStreamerFunc,
	allSteps bool, follow bool, steps []string, dc dynamic.Interface) *options.LogOptions {
	p := test.Params{
		Kube:    cs.Kube,
		Tekton:  cs.Pipeline,
		Dynamic: dc,
	}
	p.SetNamespace(ns)

	return &options.LogOptions{
		TaskrunName: run,
		AllSteps:    allSteps,
		Follow:      follow,
		Params:      &p,
		Streamer:    streamer,
		Limit:       5,
		Steps:       steps,
	}
}

func logoptsV1beta1(run, ns string, cs pipelinev1beta1test.Clients, streamer stream.NewStreamerFunc,
	allSteps bool, follow bool, steps []string, dc dynamic.Interface) *options.LogOptions {
	p := test.Params{
		Kube:    cs.Kube,
		Tekton:  cs.Pipeline,
		Dynamic: dc,
	}
	p.SetNamespace(ns)

	return &options.LogOptions{
		TaskrunName: run,
		AllSteps:    allSteps,
		Follow:      follow,
		Params:      &p,
		Streamer:    streamer,
		Limit:       5,
		Steps:       steps,
	}
}

func fetchLogs(lo *options.LogOptions) (string, error) {
	out := new(bytes.Buffer)
	lo.Stream = &cli.Stream{Out: out, Err: out}

	err := Run(lo)

	return out.String(), err
}
