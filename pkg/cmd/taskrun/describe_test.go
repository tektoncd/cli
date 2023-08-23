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

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
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
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})

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

	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
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

func TestTaskRunDescribe_empty_taskrun_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "t1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "t1",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
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

func TestTaskRunDescribe_only_taskrun_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: test.FakeClock().Now().Add(20 * time.Second)},
					Steps: []v1beta1.StepState{
						{
							Name: "step1",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: reasonCompleted.Reason,
								},
							},
						},
						{
							Name: "step2",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: reasonCompleted.Reason,
								},
							},
						},
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "t1",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
				Params: []v1beta1.Param{
					{
						Name:  "input",
						Value: v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "param"},
					},
					{
						Name:  "input2",
						Value: v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "param2"},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
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

func TestTaskRunDescribe_failed_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  v1beta1.TaskRunReasonFailed.String(),
							Message: "Testing tr failed",
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(2 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "t1",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
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

func TestTaskRunDescribe_no_taskref_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(2 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  v1beta1.TaskRunReasonFailed.String(),
							Message: "Testing tr failed",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
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

func TestTaskRunDescribe_last_no_taskrun_present_v1beta1(t *testing.T) {
	var trs []*v1beta1.TaskRun
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
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

func TestTaskRunDescribe_no_resourceref_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: test.FakeClock().Now().Add(20 * time.Second)},
					Steps: []v1beta1.StepState{
						{
							Name: "step1",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: reasonCompleted.Reason,
								},
							},
						},
						{
							Name: "step2",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: reasonCompleted.Reason,
								},
							},
						},
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "t1",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
				Params: []v1beta1.Param{
					{
						Name:  "input",
						Value: v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "param"},
					},
					{
						Name:  "input2",
						Value: v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "param2"},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
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

func TestTaskRunDescribe_step_sidecar_status_defaults_and_failures_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: test.FakeClock().Now().Add(20 * time.Second)},
					Steps: []v1beta1.StepState{
						{
							Name: "step1",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: reasonFailed.Reason,
								},
							},
						},
						{
							Name: "step2",
						},
					},
					Sidecars: []v1beta1.SidecarState{
						{
							Name: "sidecar1",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: reasonFailed.Reason,
								},
							},
						},
						{
							Name: "sidecar2",
						},
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.TaskRunReasonFailed.String(),
						},
					},
				},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "t1",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
				Params: []v1beta1.Param{
					{
						Name:  "input",
						Value: v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "param"},
					},
					{
						Name:  "input2",
						Value: v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "param2"},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
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

func TestTaskRunDescribe_step_status_pending_one_sidecar_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: test.FakeClock().Now().Add(20 * time.Second)},
					Steps: []v1beta1.StepState{
						{
							Name: "step1",
							ContainerState: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: reasonWaiting.Reason,
								},
							},
						},
						{
							Name: "step2",
							ContainerState: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: reasonWaiting.Reason,
								},
							},
						},
					},
					Sidecars: []v1beta1.SidecarState{
						{
							Name: "sidecar1",
							ContainerState: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: reasonWaiting.Reason,
								},
							},
						},
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Type:   apis.ConditionSucceeded,
							Reason: "Running",
						},
					},
				},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "t1",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
				Params: []v1beta1.Param{
					{
						Name:  "input",
						Value: v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "param"},
					},
					{
						Name:  "input2",
						Value: v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "param2"},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
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

