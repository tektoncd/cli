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

	"github.com/tektoncd/cli/pkg/actions"
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

func TestTaskRunGet_v1beta1(t *testing.T) {
	version := "v1beta1"
	clock := test.FakeClock()
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
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
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

	var tr *v1beta1.TaskRun
	if err = actions.GetV1(taskrunGroupResource, c, "taskrun1", "ns", metav1.GetOptions{}, &tr); err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "taskrun1", tr.Name)
}

func TestTaskRunGet(t *testing.T) {
	version := "v1"
	clock := test.FakeClock()
	tr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute

	trdata := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun1",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: "Task",
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
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: "Task",
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
					StartTime:      &metav1.Time{Time: tr1Started},
					CompletionTime: &metav1.Time{Time: tr1Started.Add(runDuration)},
				},
			},
		},
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

	var tr *v1beta1.TaskRun
	if err = actions.GetV1(taskrunGroupResource, c, "taskrun1", "ns", metav1.GetOptions{}, &tr); err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "taskrun1", tr.Name)
}
