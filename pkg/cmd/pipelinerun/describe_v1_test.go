// Copyright © 2023 The Tekton Authors.
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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestPipelineRunDescribe_invalid_namespace(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	pipelinerun := Command(p)
	out, err := test.ExecuteCommand(pipelinerun, "desc", "bar", "-n", "invalid")
	if err == nil {
		t.Errorf("Expected error for invalid namespace")
	}
	expected := "Error: failed to find pipelinerun \"bar\"\n"
	test.AssertOutput(t, expected, out)
}

func TestPipelineRunDescribe_not_found(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	_, err = test.ExecuteCommand(pipelinerun, "desc", "bar", "-n", "ns")
	if err == nil {
		t.Errorf("Expected error, did not get any")
	}
	expected := "failed to find pipelinerun \"bar\""
	test.AssertOutput(t, expected, err.Error())
}

func TestPipelineRunDescribe_only_taskrun(t *testing.T) {
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
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
		},
	}

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
				Timeouts: &v1.TimeoutFields{
					Pipeline: &metav1.Duration{Duration: 1 * time.Hour},
				},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.PipelineRunReasonSuccessful.String(),
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_multiple_taskrun_ordering(t *testing.T) {
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
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-2",
				Namespace: "ns",
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(9 * time.Minute)},
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
		},
	}

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
				Timeouts: &v1.TimeoutFields{
					Pipeline: &metav1.Duration{Duration: 1 * time.Hour},
				},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
						{
							Name:             "tr-2",
							PipelineTaskName: "t-2",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(15 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.PipelineRunReasonSuccessful.String(),
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
		cb.UnstructuredTR(trs[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))

}

func TestPipelineRunDescribe_multiple_taskrun_without_status(t *testing.T) {
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
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-0",
				Namespace: "ns",
			},
			Status: v1.TaskRunStatus{},
		},
	}

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1.PipelineRunSpec{
				Timeouts: &v1.TimeoutFields{
					Pipeline: &metav1.Duration{Duration: 1 * time.Hour},
				},
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
						{
							Name:             "tr-0",
							PipelineTaskName: "t-0",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1.PipelineRunReasonFailed.String(),
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
		cb.UnstructuredTR(trs[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_failed(t *testing.T) {
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
							Reason:  v1.PipelineRunReasonFailed.String(),
							Message: "Testing tr failed",
						},
					},
				},
			},
		},
	}

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1.PipelineRunSpec{
				Timeouts: &v1.TimeoutFields{
					Pipeline: &metav1.Duration{Duration: 1 * time.Hour},
				},
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
				TaskRunTemplate: v1.PipelineTaskRunTemplate{
					ServiceAccountName: "test-sa",
				},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  "Resource not found",
							Message: "Resource test-resource not found in the pipelinerun",
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_last_no_PipelineRun_present(t *testing.T) {
	pipelineRuns := []*v1.PipelineRun{}
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns})

	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	out, err := test.ExecuteCommand(pipelinerun, "desc", "--last", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := "No PipelineRuns present in namespace ns\n"
	test.AssertOutput(t, expected, out)
}