func TestTaskRunDescribe_step_status_running_multiple_sidecars_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: test.FakeClock().Now().Add(20 * time.Second)},
					Steps: []v1beta1.StepState{
						{
							Name: "step1",
							ContainerState: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{
									StartedAt: reasonRunning.StartedAt,
								},
							},
						},
						{
							Name: "step2",
							ContainerState: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{
									StartedAt: reasonRunning.StartedAt,
								},
							},
						},
					},
					Sidecars: []v1beta1.SidecarState{
						{
							Name: "sidecar1",
							ContainerState: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{
									StartedAt: reasonRunning.StartedAt,
								},
							},
						},
						{
							Name: "sidecar2",
							ContainerState: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{
									StartedAt: reasonRunning.StartedAt,
								},
							},
						},
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Type:   apis.ConditionSucceeded,
							Reason: "Running",
						},
					},
				},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "t1",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
				Params: []v1beta1.Param{
					{
						Name:  "input",
						Value: v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "param"},
					},
					{
						Name:  "input2",
						Value: v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "param2"},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
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

func TestTaskRunDescribe_cancel_taskrun_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(2 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  "TaskRunCancelled",
							Message: "TaskRun \"tr-1\" was cancelled",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
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

func TestTaskRunDescribe_custom_output_v1beta1(t *testing.T) {
	name := "task-run"
	expected := "taskrun.tekton.dev/" + name

	clock := test.FakeClock()

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
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

func TestTaskRunDescribe_WithSpec_custom_timeout_v1beta1(t *testing.T) {
	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-custom-timeout",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskRunSpec{
				Timeout: &metav1.Duration{Duration: time.Minute},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
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

func TestTaskRunDescribe_last_v1beta1(t *testing.T) {
	clock := test.FakeClock()
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
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(taskRuns[0], versionv1beta1),
		cb.UnstructuredV1beta1TR(taskRuns[1], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	taskrun := Command(p)
	got, err := test.ExecuteCommand(taskrun, "desc", "-n", "ns", "--last")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_With_Results_v1beta1(t *testing.T) {
	clock := test.FakeClock()
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
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.TaskRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
					TaskRunResults: []v1beta1.TaskRunResult{
						{
							Name: "result-1",
							Value: v1beta1.ParamValue{
								Type:      v1beta1.ParamTypeString,
								StringVal: "value-1",
							},
						},
						{
							Name: "result-2",
							Value: v1beta1.ParamValue{
								Type:      v1beta1.ParamTypeString,
								StringVal: "value-2",
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
				Name: "ns",
			},
		},
	}

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(taskRuns[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	taskrun := Command(p)
	got, err := test.ExecuteCommand(taskrun, "desc", "tr-1")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_zero_timeout_v1beta1(t *testing.T) {
	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-zero-timeout",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskRunSpec{
				Timeout: &metav1.Duration{Duration: 0},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: trs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	taskrun := Command(p)
	actual, err := test.ExecuteCommand(taskrun, "desc", "tr-zero-timeout", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_With_Workspaces_v1beta1(t *testing.T) {
	clock := test.FakeClock()
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
				Workspaces: []v1beta1.WorkspaceBinding{
					{
						Name:     "test",
						SubPath:  "test",
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
					{
						Name: "configmap",
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "bar"},
						},
					},
					{
						Name: "secret",
						Secret: &corev1.SecretVolumeSource{
							SecretName: "foobar",
						},
					},
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.TaskRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
					TaskRunResults: []v1beta1.TaskRunResult{
						{
							Name: "result-1",
							Value: v1beta1.ParamValue{
								Type:      v1beta1.ParamTypeString,
								StringVal: "value-1",
							},
						},
						{
							Name: "result-2",
							Value: v1beta1.ParamValue{
								Type:      v1beta1.ParamTypeString,
								StringVal: "value-2",
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
				Name: "ns",
			},
		},
	}

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(taskRuns[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	taskrun := Command(p)
	got, err := test.ExecuteCommand(taskrun, "desc", "tr-1")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_WithoutNameIfOnlyOneTaskRunPresent_v1beta1(t *testing.T) {
	clock := test.FakeClock()
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
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(taskRuns[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	taskrun := Command(p)
	out, err := test.ExecuteCommand(taskrun, "desc")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_with_annotations_v1beta1(t *testing.T) {
	taskRuns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-with-annotations",
				Annotations: map[string]string{
					corev1.LastAppliedConfigAnnotation: "LastAppliedConfig",
					"tekton.dev/tags":                  "testing",
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

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(taskRuns[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	taskrun := Command(p)
	got, err := test.ExecuteCommand(taskrun, "desc", "tr-with-annotations")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_empty_taskrun(t *testing.T) {
	clock := test.FakeClock()

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "t1"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "t1",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
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
	clock := test.FakeClock()

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: test.FakeClock().Now().Add(20 * time.Second)},
					Steps: []v1.StepState{
						{
							Name: "step1",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: reasonCompleted.Reason,
								},
							},
						},
						{
							Name: "step2",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: reasonCompleted.Reason,
								},
							},
						},
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "t1",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
				Params: []v1.Param{
					{
						Name:  "input",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "param"},
					},
					{
						Name:  "input2",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "param2"},
					},
				},
			},
		},
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
	clock := test.FakeClock()

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  v1.TaskRunReasonFailed.String(),
							Message: "Testing tr failed",
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(2 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "t1",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
			},
		},
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
	clock := test.FakeClock()

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(2 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  v1.TaskRunReasonFailed.String(),
							Message: "Testing tr failed",
						},
					},
				},
			},
		},
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
	var trs []*v1.TaskRun
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
	clock := test.FakeClock()

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: test.FakeClock().Now().Add(20 * time.Second)},
					Steps: []v1.StepState{
						{
							Name: "step1",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: reasonCompleted.Reason,
								},
							},
						},
						{
							Name: "step2",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: reasonCompleted.Reason,
								},
							},
						},
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "t1",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
				Params: []v1.Param{
					{
						Name:  "input",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "param"},
					},
					{
						Name:  "input2",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "param2"},
					},
				},
			},
		},
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
	clock := test.FakeClock()

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: test.FakeClock().Now().Add(20 * time.Second)},
					Steps: []v1.StepState{
						{
							Name: "step1",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: reasonFailed.Reason,
								},
							},
						},
						{
							Name: "step2",
						},
					},
					Sidecars: []v1.SidecarState{
						{
							Name: "sidecar1",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: reasonFailed.Reason,
								},
							},
						},
						{
							Name: "sidecar2",
						},
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1.TaskRunReasonFailed.String(),
						},
					},
				},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "t1",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
				Params: []v1.Param{
					{
						Name:  "input",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "param"},
					},
					{
						Name:  "input2",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "param2"},
					},
				},
			},
		},
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
	clock := test.FakeClock()

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: test.FakeClock().Now().Add(20 * time.Second)},
					Steps: []v1.StepState{
						{
							Name: "step1",
							ContainerState: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: reasonWaiting.Reason,
								},
							},
						},
						{
							Name: "step2",
							ContainerState: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: reasonWaiting.Reason,
								},
							},
						},
					},
					Sidecars: []v1.SidecarState{
						{
							Name: "sidecar1",
							ContainerState: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: reasonWaiting.Reason,
								},
							},
						},
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Type:   apis.ConditionSucceeded,
							Reason: "Running",
						},
					},
				},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "t1",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
				Params: []v1.Param{
					{
						Name:  "input",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "param"},
					},
					{
						Name:  "input2",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "param2"},
					},
				},
			},
		},
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
	clock := test.FakeClock()

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: test.FakeClock().Now().Add(20 * time.Second)},
					Steps: []v1.StepState{
						{
							Name: "step1",
							ContainerState: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{
									StartedAt: reasonRunning.StartedAt,
								},
							},
						},
						{
							Name: "step2",
							ContainerState: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{
									StartedAt: reasonRunning.StartedAt,
								},
							},
						},
					},
					Sidecars: []v1.SidecarState{
						{
							Name: "sidecar1",
							ContainerState: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{
									StartedAt: reasonRunning.StartedAt,
								},
							},
						},
						{
							Name: "sidecar2",
							ContainerState: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{
									StartedAt: reasonRunning.StartedAt,
								},
							},
						},
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Type:   apis.ConditionSucceeded,
							Reason: "Running",
						},
					},
				},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "t1",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
				Params: []v1.Param{
					{
						Name:  "input",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "param"},
					},
					{
						Name:  "input2",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "param2"},
					},
				},
			},
		},
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
	clock := test.FakeClock()

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(2 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  "TaskRunCancelled",
							Message: "TaskRun \"tr-1\" was cancelled",
						},
					},
				},
			},
		},
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

	clock := test.FakeClock()

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "ns",
			},
		},
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

