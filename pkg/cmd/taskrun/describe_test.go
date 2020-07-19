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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

var (
	reasonCompleted = corev1.ContainerStateTerminated{Reason: "Completed"}
	reasonWaiting   = corev1.ContainerStateWaiting{Reason: "PodInitializing"}
	reasonFailed    = corev1.ContainerStateTerminated{Reason: "Error"}
	reasonRunning   = corev1.ContainerStateRunning{StartedAt: metav1.Time{Time: time.Now()}}
)

func TestTaskRunDescribe_invalid_namespace(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			},
		},
	})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client()
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"taskrun"})

	taskrun := Command(p)
	out, err := test.ExecuteCommand(taskrun, "desc", "bar", "-n", "invalid")
	if err == nil {
		t.Errorf("Expected error but did not get one")
	}
	expected := "Error: failed to get TaskRun bar: taskruns.tekton.dev \"bar\" not found\n"
	test.AssertOutput(t, expected, out)
}

func TestTaskRunDescribe_not_found(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"taskrun"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	taskrun := Command(p)
	out, err := test.ExecuteCommand(taskrun, "desc", "bar", "-n", "ns")
	if err == nil {
		t.Errorf("Expected error but did not get one")
	}
	expected := "Error: failed to get TaskRun bar: taskruns.tekton.dev \"bar\" not found\n"
	test.AssertOutput(t, expected, out)
}

