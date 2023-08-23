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
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/cli"
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

func TestPipelinesList_GetAllTaskRuns_v1beta1(t *testing.T) {
	clock := test.FakeClock()
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
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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

	cs, _ := test.SeedV1beta1TestData(t, test.Data{TaskRuns: trs})
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"taskrun"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], "v1beta1"),
		cb.UnstructuredV1beta1TR(trs[1], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	p2 := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	p2.SetNamespace("unknown")

	c1, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	c2, err := p2.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	testParams := []struct {
		name        string
		namespace   string
		client      *cli.Clients
		time        clockwork.Clock
		listOptions metav1.ListOptions
		want        []string
	}{
		{
			name:        "Specify related task",
			client:      c1,
			namespace:   p.Namespace(),
			time:        p.Time(),
			listOptions: metav1.ListOptions{LabelSelector: "tekton.dev/task=task"},
			want: []string{
				"taskrun1 started -10 seconds ago",
				"taskrun2 started -10 seconds ago",
			},
		},
		{
			name:        "Not specify related task",
			client:      c1,
			namespace:   p.Namespace(),
			time:        p.Time(),
			listOptions: metav1.ListOptions{},
			want: []string{
				"taskrun1 started -10 seconds ago",
				"taskrun2 started -10 seconds ago",
			},
		},
		{
			name:        "Specify unknown namespace",
			client:      c2,
			namespace:   p2.Namespace(),
			time:        p2.Time(),
			listOptions: metav1.ListOptions{},
			want:        []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllTaskRuns(taskrunGroupResource, metav1.ListOptions{}, tp.client, tp.namespace, 5, tp.time)
			if err != nil {
				t.Errorf("Unexpected error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestPipelinesList_GetAllTaskRuns(t *testing.T) {
	clock := test.FakeClock()
	trStarted := clock.Now().Add(10 * time.Second)
	runDuration1 := 1 * time.Minute
	runDuration2 := 1 * time.Minute

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun1",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
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
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: trStarted},
					CompletionTime: &metav1.Time{Time: trStarted.Add(runDuration2)},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"taskrun"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(trs[0], "v1"),
		cb.UnstructuredTR(trs[1], "v1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	p2 := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	p2.SetNamespace("unknown")

	c1, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	c2, err := p2.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	testParams := []struct {
		name        string
		namespace   string
		client      *cli.Clients
		time        clockwork.Clock
		listOptions metav1.ListOptions
		want        []string
	}{
		{
			name:        "Specify related task",
			client:      c1,
			namespace:   p.Namespace(),
			time:        p.Time(),
			listOptions: metav1.ListOptions{LabelSelector: "tekton.dev/task=task"},
			want: []string{
				"taskrun1 started -10 seconds ago",
				"taskrun2 started -10 seconds ago",
			},
		},
		{
			name:        "Not specify related task",
			client:      c1,
			namespace:   p.Namespace(),
			time:        p.Time(),
			listOptions: metav1.ListOptions{},
			want: []string{
				"taskrun1 started -10 seconds ago",
				"taskrun2 started -10 seconds ago",
			},
		},
		{
			name:        "Specify unknown namespace",
			client:      c2,
			namespace:   p2.Namespace(),
			time:        p2.Time(),
			listOptions: metav1.ListOptions{},
			want:        []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllTaskRuns(taskrunGroupResource, metav1.ListOptions{}, tp.client, tp.namespace, 5, tp.time)
			if err != nil {
				t.Errorf("Unexpected error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}
