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
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	trh "github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stest "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestTracker_pipelinerun_complete(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		ns           = "namespace"

		task1Name = "output-task-1"
		tr1Name   = "output-task-1"
		tr1Pods   = "output-task-1-pods-123456"

		task2Name = "output-task-2"
		tr2Name   = "output-task-2"
		tr2Pods   = "output-task-2-pods-123456"

		allTasks  = []string{task1Name, task2Name}
		onlyTask1 = []string{task1Name}
	)

	scenarios := []struct {
		name     string
		tasks    []string
		expected []trh.Run
	}{
		{
			name:  "for all tasks",
			tasks: allTasks,
			expected: []trh.Run{
				{
					Name: tr1Name,
					Task: task1Name,
				}, {
					Name: tr2Name,
					Task: task2Name,
				},
			},
		}, {
			name:  "for one task",
			tasks: onlyTask1,
			expected: []trh.Run{
				{
					Name: tr1Name,
					Task: task1Name,
				},
			},
		},
	}

	for _, s := range scenarios {
		taskruns := []*v1.TaskRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tr1Name,
					Namespace: ns,
				},
				Spec: v1.TaskRunSpec{
					TaskRef: &v1.TaskRef{
						Name: task1Name,
					},
				},
				Status: v1.TaskRunStatus{
					TaskRunStatusFields: v1.TaskRunStatusFields{
						PodName: tr1Pods,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tr2Name,
					Namespace: ns,
				},
				Spec: v1.TaskRunSpec{
					TaskRef: &v1.TaskRef{
						Name: task2Name,
					},
				},
				Status: v1.TaskRunStatus{
					TaskRunStatusFields: v1.TaskRunStatusFields{
						PodName: tr2Pods,
					},
				},
			},
		}

		initialPR := []*v1.PipelineRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      prName,
					Namespace: ns,
					Labels:    map[string]string{"tekton.dev/pipeline": prName},
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Status: corev1.ConditionUnknown,
								Reason: v1.PipelineRunReasonRunning.String(),
							},
						},
					},
					PipelineRunStatusFields: v1.PipelineRunStatusFields{
						ChildReferences: []v1.ChildStatusReference{
							{
								Name:             tr1Name,
								PipelineTaskName: task1Name,
								TypeMeta: runtime.TypeMeta{
									APIVersion: "tekton.dev/v1",
									Kind:       "TaskRun",
								},
							},
						},
					},
				},
			},
		}

		pr := &v1.PipelineRun{
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             tr1Name,
							PipelineTaskName: task1Name,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1",
								Kind:       "TaskRun",
							},
						}, {
							Name:             tr2Name,
							PipelineTaskName: task2Name,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1",
								Kind:       "TaskRun",
							},
						},
					},
				},
			},
		}

		tc := startPipelineRun(t, pipelinetest.Data{PipelineRuns: initialPR, TaskRuns: taskruns}, pr.Status)
		tracker := NewTracker(pipelineName, ns, tc)
		if err := actions.InitializeAPIGroupRes(tc.Tekton.Discovery()); err != nil {
			t.Errorf("failed to initialize APIGroup Resource")
		}
		output := taskRunsFor(s.tasks, tracker)

		if d := cmp.Diff(s.expected, output, cmpopts.SortSlices(func(i, j trh.Run) bool { return i.Name < j.Name })); d != "" {
			t.Errorf("Unexpected output: %s", diff.PrintWantGot(d))
		}

	}
}

func taskRunsFor(onlyTasks []string, tracker *Tracker) []trh.Run {
	output := []trh.Run{}
	for ts := range tracker.Monitor(onlyTasks) {
		output = append(output, ts...)
	}
	return output
}

