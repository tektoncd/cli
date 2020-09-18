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
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTask_GetAllTaskNames(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()
	tdata := []*v1alpha1.Task{
		tb.Task("task", tb.TaskNamespace("ns"),
			// created  5 minutes back
			cb.TaskCreationTime(clock.Now().Add(-5*time.Minute)),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks: tdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredT(tdata[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	tdata2 := []*v1alpha1.Task{
		tb.Task("task", tb.TaskNamespace("ns"),
			// created  5 minutes back
			cb.TaskCreationTime(clock.Now().Add(-5*time.Minute)),
		),
		tb.Task("task2", tb.TaskNamespace("ns"),
			// created  5 minutes back
			cb.TaskCreationTime(clock.Now().Add(-5*time.Minute)),
		),
	}
	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks: tdata2,
	})
	cs2.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredT(tdata2[0], version),
		cb.UnstructuredT(tdata2[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}
	p3 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}
	p3.SetNamespace("unknown")

	testParams := []struct {
		name   string
		params *test.Params
		want   []string
	}{
		{
			name:   "Single Task",
			params: p,
			want:   []string{"task"},
		},
		{
			name:   "Multi Tasks",
			params: p2,
			want:   []string{"task", "task2"},
		},
		{
			name:   "Unknown namespace",
			params: p3,
			want:   []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllTaskNames(tp.params)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestTask_List(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()
	tdata := []*v1alpha1.Task{
		tb.Task("task", tb.TaskNamespace("ns"),
			// created  5 minutes back
			cb.TaskCreationTime(clock.Now().Add(-5*time.Minute)),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks: tdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredT(tdata[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	tdata2 := []*v1alpha1.Task{
		tb.Task("task", tb.TaskNamespace("ns"),
			// created  5 minutes back
			cb.TaskCreationTime(clock.Now().Add(-5*time.Minute)),
		),
		tb.Task("task2", tb.TaskNamespace("ns"),
			// created  5 minutes back
			cb.TaskCreationTime(clock.Now().Add(-5*time.Minute)),
		),
	}
	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks: tdata2,
	})
	cs2.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredT(tdata2[0], version),
		cb.UnstructuredT(tdata2[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}

	c1, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	c2, err := p2.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	testParams := []struct {
		name   string
		client *cli.Clients
		want   []string
	}{
		{
			name:   "Single Task",
			client: c1,
			want:   []string{"task"},
		},
		{
			name:   "Multi Tasks",
			client: c2,
			want:   []string{"task", "task2"},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := List(tp.client, metav1.ListOptions{}, "ns")
			if err != nil {
				t.Errorf("unexpected Error")
			}

			ctnames := []string{}
			for _, ct := range got.Items {
				ctnames = append(ctnames, ct.Name)
			}
			test.AssertOutput(t, tp.want, ctnames)
		})
	}
}

func TestTaskV1beta1_List(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	tdata := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Tasks: tdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1T(tdata[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	tdata2 := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task2",
				Namespace: "ns",
			},
		},
	}
	cs2, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Tasks: tdata2,
	})
	cs2.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1T(tdata2[0], version),
		cb.UnstructuredV1beta1T(tdata2[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}

	c1, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	c2, err := p2.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	testParams := []struct {
		name   string
		client *cli.Clients
		want   []string
	}{
		{
			name:   "Single Task",
			client: c1,
			want:   []string{"task"},
		},
		{
			name:   "Multi Tasks",
			client: c2,
			want:   []string{"task", "task2"},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := List(tp.client, metav1.ListOptions{}, "ns")
			if err != nil {
				t.Errorf("unexpected Error")
			}

			ctnames := []string{}
			for _, ct := range got.Items {
				ctnames = append(ctnames, ct.Name)
			}
			test.AssertOutput(t, tp.want, ctnames)
		})
	}
}

func TestTask_Get(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()
	tdata := []*v1alpha1.Task{
		tb.Task("task", tb.TaskNamespace("ns")), // created  5 minutes back

		tb.Task("task2", tb.TaskNamespace("ns"),
			// created  5 minutes back
			cb.TaskCreationTime(clock.Now().Add(-5*time.Minute)),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks: tdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredT(tdata[0], version),
		cb.UnstructuredT(tdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Get(c, "task", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "task", got.Name)
}

func TestTaskV1beta1_Get(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	tdata := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task2",
				Namespace: "ns",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Tasks: tdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1T(tdata[0], version),
		cb.UnstructuredV1beta1T(tdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Get(c, "task", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "task", got.Name)
}