func TestTaskRunDescribe_WithSpec_custom_timeout(t *testing.T) {
	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-custom-timeout",
				Namespace: "ns",
			},
			Spec: v1.TaskRunSpec{
				Timeout: &metav1.Duration{Duration: time.Minute},
			},
		},
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

func TestTaskRunDescribe_last(t *testing.T) {
	clock := test.FakeClock()
	taskRuns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-1",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task-1",
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1.TaskRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
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
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task-1",
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
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

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(taskRuns[0], version),
		cb.UnstructuredTR(taskRuns[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	taskrun := Command(p)
	got, err := test.ExecuteCommand(taskrun, "desc", "-n", "ns", "--last")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_With_Results(t *testing.T) {
	clock := test.FakeClock()
	taskRuns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-1",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task-1",
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1.TaskRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
					Results: []v1.TaskRunResult{
						{
							Name: "result-1",
							Value: v1.ParamValue{
								Type:      v1.ParamTypeString,
								StringVal: "value-1",
							},
						},
						{
							Name: "result-2",
							Value: v1.ParamValue{
								Type:      v1.ParamTypeString,
								StringVal: "value-2",
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
				Name: "ns",
			},
		},
	}

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(taskRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	taskrun := Command(p)
	got, err := test.ExecuteCommand(taskrun, "desc", "tr-1")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_zero_timeout(t *testing.T) {
	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-zero-timeout",
				Namespace: "ns",
			},
			Spec: v1.TaskRunSpec{
				Timeout: &metav1.Duration{Duration: 0},
			},
		},
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

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	taskrun := Command(p)
	actual, err := test.ExecuteCommand(taskrun, "desc", "tr-zero-timeout", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_With_Workspaces(t *testing.T) {
	clock := test.FakeClock()
	taskRuns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-1",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task-1",
				},
				Workspaces: []v1.WorkspaceBinding{
					{
						Name:     "test",
						SubPath:  "test",
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
					{
						Name: "configmap",
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "bar"},
						},
					},
					{
						Name: "secret",
						Secret: &corev1.SecretVolumeSource{
							SecretName: "foobar",
						},
					},
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1.TaskRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
					Results: []v1.TaskRunResult{
						{
							Name: "result-1",
							Value: v1.ParamValue{
								Type:      v1.ParamTypeString,
								StringVal: "value-1",
							},
						},
						{
							Name: "result-2",
							Value: v1.ParamValue{
								Type:      v1.ParamTypeString,
								StringVal: "value-2",
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
				Name: "ns",
			},
		},
	}

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(taskRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	taskrun := Command(p)
	got, err := test.ExecuteCommand(taskrun, "desc", "tr-1")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_WithoutNameIfOnlyOneTaskRunPresent(t *testing.T) {
	clock := test.FakeClock()
	taskRuns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-1",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task-1",
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1.TaskRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
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

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(taskRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	taskrun := Command(p)
	out, err := test.ExecuteCommand(taskrun, "desc")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunDescribe_with_annotations(t *testing.T) {
	taskRuns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-with-annotations",
				Annotations: map[string]string{
					corev1.LastAppliedConfigAnnotation: "LastAppliedConfig",
					"tekton.dev/tags":                  "testing",
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

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(taskRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	taskrun := Command(p)
	got, err := test.ExecuteCommand(taskrun, "desc", "tr-with-annotations")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}