func startPipelineRun(t *testing.T, data pipelinetest.Data, prStatus ...v1.PipelineRunStatus) *cli.Clients {
	cs, _ := test.SeedTestData(t, data)

	// to keep pushing the taskrun over the period(simulate watch)
	watcher := watch.NewFake()
	cs.Pipeline.PrependWatchReactor("pipelineruns", k8stest.DefaultWatchReactor(watcher, nil))
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"task", "taskrun", "pipeline", "pipelinerun"})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(data.PipelineRuns[0], "v1"),
		cb.UnstructuredTR(data.TaskRuns[0], "v1"),
		cb.UnstructuredTR(data.TaskRuns[1], "v1"),
	)
	if err != nil {
		fmt.Println(err)
	}

	go func() {
		for _, status := range prStatus {
			time.Sleep(time.Second * 2)
			data.PipelineRuns[0].Status = status
			watcher.Modify(data.PipelineRuns[0])
		}
	}()

	return &cli.Clients{
		Tekton:  cs.Pipeline,
		Kube:    cs.Kube,
		Dynamic: dynamic,
	}
}

func TestTracker_watchErrorHandler(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "context.Canceled should be filtered",
			err:  context.Canceled,
		},
		{
			name: "wrapped context.Canceled should be filtered",
			err:  errors.Join(errors.New("watch failed"), context.Canceled),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// Call watchErrorHandler with context.Canceled errors
			// These should be filtered (not passed to DefaultWatchErrorHandler)
			// so passing nil reflector is safe
			watchErrorHandler(context.Background(), nil, tt.err)
		})
	}
}

