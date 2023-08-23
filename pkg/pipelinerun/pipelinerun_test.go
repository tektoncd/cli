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
	"github.com/tektoncd/cli/pkg/actions"
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

func TestPipelineRunsList_v1beta1(t *testing.T) {
	version := "v1beta1"
	clock := test.FakeClock()
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
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
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
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
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
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
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
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
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
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
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
			name:        "Specify related pipeline",
			client:      c1,
			namespace:   p.Namespace(),
			time:        p.Time(),
			listOptions: metav1.ListOptions{LabelSelector: "tekton.dev/pipeline=pipeline"},
			want: []string{
				"pipelinerun1 started -10 seconds ago",
				"pipelinerun2 started -10 seconds ago",
			},
		},
		{
			name:        "Not specify related pipeline",
			client:      c1,
			namespace:   p.Namespace(),
			time:        p.Time(),
			listOptions: metav1.ListOptions{},
			want: []string{
				"pipelinerun started -10 seconds ago",
				"pipelinerun1 started -10 seconds ago",
				"pipelinerun2 started -10 seconds ago",
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
			got, err := GetAllPipelineRuns(pipelineRunGroupResource, tp.listOptions, tp.client, tp.namespace, 5, tp.time)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestPipelineRunsList(t *testing.T) {
	version := "v1"
	clock := test.FakeClock()
	pr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute
	prdata := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipelinerun",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "random"},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "random",
				},
			},
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
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
			},
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
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
			},
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
					StartTime:      &metav1.Time{Time: pr1Started},
					CompletionTime: &metav1.Time{Time: pr1Started.Add(runDuration)},
				},
			},
		},
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
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
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
			name:        "Specify related pipeline",
			client:      c1,
			namespace:   p.Namespace(),
			time:        p.Time(),
			listOptions: metav1.ListOptions{LabelSelector: "tekton.dev/pipeline=pipeline"},
			want: []string{
				"pipelinerun1 started -10 seconds ago",
				"pipelinerun2 started -10 seconds ago",
			},
		},
		{
			name:        "Not specify related pipeline",
			client:      c1,
			namespace:   p.Namespace(),
			time:        p.Time(),
			listOptions: metav1.ListOptions{},
			want: []string{
				"pipelinerun started -10 seconds ago",
				"pipelinerun1 started -10 seconds ago",
				"pipelinerun2 started -10 seconds ago",
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
			got, err := GetAllPipelineRuns(pipelineRunGroupResource, tp.listOptions, tp.client, tp.namespace, 5, tp.time)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestPipelineRunGet_v1beta1(t *testing.T) {
	version := "v1beta1"
	clock := test.FakeClock()
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
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
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
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
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
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		PipelineRuns: prdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prdata[0], version),
		cb.UnstructuredV1beta1PR(prdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	var pipeline *v1beta1.PipelineRun
	err = actions.GetV1(pipelineRunGroupResource, c, "pipelinerun1", "ns", metav1.GetOptions{}, &pipeline)
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "pipelinerun1", pipeline.Name)
}

func TestPipelineRunGet(t *testing.T) {
	version := "v1"
	clock := test.FakeClock()
	pr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute

	prdata := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipelinerun1",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
			},
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
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
			},
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
					StartTime:      &metav1.Time{Time: pr1Started},
					CompletionTime: &metav1.Time{Time: pr1Started.Add(runDuration)},
				},
			},
		},
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
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	var pipeline *v1.PipelineRun
	err = actions.GetV1(pipelineRunGroupResource, c, "pipelinerun1", "ns", metav1.GetOptions{}, &pipeline)
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "pipelinerun1", pipeline.Name)
}

func TestPipelineRunCreate(t *testing.T) {
	version := "v1beta1"
	prdata := v1beta1.PipelineRun{
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
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Create(c, &prdata, metav1.CreateOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "pipelinerun1", got.Name)
}
