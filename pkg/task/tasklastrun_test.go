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

package task

import (
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestTaskrunLatestName_two_run_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	var (
		firstRunCreated   = clock.Now().Add(10 * time.Minute)
		firstRunStarted   = firstRunCreated.Add(2 * time.Second)
		firstRunCompleted = firstRunStarted.Add(10 * time.Minute)

		secondRunCreated   = firstRunCreated.Add(1 * time.Minute)
		secondRunStarted   = secondRunCreated.Add(2 * time.Second)
		secondRunCompleted = secondRunStarted.Add(5 * time.Minute)
	)
	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-1",
				Namespace:         "ns",
				Labels:            map[string]string{"tekton.dev/task": "task"},
				CreationTimestamp: metav1.Time{Time: firstRunCreated},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
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
					StartTime:      &metav1.Time{Time: firstRunStarted},
					CompletionTime: &metav1.Time{Time: firstRunCompleted},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-2",
				Namespace:         "ns",
				Labels:            map[string]string{"tekton.dev/task": "task"},
				CreationTimestamp: metav1.Time{Time: secondRunCompleted},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
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
					StartTime:      &metav1.Time{Time: secondRunStarted},
					CompletionTime: &metav1.Time{Time: secondRunCompleted},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-3",
				Namespace:         "ns",
				Labels:            map[string]string{"tekton.dev/clusterTask": "task"},
				CreationTimestamp: metav1.Time{Time: secondRunCompleted},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.ClusterTaskKind,
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
					StartTime:      &metav1.Time{Time: secondRunStarted},
					CompletionTime: &metav1.Time{Time: secondRunCompleted},
				},
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: taskruns,
	})
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1TR(taskruns[0], "v1beta1"),
		cb.UnstructuredV1beta1TR(taskruns[1], "v1beta1"),
		cb.UnstructuredV1beta1TR(taskruns[2], "v1beta1"),
	)
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Dynamic: dc}
	client, err := p.Clients()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	lastRun, err := LastRunName(client, "task", "ns", "Task")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "tr-2", lastRun)
}

func TestTaskrunLatest_two_run_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	var (
		firstRunCreated   = clock.Now().Add(10 * time.Minute)
		firstRunStarted   = firstRunCreated.Add(2 * time.Second)
		firstRunCompleted = firstRunStarted.Add(10 * time.Minute)

		secondRunCreated   = firstRunCreated.Add(1 * time.Minute)
		secondRunStarted   = secondRunCreated.Add(2 * time.Second)
		secondRunCompleted = secondRunStarted.Add(5 * time.Minute)
	)
	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-1",
				Namespace:         "ns",
				Labels:            map[string]string{"tekton.dev/task": "task"},
				CreationTimestamp: metav1.Time{Time: firstRunCreated},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
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
					StartTime:      &metav1.Time{Time: firstRunStarted},
					CompletionTime: &metav1.Time{Time: firstRunCompleted},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-2",
				Namespace:         "ns",
				Labels:            map[string]string{"tekton.dev/task": "task"},
				CreationTimestamp: metav1.Time{Time: secondRunCompleted},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
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
					StartTime:      &metav1.Time{Time: secondRunStarted},
					CompletionTime: &metav1.Time{Time: secondRunCompleted},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-3",
				Namespace:         "ns",
				Labels:            map[string]string{"tekton.dev/clusterTask": "task"},
				CreationTimestamp: metav1.Time{Time: secondRunCompleted},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.ClusterTaskKind,
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
					StartTime:      &metav1.Time{Time: secondRunStarted},
					CompletionTime: &metav1.Time{Time: secondRunCompleted},
				},
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: taskruns,
	})
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1TR(taskruns[0], "v1beta1"),
		cb.UnstructuredV1beta1TR(taskruns[1], "v1beta1"),
		cb.UnstructuredV1beta1TR(taskruns[2], "v1beta1"),
	)
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Dynamic: dc}
	client, err := p.Clients()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	lastRun, err := LastRun(client, "task", "ns", "Task")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "tr-2", lastRun.Name)
}

func TestTaskrunLatest_no_run_v1beta1(t *testing.T) {

	clock := test.FakeClock()
	cs, _ := test.SeedV1beta1TestData(t, test.Data{})
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client()
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Dynamic: dc}
	client, err := p.Clients()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	_, err = LastRun(client, "task", "ns", "Task")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	expected := "no TaskRuns related to Task task found in namespace ns"
	test.AssertOutput(t, expected, err.Error())
}