func TestGetTaskRunsWithStatus_DirectTaskRuns(t *testing.T) {
	ns := "namespace"
	trName := "tr-1"
	taskName := "task-1"
	tr := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{Name: trName, Namespace: ns},
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{
				PodName:   "pod-1",
				StartTime: &metav1.Time{Time: time.Now()},
			},
			Status: duckv1.Status{
				Conditions: duckv1.Conditions{{
					Status: corev1.ConditionTrue,
					Type:   apis.ConditionSucceeded,
				}},
			},
		},
	}
	pr := &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{Name: "pr", Namespace: ns},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				ChildReferences: []v1.ChildStatusReference{{
					Name:             trName,
					PipelineTaskName: taskName,
					TypeMeta:         runtime.TypeMeta{Kind: "TaskRun"},
				}},
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		PipelineRuns: []*v1.PipelineRun{pr},
		TaskRuns:     []*v1.TaskRun{tr},
		Namespaces:   []*corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: ns}}},
	})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"taskrun", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(pr, "v1"),
		cb.UnstructuredTR(tr, "v1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	clients := &cli.Clients{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	if err := actions.InitializeAPIGroupRes(clients.Tekton.Discovery()); err != nil {
		t.Fatal(err)
	}
	result, err := GetTaskRunsWithStatus(pr, clients, ns)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 TaskRun, got %d", len(result))
	}
	trs, ok := result[trName]
	if !ok {
		t.Fatalf("expected TaskRun %q in result", trName)
	}
	if trs.PipelineTaskName != taskName {
		t.Errorf("expected PipelineTaskName %q, got %q", taskName, trs.PipelineTaskName)
	}
}
func TestGetTaskRunsWithStatus_ChildPipelineRun(t *testing.T) {
	ns := "namespace"
	childTRName := "parent-run-call-child-greet"
	childTaskName := "greet"
	parentTaskName := "call-child"
	childTR := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{Name: childTRName, Namespace: ns},
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{
				PodName:   "pod-greet",
				StartTime: &metav1.Time{Time: time.Now()},
			},
			Status: duckv1.Status{
				Conditions: duckv1.Conditions{{
					Status: corev1.ConditionTrue,
					Type:   apis.ConditionSucceeded,
				}},
			},
		},
	}
	childPR := &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{Name: "parent-run-call-child", Namespace: ns},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				ChildReferences: []v1.ChildStatusReference{{
					Name:             childTRName,
					PipelineTaskName: childTaskName,
					TypeMeta:         runtime.TypeMeta{Kind: "TaskRun"},
				}},
			},
		},
	}
	parentPR := &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{Name: "parent-run", Namespace: ns},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				ChildReferences: []v1.ChildStatusReference{{
					Name:             "parent-run-call-child",
					PipelineTaskName: parentTaskName,
					TypeMeta:         runtime.TypeMeta{Kind: "PipelineRun"},
				}},
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		PipelineRuns: []*v1.PipelineRun{parentPR, childPR},
		TaskRuns:     []*v1.TaskRun{childTR},
		Namespaces:   []*corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: ns}}},
	})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"taskrun", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(parentPR, "v1"),
		cb.UnstructuredPR(childPR, "v1"),
		cb.UnstructuredTR(childTR, "v1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	clients := &cli.Clients{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	if err := actions.InitializeAPIGroupRes(clients.Tekton.Discovery()); err != nil {
		t.Fatal(err)
	}
	result, err := GetTaskRunsWithStatus(parentPR, clients, ns)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 TaskRun, got %d", len(result))
	}
	trs, ok := result[childTRName]
	if !ok {
		t.Fatalf("expected TaskRun %q in result", childTRName)
	}
	expectedTaskName := parentTaskName + " > " + childTaskName
	if trs.PipelineTaskName != expectedTaskName {
		t.Errorf("expected PipelineTaskName %q, got %q", expectedTaskName, trs.PipelineTaskName)
	}
}
func TestGetTaskRunsWithStatus_DeepNesting(t *testing.T) {
	ns := "namespace"
	grandchildPR := &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{Name: "a-b-c", Namespace: ns},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				ChildReferences: []v1.ChildStatusReference{{
					Name:             "tr-c",
					PipelineTaskName: "c-task",
					TypeMeta:         runtime.TypeMeta{Kind: "TaskRun"},
				}},
			},
		},
	}
	trC := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{Name: "tr-c", Namespace: ns},
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{PodName: "pod-c",
				StartTime: &metav1.Time{Time: time.Now()}},
			Status: duckv1.Status{Conditions: duckv1.Conditions{{
				Status: corev1.ConditionTrue, Type: apis.ConditionSucceeded}}},
		},
	}
	childPR := &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{Name: "a-b", Namespace: ns},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				ChildReferences: []v1.ChildStatusReference{{
					Name:             "a-b-c",
					PipelineTaskName: "b-task",
					TypeMeta:         runtime.TypeMeta{Kind: "PipelineRun"},
				}},
			},
		},
	}
	parentPR := &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: ns},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				ChildReferences: []v1.ChildStatusReference{{
					Name:             "a-b",
					PipelineTaskName: "a-task",
					TypeMeta:         runtime.TypeMeta{Kind: "PipelineRun"},
				}},
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		PipelineRuns: []*v1.PipelineRun{parentPR, childPR, grandchildPR},
		TaskRuns:     []*v1.TaskRun{trC},
		Namespaces:   []*corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: ns}}},
	})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"taskrun", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(parentPR, "v1"),
		cb.UnstructuredPR(childPR, "v1"),
		cb.UnstructuredPR(grandchildPR, "v1"),
		cb.UnstructuredTR(trC, "v1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	clients := &cli.Clients{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	if err := actions.InitializeAPIGroupRes(clients.Tekton.Discovery()); err != nil {
		t.Fatal(err)
	}
	result, err := GetTaskRunsWithStatus(parentPR, clients, ns)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 TaskRun, got %d", len(result))
	}
	trs := result["tr-c"]
	expected := "a-task > b-task > c-task"
	if trs.PipelineTaskName != expected {
		t.Errorf("expected %q, got %q", expected, trs.PipelineTaskName)
	}
}
func TestGetTaskRunsWithStatus_Mixed(t *testing.T) {
	ns := "namespace"
	trDirect := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{Name: "tr-build", Namespace: ns},
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{PodName: "pod-build",
				StartTime: &metav1.Time{Time: time.Now()}},
			Status: duckv1.Status{Conditions: duckv1.Conditions{{
				Status: corev1.ConditionTrue, Type: apis.ConditionSucceeded}}},
		},
	}
	trChild := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{Name: "tr-child-greet", Namespace: ns},
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{PodName: "pod-greet",
				StartTime: &metav1.Time{Time: time.Now()}},
			Status: duckv1.Status{Conditions: duckv1.Conditions{{
				Status: corev1.ConditionTrue, Type: apis.ConditionSucceeded}}},
		},
	}
	childPR := &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{Name: "pr-child", Namespace: ns},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				ChildReferences: []v1.ChildStatusReference{{
					Name: "tr-child-greet", PipelineTaskName: "greet",
					TypeMeta: runtime.TypeMeta{Kind: "TaskRun"},
				}},
			},
		},
	}
	parentPR := &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{Name: "pr-parent", Namespace: ns},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				ChildReferences: []v1.ChildStatusReference{
					{
						Name: "tr-build", PipelineTaskName: "build",
						TypeMeta: runtime.TypeMeta{Kind: "TaskRun"},
					},
					{
						Name: "pr-child", PipelineTaskName: "deploy",
						TypeMeta: runtime.TypeMeta{Kind: "PipelineRun"},
					},
				},
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		PipelineRuns: []*v1.PipelineRun{parentPR, childPR},
		TaskRuns:     []*v1.TaskRun{trDirect, trChild},
		Namespaces:   []*corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: ns}}},
	})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"taskrun", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(parentPR, "v1"),
		cb.UnstructuredPR(childPR, "v1"),
		cb.UnstructuredTR(trDirect, "v1"),
		cb.UnstructuredTR(trChild, "v1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	clients := &cli.Clients{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	if err := actions.InitializeAPIGroupRes(clients.Tekton.Discovery()); err != nil {
		t.Fatal(err)
	}
	result, err := GetTaskRunsWithStatus(parentPR, clients, ns)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 TaskRuns, got %d", len(result))
	}
	// Direct TaskRun — no prefix
	trs, ok := result["tr-build"]
	if !ok {
		t.Fatal("expected tr-build in result")
	}
	if trs.PipelineTaskName != "build" {
		t.Errorf("expected 'build', got %q", trs.PipelineTaskName)
	}
	// Child PipelineRun TaskRun — prefixed
	trs, ok = result["tr-child-greet"]
	if !ok {
		t.Fatal("expected tr-child-greet in result")
	}
	if trs.PipelineTaskName != "deploy > greet" {
		t.Errorf("expected 'deploy > greet', got %q", trs.PipelineTaskName)
	}
}
func TestGetTaskRunsWithStatus_Empty(t *testing.T) {
	ns := "namespace"
	pr := &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{Name: "pr", Namespace: ns},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		PipelineRuns: []*v1.PipelineRun{pr},
		Namespaces:   []*corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: ns}}},
	})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(cb.UnstructuredPR(pr, "v1"))
	if err != nil {
		t.Fatal(err)
	}
	clients := &cli.Clients{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	if err := actions.InitializeAPIGroupRes(clients.Tekton.Discovery()); err != nil {
		t.Fatal(err)
	}
	result, err := GetTaskRunsWithStatus(pr, clients, ns)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}