func TestPipelineRunDescribe_failed_withoutTRCondition(t *testing.T) {
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
			},
		},
	}

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1.PipelineRunSpec{
				Timeouts: &v1.TimeoutFields{
					Pipeline: &metav1.Duration{Duration: 1 * time.Hour},
				},
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
				TaskRunTemplate: v1.PipelineTaskRunTemplate{
					ServiceAccountName: "test-sa",
				},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  "Resource not found",
							Message: "Resource test-resource not found in the pipelinerun",
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_failed_withoutPRCondition(t *testing.T) {
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
			},
		},
	}

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1.PipelineRunSpec{
				Timeouts: &v1.TimeoutFields{
					Pipeline: &metav1.Duration{Duration: 1 * time.Hour},
				},
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
				TaskRunTemplate: v1.PipelineTaskRunTemplate{
					ServiceAccountName: "test-sa",
				},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_with_resources_taskrun(t *testing.T) {
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
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
		},
	}

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1.PipelineRunSpec{
				Timeouts: &v1.TimeoutFields{
					Pipeline: &metav1.Duration{Duration: 1 * time.Hour},
				},
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
				TaskRunTemplate: v1.PipelineTaskRunTemplate{
					ServiceAccountName: "test-sa",
				},
				Params: []v1.Param{
					{
						Name: "test-param",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "param-value",
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.PipelineRunReasonSuccessful.String(),
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_without_start_time(t *testing.T) {
	clock := test.FakeClock()

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1.PipelineRunSpec{
				Timeouts: &v1.TimeoutFields{
					Pipeline: &metav1.Duration{Duration: 1 * time.Hour},
				},
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_without_pipelineref(t *testing.T) {
	clock := test.FakeClock()

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_withoutNameOfOnlyOnePipelineRunPresent(t *testing.T) {
	clock := test.FakeClock()

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	p.SetNamespace("ns")
	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_no_resourceref(t *testing.T) {
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
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
		},
	}

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1.PipelineRunSpec{
				Timeouts: &v1.TimeoutFields{
					Pipeline: &metav1.Duration{Duration: 1 * time.Hour},
				},
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
				TaskRunTemplate: v1.PipelineTaskRunTemplate{
					ServiceAccountName: "test-sa",
				},
				Params: []v1.Param{
					{
						Name: "test-param",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "param-value",
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.PipelineRunReasonSuccessful.String(),
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_cancelled_pipelinerun(t *testing.T) {
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
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
		},
	}

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1.PipelineRunSpec{
				Timeouts: &v1.TimeoutFields{
					Pipeline: &metav1.Duration{Duration: 1 * time.Hour},
				},
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  "PipelineRunCancelled",
							Message: "PipelineRun \"pipeline-run\" was cancelled",
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_without_tr_start_time(t *testing.T) {
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
							Status: corev1.ConditionUnknown,
							Type:   apis.ConditionReady,
						},
					},
				},
			},
		},
	}

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1.PipelineRunSpec{
				Timeouts: &v1.TimeoutFields{
					Pipeline: &metav1.Duration{Duration: 1 * time.Hour},
				},
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
					StartTime: &metav1.Time{Time: clock.Now()},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: v1.PipelineRunReasonRunning.String(),
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_custom_timeout(t *testing.T) {
	prun := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pr-custom-timeout",
				Namespace: "ns",
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pr-custom-timeout",
				},
				Timeouts: &v1.TimeoutFields{
					Pipeline: &metav1.Duration{Duration: 1 * time.Minute},
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(prun[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: prun})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pr-custom-timeout", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_custom_output(t *testing.T) {
	pipelinerunname := "pipeline-run"
	expected := "pipelinerun.tekton.dev/" + pipelinerunname
	prun := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname,
				Namespace: "ns",
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(prun[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: prun})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-o", "name", "-n", "ns", pipelinerunname)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	got = strings.TrimSpace(got)
	if got != expected {
		t.Errorf("Result should be '%s' != '%s'", got, expected)
	}
}

func TestPipelineRunDescribe(t *testing.T) {
	clock := test.FakeClock()
	pipelinerunname := "pipeline-run"
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
							Reason: v1.PipelineRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
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
							Reason: v1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(17 * time.Minute)},
				},
			},
		},
	}

	prun := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname,
				Namespace: "ns",
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
				Params: []v1.Param{
					{
						Name: "p-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "p-2",
						Value: v1.ParamValue{
							Type:     v1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Reason:  v1.PipelineRunReasonSuccessful.String(),
							Message: "Completed",
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(20 * time.Minute)},
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
						{
							Name:             "tr-2",
							PipelineTaskName: "t-2",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(prun[0], version),
		cb.UnstructuredTR(taskRuns[0], version),
		cb.UnstructuredTR(taskRuns[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: prun, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-n", "ns", pipelinerunname)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_taskrun_with_no_status(t *testing.T) {
	clock := test.FakeClock()
	pipelinerunname := "pipeline-run"
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
							Reason: v1.PipelineRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
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
		},
	}

	prun := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname,
				Namespace: "ns",
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
				Params: []v1.Param{
					{
						Name: "p-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "p-2",
						Value: v1.ParamValue{
							Type:     v1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Reason:  v1.PipelineRunReasonSuccessful.String(),
							Message: "Completed",
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(20 * time.Minute)},
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
						{
							Name:             "tr-2",
							PipelineTaskName: "t-2",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(prun[0], version),
		cb.UnstructuredTR(taskRuns[0], version),
		cb.UnstructuredTR(taskRuns[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: prun, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-n", "ns", pipelinerunname)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_last(t *testing.T) {
	clock := test.FakeClock()
	pipelinerunname1 := "pipeline-run"
	pipelinerunname2 := "pipeline-run2"
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
							Reason: v1.PipelineRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
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
							Reason: v1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(-3 * time.Minute)},
				},
			},
		},
	}

	prun := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname1,
				Namespace: "ns",
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
				Params: []v1.Param{
					{
						Name: "p-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "p-2",
						Value: v1.ParamValue{
							Type:     v1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Reason:  v1.PipelineRunReasonSuccessful.String(),
							Message: "Completed",
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
						{
							Name:             "tr-2",
							PipelineTaskName: "t-2",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname2,
				Namespace: "ns",
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline2",
				},
				Params: []v1.Param{
					{
						Name: "p-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "p-2",
						Value: v1.ParamValue{
							Type:     v1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Reason:  v1.PipelineRunReasonSuccessful.String(),
							Message: "Completed",
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(20 * time.Minute)},
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
						{
							Name:             "tr-2",
							PipelineTaskName: "t-2",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(prun[0], version),
		cb.UnstructuredPR(prun[1], version),
		cb.UnstructuredTR(taskRuns[0], version),
		cb.UnstructuredTR(taskRuns[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: prun, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-n", "ns", "--last")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_with_results(t *testing.T) {
	clock := test.FakeClock()
	pipelinerunname := "pipeline-run"
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
							Reason: v1.PipelineRunReasonFailed.String(),
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

	prun := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname,
				Namespace: "ns",
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
				Params: []v1.Param{
					{
						Name: "p-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Reason:  v1.PipelineRunReasonSuccessful.String(),
							Message: "Completed",
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
					Results: []v1.PipelineRunResult{
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(prun[0], version),
		cb.UnstructuredTR(taskRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: prun, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-n", "ns", "--last")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_zero_timeout(t *testing.T) {
	prun := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline-run-zero-timeout",
				Namespace: "ns",
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline-zero-timeout",
				},
				Timeouts: &v1.TimeoutFields{
					Pipeline: &metav1.Duration{Duration: 0},
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(prun[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: prun})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run-zero-timeout", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_with_workspaces(t *testing.T) {
	clock := test.FakeClock()
	pipelinerunname := "pipeline-run"
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
							Reason: v1.PipelineRunReasonFailed.String(),
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

	prun := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname,
				Namespace: "ns",
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
				Params: []v1.Param{
					{
						Name: "p-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
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
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Reason:  v1.PipelineRunReasonSuccessful.String(),
							Message: "Completed",
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
					Results: []v1.PipelineRunResult{
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(prun[0], version),
		cb.UnstructuredTR(taskRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: prun, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-n", "ns", "--last")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribeWithSkippedTasks(t *testing.T) {
	clock := test.FakeClock()
	pipelinerunname := "pipeline-run"
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
							Reason: v1.PipelineRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
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
							Reason: v1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(17 * time.Minute)},
				},
			},
		},
	}

	prun := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname,
				Namespace: "ns",
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
				Params: []v1.Param{
					{
						Name: "p-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "p-2",
						Value: v1.ParamValue{
							Type:     v1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Reason:  v1.PipelineRunReasonSuccessful.String(),
							Message: "Completed",
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(20 * time.Minute)},
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
						{
							Name:             "tr-2",
							PipelineTaskName: "t-2",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
					SkippedTasks: []v1.SkippedTask{
						{
							Name: "task-should-be-skipped-1",
							WhenExpressions: []v1.WhenExpression{
								{
									Input:    "yes",
									Operator: selection.In,
									Values:   []string{"missing"},
								},
							},
						},
						{
							Name: "task-should-be-skipped-2",
							WhenExpressions: []v1.WhenExpression{
								{
									Input:    "README.md",
									Operator: selection.NotIn,
									Values:   []string{"README.md"},
								},
							},
						},
						{
							Name: "task-should-be-skipped-3",
							WhenExpressions: []v1.WhenExpression{
								{
									Input:    "monday",
									Operator: selection.NotIn,
									Values:   []string{"friday"},
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
				Name: "ns",
			},
		},
	}

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(prun[0], version),
		cb.UnstructuredTR(taskRuns[0], version),
		cb.UnstructuredTR(taskRuns[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: prun, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-n", "ns", pipelinerunname)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_cancelled_pipelinerun_multiple_taskrun(t *testing.T) {
	clock := test.FakeClock()

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(1 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(3 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Type:    apis.ConditionSucceeded,
							Reason:  v1.TaskRunReasonCancelled.String(),
							Message: "TaskRun \"tr-1\" was cancelled",
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-2",
				Namespace: "ns",
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(2 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(4 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Type:    apis.ConditionSucceeded,
							Reason:  v1.TaskRunReasonCancelled.String(),
							Message: "TaskRun \"tr-2\" was cancelled",
						},
					},
				},
			},
		},
	}

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1.PipelineRunSpec{
				Timeouts: &v1.TimeoutFields{
					Pipeline: &metav1.Duration{Duration: 1 * time.Hour},
				},
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
						{
							Name:             "tr-2",
							PipelineTaskName: "t-2",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  "PipelineRunCancelled",
							Message: "PipelineRun \"pipeline-run\" was cancelled",
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
		cb.UnstructuredTR(trs[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribeWithTimeouts(t *testing.T) {
	clock := test.FakeClock()
	pipelinerunname := "pipeline-run"
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
							Reason: v1.PipelineRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
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
							Reason: v1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(17 * time.Minute)},
				},
			},
		},
	}

	prun := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname,
				Namespace: "ns",
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
				Timeouts: &v1.TimeoutFields{
					Tasks:    &metav1.Duration{Duration: 50 * time.Minute},
					Pipeline: &metav1.Duration{Duration: 1 * time.Hour},
				},
				Params: []v1.Param{
					{
						Name: "p-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "p-2",
						Value: v1.ParamValue{
							Type:     v1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Reason:  v1.PipelineRunReasonSuccessful.String(),
							Message: "Completed",
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(20 * time.Minute)},
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
						{
							Name:             "tr-2",
							PipelineTaskName: "t-2",
							TypeMeta: runtime.TypeMeta{
								Kind: "TaskRun",
							},
						},
					},
					SkippedTasks: []v1.SkippedTask{
						{
							Name: "task-should-be-skipped-1",
							WhenExpressions: []v1.WhenExpression{
								{
									Input:    "yes",
									Operator: selection.In,
									Values:   []string{"missing"},
								},
							},
						},
						{
							Name: "task-should-be-skipped-2",
							WhenExpressions: []v1.WhenExpression{
								{
									Input:    "README.md",
									Operator: selection.NotIn,
									Values:   []string{"README.md"},
								},
							},
						},
						{
							Name: "task-should-be-skipped-3",
							WhenExpressions: []v1.WhenExpression{
								{
									Input:    "monday",
									Operator: selection.NotIn,
									Values:   []string{"friday"},
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
				Name: "ns",
			},
		},
	}

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(prun[0], version),
		cb.UnstructuredTR(taskRuns[0], version),
		cb.UnstructuredTR(taskRuns[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: prun, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-n", "ns", pipelinerunname)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_with_annotations(t *testing.T) {
	prun := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pr-with-annotations",
				Namespace: "ns",
				Annotations: map[string]string{
					corev1.LastAppliedConfigAnnotation: "LastAppliedConfig",
					"tekton.dev/tags":                  "testing",
				},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pr-with-annotations",
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(prun[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: prun})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pr-with-annotations", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}