func TestTaskrunLatestForClusterTask_two_run_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	var (
		firstRunCreated   = clock.Now().Add(10 * time.Minute)
		firstRunStarted   = firstRunCreated.Add(2 * time.Second)
		firstRunCompleted = firstRunStarted.Add(10 * time.Minute)

		secondRunCreated   = firstRunCreated.Add(1 * time.Minute)
		secondRunStarted   = secondRunCreated.Add(2 * time.Second)
		secondRunCompleted = secondRunStarted.Add(5 * time.Minute)
	)
	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-1",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: firstRunCreated},
				Labels:            map[string]string{"tekton.dev/clusterTask": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.ClusterTaskKind,
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
					StartTime:      &metav1.Time{Time: firstRunStarted},
					CompletionTime: &metav1.Time{Time: firstRunCompleted},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-2",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: secondRunCompleted},
				Labels:            map[string]string{"tekton.dev/clusterTask": "task", "tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.ClusterTaskKind,
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
					StartTime:      &metav1.Time{Time: secondRunStarted},
					CompletionTime: &metav1.Time{Time: secondRunCompleted},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-3",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: secondRunCompleted},
				Labels:            map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
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
					StartTime:      &metav1.Time{Time: secondRunStarted},
					CompletionTime: &metav1.Time{Time: secondRunCompleted},
				},
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns: taskruns,
	})
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1TR(taskruns[0], "v1beta1"),
		cb.UnstructuredV1beta1TR(taskruns[1], "v1beta1"),
		cb.UnstructuredV1beta1TR(taskruns[2], "v1beta1"),
	)
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Dynamic: dc}
	client, err := p.Clients()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	lastRun, err := LastRun(client, "task", "ns", "ClusterTask")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "tr-2", lastRun.Name)
}

func TestTaskrunLatestName_two_run(t *testing.T) {
	clock := test.FakeClock()

	var (
		firstRunCreated   = clock.Now().Add(10 * time.Minute)
		firstRunStarted   = firstRunCreated.Add(2 * time.Second)
		firstRunCompleted = firstRunStarted.Add(10 * time.Minute)

		secondRunCreated   = firstRunCreated.Add(1 * time.Minute)
		secondRunStarted   = secondRunCreated.Add(2 * time.Second)
		secondRunCompleted = secondRunStarted.Add(5 * time.Minute)
	)
	taskruns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-1",
				Namespace:         "ns",
				Labels:            map[string]string{"tekton.dev/task": "task"},
				CreationTimestamp: metav1.Time{Time: firstRunCreated},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
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
					StartTime:      &metav1.Time{Time: firstRunStarted},
					CompletionTime: &metav1.Time{Time: firstRunCompleted},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-2",
				Namespace:         "ns",
				Labels:            map[string]string{"tekton.dev/task": "task"},
				CreationTimestamp: metav1.Time{Time: secondRunCompleted},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
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
					StartTime:      &metav1.Time{Time: secondRunStarted},
					CompletionTime: &metav1.Time{Time: secondRunCompleted},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-3",
				Namespace:         "ns",
				Labels:            map[string]string{"tekton.dev/clusterTask": "task"},
				CreationTimestamp: metav1.Time{Time: secondRunCompleted},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: "ClusterTask",
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
					StartTime:      &metav1.Time{Time: secondRunStarted},
					CompletionTime: &metav1.Time{Time: secondRunCompleted},
				},
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: taskruns,
	})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredTR(taskruns[0], "v1"),
		cb.UnstructuredTR(taskruns[1], "v1"),
		cb.UnstructuredTR(taskruns[2], "v1"),
	)
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Dynamic: dc}
	client, err := p.Clients()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	lastRun, err := LastRunName(client, "task", "ns", "Task")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "tr-2", lastRun)
}

func TestTaskrunLatest_two_run(t *testing.T) {
	clock := test.FakeClock()

	var (
		firstRunCreated   = clock.Now().Add(10 * time.Minute)
		firstRunStarted   = firstRunCreated.Add(2 * time.Second)
		firstRunCompleted = firstRunStarted.Add(10 * time.Minute)

		secondRunCreated   = firstRunCreated.Add(1 * time.Minute)
		secondRunStarted   = secondRunCreated.Add(2 * time.Second)
		secondRunCompleted = secondRunStarted.Add(5 * time.Minute)
	)
	taskruns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-1",
				Namespace:         "ns",
				Labels:            map[string]string{"tekton.dev/task": "task"},
				CreationTimestamp: metav1.Time{Time: firstRunCreated},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
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
					StartTime:      &metav1.Time{Time: firstRunStarted},
					CompletionTime: &metav1.Time{Time: firstRunCompleted},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-2",
				Namespace:         "ns",
				Labels:            map[string]string{"tekton.dev/task": "task"},
				CreationTimestamp: metav1.Time{Time: secondRunCompleted},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
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
					StartTime:      &metav1.Time{Time: secondRunStarted},
					CompletionTime: &metav1.Time{Time: secondRunCompleted},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-3",
				Namespace:         "ns",
				Labels:            map[string]string{"tekton.dev/clusterTask": "task"},
				CreationTimestamp: metav1.Time{Time: secondRunCompleted},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: "ClusterTask",
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
					StartTime:      &metav1.Time{Time: secondRunStarted},
					CompletionTime: &metav1.Time{Time: secondRunCompleted},
				},
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: taskruns,
	})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredTR(taskruns[0], "v1"),
		cb.UnstructuredTR(taskruns[1], "v1"),
		cb.UnstructuredTR(taskruns[2], "v1"),
	)
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Dynamic: dc}
	client, err := p.Clients()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	lastRun, err := LastRun(client, "task", "ns", "Task")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "tr-2", lastRun.Name)
}