func TestGetTaskRunsWithStatus_NilPR(t *testing.T) {
	result, err := GetTaskRunsWithStatus(nil, nil, "ns")
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestFindNewTaskruns_PinP_RetryCount(t *testing.T) {
	ns := "namespace"
	childTaskName := "greet"
	childTRName := "child-taskrun"
	childPRName := "child-pr"
	parentPRName := "parent-pr"
	parentTaskName := "call-child"

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{Name: childTRName, Namespace: ns},
			Spec:       v1.TaskRunSpec{TaskRef: &v1.TaskRef{Name: childTaskName}},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{{Status: corev1.ConditionTrue}},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: time.Now()},
					PodName:   childTRName + "-pod",
				},
			},
		},
	}

	childPR := &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{Name: childPRName, Namespace: ns},
		Spec: v1.PipelineRunSpec{
			PipelineSpec: &v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{Name: childTaskName, TaskRef: &v1.TaskRef{Name: childTaskName}, Retries: 2},
				},
			},
		},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				PipelineSpec: &v1.PipelineSpec{
					Tasks: []v1.PipelineTask{
						{Name: childTaskName, TaskRef: &v1.TaskRef{Name: childTaskName}, Retries: 2},
					},
				},
				ChildReferences: []v1.ChildStatusReference{
					{
						Name:             childTRName,
						PipelineTaskName: childTaskName,
						TypeMeta:         runtime.TypeMeta{APIVersion: "tekton.dev/v1", Kind: "TaskRun"},
					},
				},
			},
		},
	}

	parentPR := &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{Name: parentPRName, Namespace: ns},
		Spec: v1.PipelineRunSpec{
			PipelineSpec: &v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{Name: parentTaskName, TaskRef: &v1.TaskRef{Name: parentTaskName}},
				},
			},
		},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				ChildReferences: []v1.ChildStatusReference{
					{
						Name:             childPRName,
						PipelineTaskName: parentTaskName,
						TypeMeta:         runtime.TypeMeta{APIVersion: "tekton.dev/v1", Kind: "PipelineRun"},
					},
				},
			},
		},
	}

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(parentPR, "v1"),
		cb.UnstructuredPR(childPR, "v1"),
		cb.UnstructuredTR(trs[0], "v1"),
	)
	if err != nil {
		t.Fatal(err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces:   []*corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: ns}}},
		PipelineRuns: []*v1.PipelineRun{parentPR, childPR},
		TaskRuns:     trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"pipelinerun", "taskrun"})

	tc := &cli.Clients{
		Tekton:  cs.Pipeline,
		Kube:    cs.Kube,
		Dynamic: dynamic,
	}
	if err := actions.InitializeAPIGroupRes(tc.Tekton.Discovery()); err != nil {
		t.Fatal(err)
	}

	tracker := NewTracker(parentPRName, ns, tc)
	trStatuses, err := GetTaskRunsWithStatus(parentPR, tc, ns)
	if err != nil {
		t.Fatal(err)
	}

	runs := tracker.findNewTaskruns(parentPR, nil, trStatuses)
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d: %v", len(runs), runs)
	}

	expectedTask := parentTaskName + ChildTaskSeparator + childTaskName
	if runs[0].Task != expectedTask {
		t.Errorf("expected task %q, got %q", expectedTask, runs[0].Task)
	}
	if runs[0].Retries != 2 {
		t.Errorf("expected retries 2, got %d", runs[0].Retries)
	}
}

