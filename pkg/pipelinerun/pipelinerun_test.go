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
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestPipelinesList_with_single_run(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()
	pr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute

	prdata := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipelinerun", "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", "random"),
			tb.PipelineRunStatus(),
		),
		tb.PipelineRun("pipelinerun1", "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.PipelineRunStartTime(pr1Started),
				cb.PipelineRunCompletionTime(pr1Started.Add(runDuration)),
			),
		),
		tb.PipelineRun("pipelinerun2", "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.PipelineRunStartTime(pr1Started),
				cb.PipelineRunCompletionTime(pr1Started.Add(runDuration)),
			),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		PipelineRuns: prdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prdata[0], version),
		cb.UnstructuredPR(prdata[1], version),
		cb.UnstructuredPR(prdata[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic clinet: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2.SetNamespace("unknown")

	testParams := []struct {
		name        string
		params      *test.Params
		listOptions metav1.ListOptions
		want        []string
	}{
		{
			name:        "Specify related pipeline",
			params:      p,
			listOptions: metav1.ListOptions{LabelSelector: "tekton.dev/pipeline=pipeline"},
			want: []string{
				"pipelinerun1 started -10 seconds ago",
				"pipelinerun2 started -10 seconds ago",
			},
		},
		{
			name:        "Not specify related pipeline",
			params:      p,
			listOptions: metav1.ListOptions{},
			want: []string{
				"pipelinerun started ---",
				"pipelinerun1 started -10 seconds ago",
				"pipelinerun2 started -10 seconds ago",
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
			got, err := GetAllPipelineRuns(tp.params, tp.listOptions, 5)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestPipelinesV1beta1List_with_single_run(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	pr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute
	prdata := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipelinerun",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "random"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "random",
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: pr1Started},
					CompletionTime: &metav1.Time{Time: pr1Started.Add(runDuration)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipelinerun1",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: pr1Started},
					CompletionTime: &metav1.Time{Time: pr1Started.Add(runDuration)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipelinerun2",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: pr1Started},
					CompletionTime: &metav1.Time{Time: pr1Started.Add(runDuration)},
				},
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		PipelineRuns: prdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prdata[0], version),
		cb.UnstructuredV1beta1PR(prdata[1], version),
		cb.UnstructuredV1beta1PR(prdata[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic clinet: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2.SetNamespace("unknown")

	testParams := []struct {
		name        string
		params      *test.Params
		listOptions metav1.ListOptions
		want        []string
	}{
		{
			name:        "Specify related pipeline",
			params:      p,
			listOptions: metav1.ListOptions{LabelSelector: "tekton.dev/pipeline=pipeline"},
			want: []string{
				"pipelinerun1 started -10 seconds ago",
				"pipelinerun2 started -10 seconds ago",
			},
		},
		{
			name:        "Not specify related pipeline",
			params:      p,
			listOptions: metav1.ListOptions{},
			want: []string{
				"pipelinerun started -10 seconds ago",
				"pipelinerun1 started -10 seconds ago",
				"pipelinerun2 started -10 seconds ago",
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
			got, err := GetAllPipelineRuns(tp.params, tp.listOptions, 5)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestPipelineRunGet(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()
	pr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute

	prdata := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipelinerun1", "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.PipelineRunStartTime(pr1Started),
				cb.PipelineRunCompletionTime(pr1Started.Add(runDuration)),
			),
		),
		tb.PipelineRun("pipelinerun2", "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.PipelineRunStartTime(pr1Started),
				cb.PipelineRunCompletionTime(pr1Started.Add(runDuration)),
			),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		PipelineRuns: prdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prdata[0], version),
		cb.UnstructuredPR(prdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic clinet: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Get(c, "pipelinerun1", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "pipelinerun1", got.Name)
}

func TestPipelineGet_v1beta1(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	pr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute

	prdata := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipelinerun1",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: pr1Started},
					CompletionTime: &metav1.Time{Time: pr1Started.Add(runDuration)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipelinerun2",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: pr1Started},
					CompletionTime: &metav1.Time{Time: pr1Started.Add(runDuration)},
				},
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		PipelineRuns: prdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prdata[0], version),
		cb.UnstructuredV1beta1PR(prdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic clinet: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Get(c, "pipelinerun1", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "pipelinerun1", got.Name)
}