func TestTaskrunLatest_no_run(t *testing.T) {

	clock := test.FakeClock()
	cs, _ := test.SeedTestData(t, pipelinetest.Data{})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client()
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Dynamic: dc}
	client, err := p.Clients()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	_, err = LastRun(client, "task", "ns", "Task")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	expected := "no TaskRuns related to Task task found in namespace ns"
	test.AssertOutput(t, expected, err.Error())
}

func TestTaskrunLatestForClusterTask_two_run(t *testing.T) {
	clock := test.FakeClock()

	var (
		firstRunCreated   = clock.Now().Add(10 * time.Minute)
		firstRunStarted   = firstRunCreated.Add(2 * time.Second)
		firstRunCompleted = firstRunStarted.Add(10 * time.Minute)

		secondRunCreated   = firstRunCreated.Add(1 * time.Minute)
		secondRunStarted   = secondRunCreated.Add(2 * time.Second)
		secondRunCompleted = secondRunStarted.Add(5 * time.Minute)
	)
	taskruns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-1",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: firstRunCreated},
				Labels:            map[string]string{"tekton.dev/clusterTask": "task"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: "ClusterTask",
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
					StartTime:      &metav1.Time{Time: firstRunStarted},
					CompletionTime: &metav1.Time{Time: firstRunCompleted},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-2",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: secondRunCompleted},
				Labels:            map[string]string{"tekton.dev/clusterTask": "task", "tekton.dev/task": "task"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: "ClusterTask",
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
					StartTime:      &metav1.Time{Time: secondRunStarted},
					CompletionTime: &metav1.Time{Time: secondRunCompleted},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-3",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: secondRunCompleted},
				Labels:            map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
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
					StartTime:      &metav1.Time{Time: secondRunStarted},
					CompletionTime: &metav1.Time{Time: secondRunCompleted},
				},
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: taskruns,
	})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredTR(taskruns[0], "v1"),
		cb.UnstructuredTR(taskruns[1], "v1"),
		cb.UnstructuredTR(taskruns[2], "v1"),
	)
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Dynamic: dc}
	client, err := p.Clients()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	lastRun, err := LastRun(client, "task", "ns", "ClusterTask")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "tr-2", lastRun.Name)
}

func TestFilterByRef(t *testing.T) {
	clock := test.FakeClock()

	var (
		firstRunCreated   = clock.Now().Add(10 * time.Minute)
		firstRunStarted   = firstRunCreated.Add(2 * time.Second)
		firstRunCompleted = firstRunStarted.Add(10 * time.Minute)

		secondRunCreated   = firstRunCreated.Add(1 * time.Minute)
		secondRunStarted   = secondRunCreated.Add(2 * time.Second)
		secondRunCompleted = secondRunStarted.Add(5 * time.Minute)
	)
	taskruns := []v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-1",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: firstRunCreated},
				Labels:            map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
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
					StartTime:      &metav1.Time{Time: firstRunStarted},
					CompletionTime: &metav1.Time{Time: firstRunCompleted},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tr-2",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: secondRunCompleted},
				Labels:            map[string]string{"tekton.dev/clusterTask": "task"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: "ClusterTask",
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
					StartTime:      &metav1.Time{Time: secondRunStarted},
					CompletionTime: &metav1.Time{Time: secondRunCompleted},
				},
			},
		},
	}

	filteredClusterTask := FilterByRef(taskruns, "ClusterTask")

	test.AssertOutput(t, "tr-2", filteredClusterTask[0].Name)

	filteredTask := FilterByRef(taskruns, "Task")

	test.AssertOutput(t, "tr-1", filteredTask[0].Name)
}
