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

package taskrun

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
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestTaskRunGet(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()
	tr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute

	trdata := []*v1alpha1.TaskRun{
		tb.TaskRun("taskrun1", tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task", tb.TaskRefKind("Task"))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.TaskRunReasonSuccessful.String(),
				}),
				tb.TaskRunStartTime(tr1Started),
				cb.TaskRunCompletionTime(tr1Started.Add(runDuration)),
			),
		),
		tb.TaskRun("taskrun2", tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task", tb.TaskRefKind("Task"))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.TaskRunReasonSuccessful.String(),
				}),
				tb.TaskRunStartTime(tr1Started),
				cb.TaskRunCompletionTime(tr1Started.Add(runDuration)),
			),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		TaskRuns: trdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trdata[0], version),
		cb.UnstructuredTR(trdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Get(c, "taskrun1", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "taskrun1", got.Name)
}

func TestTaskGet_v1beta1(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	tr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute

	trdata := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun1",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: "Task",
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
					StartTime:      &metav1.Time{Time: tr1Started},
					CompletionTime: &metav1.Time{Time: tr1Started.Add(runDuration)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun2",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: "Task",
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
					StartTime:      &metav1.Time{Time: tr1Started},
					CompletionTime: &metav1.Time{Time: tr1Started.Add(runDuration)},
				},
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		TaskRuns: trdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trdata[0], version),
		cb.UnstructuredV1beta1TR(trdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Get(c, "taskrun1", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "taskrun1", got.Name)
}
