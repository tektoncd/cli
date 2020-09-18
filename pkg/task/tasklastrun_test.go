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

	"github.com/jonboulle/clockwork"
	tb "github.com/tektoncd/cli/internal/builder/v1alpha1"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
)

const (
	versionA1 = "v1alpha1"
)

func TestTaskrunLatest_two_run(t *testing.T) {
	clock := clockwork.NewFakeClock()

	var (
		taskCreated = clock.Now().Add(5 * time.Minute)

		firstRunCreated   = clock.Now().Add(10 * time.Minute)
		firstRunStarted   = firstRunCreated.Add(2 * time.Second)
		firstRunCompleted = firstRunStarted.Add(10 * time.Minute)

		secondRunCreated   = firstRunCreated.Add(1 * time.Minute)
		secondRunStarted   = secondRunCreated.Add(2 * time.Second)
		secondRunCompleted = secondRunStarted.Add(5 * time.Minute)
	)
	tasks := []*v1alpha1.Task{
		tb.Task("task", tb.TaskNamespace("ns"),
			cb.TaskCreationTime(taskCreated),
		),
	}
	taskruns := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1", tb.TaskRunNamespace("ns"),
			cb.TaskRunCreationTime(firstRunCreated),
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task", tb.TaskRefKind(v1alpha1.NamespacedTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.TaskRunReasonSuccessful.String(),
				}),
				tb.TaskRunStartTime(firstRunStarted),
				cb.TaskRunCompletionTime(firstRunCompleted),
			),
		),
		tb.TaskRun("tr-2", tb.TaskRunNamespace("ns"),
			cb.TaskRunCreationTime(secondRunCompleted),
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task", tb.TaskRefKind(v1alpha1.NamespacedTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.TaskRunReasonSuccessful.String(),
				}),
				tb.TaskRunStartTime(secondRunStarted),
				cb.TaskRunCompletionTime(secondRunCompleted),
			),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks:    tasks,
		TaskRuns: taskruns,
	})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], versionA1),
		cb.UnstructuredTR(taskruns[0], versionA1),
		cb.UnstructuredTR(taskruns[1], versionA1),
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

	clock := clockwork.NewFakeClock()
	tasks := []*v1alpha1.Task{
		tb.Task("task", tb.TaskNamespace("ns"),
			cb.TaskCreationTime(clock.Now().Add(5*time.Minute)),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks: tasks,
	})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], versionA1))
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