func TestGetTaskRunsWithStatus_ChildPR_NotFound(t *testing.T) {
	ns := "namespace"
	directTaskName := "build"
	directTRName := "build-taskrun"
	childPRTaskName := "call-child"
	childPRName := "missing-child"

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{Name: directTRName, Namespace: ns},
			Spec:       v1.TaskRunSpec{TaskRef: &v1.TaskRef{Name: directTaskName}},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{{Status: corev1.ConditionTrue}},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: time.Now()},
					PodName:   directTRName + "-pod",
				},
			},
		},
	}

	pr := &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{Name: "pr", Namespace: ns},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				ChildReferences: []v1.ChildStatusReference{
					{
						Name:             directTRName,
						PipelineTaskName: directTaskName,
						TypeMeta:         runtime.TypeMeta{APIVersion: "tekton.dev/v1", Kind: "TaskRun"},
					},
					{
						Name:             childPRName,
						PipelineTaskName: childPRTaskName,
						TypeMeta:         runtime.TypeMeta{APIVersion: "tekton.dev/v1", Kind: "PipelineRun"},
					},
				},
			},
		},
	}

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pr, "v1"),
		cb.UnstructuredTR(trs[0], "v1"),
	)
	if err != nil {
		t.Fatal(err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces:   []*corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: ns}}},
		PipelineRuns: []*v1.PipelineRun{pr},
		TaskRuns:     trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"pipelinerun", "taskrun"})

	tc := &cli.Clients{
		Tekton:  cs.Pipeline,
		Kube:    cs.Kube,
		Dynamic: dynamic,
	}
	if err := actions.InitializeAPIGroupRes(tc.Tekton.Discovery()); err != nil {
		t.Fatal(err)
	}

	result, err := GetTaskRunsWithStatus(pr, tc, ns)
	if err != nil {
		t.Fatalf("expected no error despite missing child PR, got: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 taskrun (direct only), got %d", len(result))
	}
	if result[directTRName].PipelineTaskName != directTaskName {
		t.Errorf("expected PipelineTaskName %q, got %q", directTaskName, result[directTRName].PipelineTaskName)
	}
}
