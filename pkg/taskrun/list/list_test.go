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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestPipelinesList_GetAllTaskRuns(t *testing.T) {
	clock := clockwork.NewFakeClock()
	trStarted := clock.Now().Add(10 * time.Second)
	runDuration1 := 1 * time.Minute
	runDuration2 := 1 * time.Minute

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun1",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
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
					StartTime:      &metav1.Time{Time: trStarted},
					CompletionTime: &metav1.Time{Time: trStarted.Add(runDuration1)},
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
					StartTime:      &metav1.Time{Time: trStarted},
					CompletionTime: &metav1.Time{Time: trStarted.Add(runDuration2)},
				},
			},
		},
	}

	version := "v1beta1"

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{TaskRuns: trs})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], version),
		cb.UnstructuredV1beta1TR(trs[1], version),
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
