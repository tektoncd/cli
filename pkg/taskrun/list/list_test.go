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

package list

import (
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestPipelinesList_GetAllTaskRuns(t *testing.T) {
	clock := clockwork.NewFakeClock()
	trStarted := clock.Now().Add(10 * time.Second)
	runDuration1 := 1 * time.Minute
	runDuration2 := 1 * time.Minute

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("taskrun1", tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.TaskRunStartTime(trStarted),
				taskRunCompletionTime(trStarted.Add(runDuration1)),
			),
		),
		tb.TaskRun("taskrun2", tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.TaskRunStartTime(trStarted),
				taskRunCompletionTime(trStarted.Add(runDuration2)),
			),
		),
	}

	version := "v1alpha1"

	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
		cb.UnstructuredTR(trs[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	p2 := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	p2.SetNamespace("unknown")

	testParams := []struct {
		name        string
		params      *test.Params
		listOptions metav1.ListOptions
		want        []string
	}{
		{
			name:        "Specify related task",
			params:      p,
			listOptions: metav1.ListOptions{LabelSelector: "tekton.dev/task=task"},
			want: []string{
				"taskrun1 started -10 seconds ago",
				"taskrun2 started -10 seconds ago",
			},
		},
		{
			name:        "Not specify related task",
			params:      p,
			listOptions: metav1.ListOptions{},
			want: []string{
				"taskrun1 started -10 seconds ago",
				"taskrun2 started -10 seconds ago",
			},
		},
		{
			name:        "Specify unknown namespace",
			params:      p2,
			listOptions: metav1.ListOptions{},
			want:        []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllTaskRuns(tp.params, metav1.ListOptions{}, 5)
			if err != nil {
				t.Errorf("Unexpected error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestPipelinesList_GetAllTaskRuns_v1beta1(t *testing.T) {
	clock := clockwork.NewFakeClock()
	trStarted := clock.Now().Add(10 * time.Second)
	runDuration1 := 1 * time.Minute
	runDuration2 := 1 * time.Minute

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("taskrun1", tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.TaskRunStartTime(trStarted),
				taskRunCompletionTime(trStarted.Add(runDuration1)),
			),
		),
		tb.TaskRun("taskrun2", tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.TaskRunStartTime(trStarted),
				taskRunCompletionTime(trStarted.Add(runDuration2)),
			),
		),
	}

	version := "v1beta1"
	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
		cb.UnstructuredTR(trs[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	p2 := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	p2.SetNamespace("unknown")

	testParams := []struct {
		name        string
		params      *test.Params
		listOptions metav1.ListOptions
		want        []string
	}{
		{
			name:        "Specify related task",
			params:      p,
			listOptions: metav1.ListOptions{LabelSelector: "tekton.dev/task=task"},
			want: []string{
				"taskrun1 started -10 seconds ago",
				"taskrun2 started -10 seconds ago",
			},
		},
		{
			name:        "Not specify related task",
			params:      p,
			listOptions: metav1.ListOptions{},
			want: []string{
				"taskrun1 started -10 seconds ago",
				"taskrun2 started -10 seconds ago",
			},
		},
		{
			name:        "Specify unknown namespace",
			params:      p2,
			listOptions: metav1.ListOptions{},
			want:        []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllTaskRuns(tp.params, metav1.ListOptions{}, 5)
			if err != nil {
				t.Errorf("Unexpected error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func taskRunCompletionTime(ct time.Time) tb.TaskRunStatusOp {
	return func(s *v1alpha1.TaskRunStatus) {
		s.CompletionTime = &metav1.Time{Time: ct}
	}
}