func TestTaskRunDescribe_empty_taskrun(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "t1"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("t1")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.TaskRunReasonSuccessful.String(),
				}),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	taskrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(taskrun, "desc", "tr-1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_only_taskrun(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clockwork.NewFakeClock().Now().Add(20*time.Second)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName("step1"),
					tb.SetStepStateTerminated(reasonCompleted),
				),
				tb.StepState(
					cb.StepName("step2"),
					tb.SetStepStateTerminated(reasonCompleted),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("t1"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("input", "param")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("input2", "param2")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("git", tb.TaskResourceBindingRef("git"))),
				tb.TaskRunInputs(tb.TaskRunInputsResource("image-input", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("image-output", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("image-output2", tb.TaskResourceBindingRef("image"))),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	taskrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(taskrun, "desc", "tr-1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_failed(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(2*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(5*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Status:  corev1.ConditionFalse,
					Reason:  v1beta1.TaskRunReasonFailed.String(),
					Message: "Testing tr failed",
				}),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("t1"),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	taskrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(taskrun, "desc", "tr-1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_no_taskref(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(2*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(5*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Status:  corev1.ConditionFalse,
					Reason:  v1beta1.TaskRunReasonFailed.String(),
					Message: "Testing tr failed",
				}),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	taskrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(taskrun, "desc", "tr-1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_last_no_taskrun_present(t *testing.T) {
	trs := []*v1beta1.TaskRun{}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	taskrun := Command(p)
	out, err := test.ExecuteCommand(taskrun, "desc", "--last", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := "No TaskRuns present in namespace ns\n"
	test.AssertOutput(t, expected, out)
}

func TestTaskRunDescribe_no_resourceref(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clockwork.NewFakeClock().Now().Add(20*time.Second)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName("step1"),
					tb.SetStepStateTerminated(reasonCompleted),
				),
				tb.StepState(
					cb.StepName("step2"),
					tb.SetStepStateTerminated(reasonCompleted),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("t1"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("input", "param")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("input2", "param2")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("git")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("image-input", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("image-output")),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("image-output2")),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	taskrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(taskrun, "desc", "tr-1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_step_sidecar_status_defaults_and_failures(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clockwork.NewFakeClock().Now().Add(20*time.Second)),
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionFalse,
					Reason: v1beta1.TaskRunReasonFailed.String(),
				}),
				tb.StepState(
					cb.StepName("step1"),
					tb.SetStepStateTerminated(reasonFailed),
				),
				tb.StepState(
					cb.StepName("step2"),
				),
				tb.SidecarState(
					tb.SidecarStateName("sidecar1"),
					tb.SetSidecarStateTerminated(reasonFailed),
				),
				tb.SidecarState(
					tb.SidecarStateName("sidecar2"),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("t1"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("input", "param")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("input2", "param2")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("git")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("image-input", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("image-output")),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("image-output2")),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	taskrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(taskrun, "desc", "tr-1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_step_status_pending_one_sidecar(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clockwork.NewFakeClock().Now().Add(20*time.Second)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionUnknown,
					Reason: "Running",
				}),
				tb.StepState(
					cb.StepName("step1"),
					tb.SetStepStateWaiting(reasonWaiting),
				),
				tb.StepState(
					cb.StepName("step2"),
					tb.SetStepStateWaiting(reasonWaiting),
				),
				tb.SidecarState(
					tb.SidecarStateName("sidecar1"),
					tb.SetSidecarStateWaiting(reasonWaiting),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("t1"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("input", "param")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("input2", "param2")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("git")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("image-input", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("image-output")),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("image-output2")),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	taskrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(taskrun, "desc", "tr-1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_step_status_running_multiple_sidecars(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clockwork.NewFakeClock().Now().Add(20*time.Second)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionUnknown,
					Reason: "Running",
				}),
				tb.StepState(
					cb.StepName("step1"),
					tb.SetStepStateRunning(reasonRunning),
				),
				tb.StepState(
					cb.StepName("step2"),
					tb.SetStepStateRunning(reasonRunning),
				),
				tb.SidecarState(
					tb.SidecarStateName("sidecar1"),
					tb.SetSidecarStateRunning(reasonRunning),
				),
				tb.SidecarState(
					tb.SidecarStateName("sidecar2"),
					tb.SetSidecarStateRunning(reasonRunning),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("t1"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("input", "param")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("input2", "param2")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("git")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("image-input", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("image-output")),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("image-output2")),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	taskrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(taskrun, "desc", "tr-1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_cancel_taskrun(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(2*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(5*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Status:  corev1.ConditionFalse,
					Reason:  "TaskRunCancelled",
					Message: "TaskRun \"tr-1\" was cancelled",
				}),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	taskrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(taskrun, "desc", "tr-1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_custom_output(t *testing.T) {
	name := "task-run"
	expected := "taskrun.tekton.dev/" + name

	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun(name, tb.TaskRunNamespace("ns")),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	taskrun := Command(p)

	got, err := test.ExecuteCommand(taskrun, "desc", "-o", "name", "-n", "ns", name)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	got = strings.TrimSpace(got)
	if got != expected {
		t.Errorf("Result should be '%s' != '%s'", got, expected)
	}
}

func TestTaskRunWithSpecDescribe_custom_timeout(t *testing.T) {
	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-custom-timeout", tb.TaskRunNamespace("ns"),
			tb.TaskRunSpec(tb.TaskRunTimeout(time.Minute)),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	taskrun := Command(p)
	actual, err := test.ExecuteCommand(taskrun, "desc", "tr-custom-timeout", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_lastV1beta1(t *testing.T) {
	clock := clockwork.NewFakeClock()
	taskRuns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-1",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.TaskRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-2",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
			},
		},
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(taskRuns[0], version),
		cb.UnstructuredV1beta1TR(taskRuns[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	taskrun := Command(p)
	got, err := test.ExecuteCommand(taskrun, "desc", "-n", "ns", "--last")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_last(t *testing.T) {
	clock := clockwork.NewFakeClock()
	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1", tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clockwork.NewFakeClock().Now().Add(20*time.Second)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName("step1"),
					tb.SetStepStateTerminated(reasonCompleted),
				),
				tb.StepState(
					cb.StepName("step2"),
					tb.SetStepStateTerminated(reasonCompleted),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("t1"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("input", "param")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("input2", "param2")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("git", tb.TaskResourceBindingRef("git"))),
				tb.TaskRunInputs(tb.TaskRunInputsResource("image-input", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("image-output", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("image-output2", tb.TaskResourceBindingRef("image"))),
			),
		),
		tb.TaskRun("tr-2", tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clockwork.NewFakeClock().Now().Add(2*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
				tb.StepState(
					cb.StepName("step3"),
					tb.SetStepStateTerminated(reasonCompleted),
				),
				tb.StepState(
					cb.StepName("step4"),
					tb.SetStepStateTerminated(reasonCompleted),
				),
			),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("t1"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("input", "param")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("input2", "param2")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("git", tb.TaskResourceBindingRef("git"))),
				tb.TaskRunInputs(tb.TaskRunInputsResource("image-input", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("image-output", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("image-output2", tb.TaskResourceBindingRef("image"))),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
		cb.UnstructuredTR(trs[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	taskrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(taskrun, "desc", "--last", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_With_Results(t *testing.T) {
	clock := clockwork.NewFakeClock()
	taskRuns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-1",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.TaskRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
					TaskRunResults: []v1alpha1.TaskRunResult{
						{
							Name:  "result-1",
							Value: "value-1",
						},
						{
							Name:  "result-2",
							Value: "value-2",
						},
					},
				},
			},
		},
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(taskRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	taskrun := Command(p)
	got, err := test.ExecuteCommand(taskrun, "desc", "tr-1")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}
