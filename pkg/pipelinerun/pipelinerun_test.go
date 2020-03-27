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
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinetest "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestPipelinesList_with_single_run(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()
	pr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute

	prdata := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline", "ns",
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
	cs.Pipeline.Resources = cb.APIResourceList(version, "pipelinerun")

	dc, err := testDynamic.Client(
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
				"pipeline started ---",
				"pipelinerun1 started -10 seconds ago",
				"pipelinerun2 started -10 seconds ago",
			},
		},
		{
			name:        "Not specify related pipeline",
			params:      p,
			listOptions: metav1.ListOptions{},
			want: []string{
				"pipeline started ---",
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
			got, err := GetAllPipelineRuns(tp.params, metav1.ListOptions{}, 5)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestPipelinesList_with_single_run_v1beta1(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	pr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute

	prdata := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline", "ns",
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
	cs.Pipeline.Resources = cb.APIResourceList(version, "pipelinerun")

	dc, err := testDynamic.Client(
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
				"pipeline started ---",
				"pipelinerun1 started -10 seconds ago",
				"pipelinerun2 started -10 seconds ago",
			},
		},
		{
			name:        "Not specify related pipeline",
			params:      p,
			listOptions: metav1.ListOptions{},
			want: []string{
				"pipeline started ---",
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
			got, err := GetAllPipelineRuns(tp.params, metav1.ListOptions{}, 5)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}
