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
	"fmt"
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTask_GetAllTaskNames_v1beta1(t *testing.T) {
	version := "v1beta1"
	clock := test.FakeClock()
	tdata := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
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
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task2",
				Namespace: "ns",
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}
	cs2, _ := test.SeedV1beta1TestData(t, test.Data{
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
	p3 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}
	p3.SetNamespace("unknown")

	c1, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	c2, err := p2.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	c3, err := p3.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	testParams := []struct {
		name      string
		namespace string
		client    *cli.Clients
		want      []string
	}{
		{
			name:      "Single Task",
			client:    c1,
			namespace: p.Namespace(),
			want:      []string{"task"},
		},
		{
			name:      "Multi Tasks",
			client:    c2,
			namespace: p2.Namespace(),
			want:      []string{"task", "task2"},
		},
		{
			name:      "Unknown namespace",
			client:    c3,
			namespace: p3.Namespace(),
			want:      []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllTaskNames(taskGroupResource, tp.client, tp.namespace)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestTask_GetAllTaskNames(t *testing.T) {
	version := "v1"
	clock := test.FakeClock()
	tdata := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
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

	tdata2 := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task2",
				Namespace: "ns",
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
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

	c1, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	c2, err := p2.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	c3, err := p3.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	testParams := []struct {
		name      string
		namespace string
		client    *cli.Clients
		want      []string
	}{
		{
			name:      "Single Task",
			client:    c1,
			namespace: p.Namespace(),
			want:      []string{"task"},
		},
		{
			name:      "Multi Tasks",
			client:    c2,
			namespace: p2.Namespace(),
			want:      []string{"task", "task2"},
		},
		{
			name:      "Unknown namespace",
			client:    c3,
			namespace: p3.Namespace(),
			want:      []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllTaskNames(taskGroupResource, tp.client, tp.namespace)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestTask_List_v1beta1(t *testing.T) {
	version := "v1beta1"
	clock := test.FakeClock()
	tdata := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
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
	cs2, _ := test.SeedV1beta1TestData(t, test.Data{
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
			var tasks *v1beta1.TaskList
			if err := actions.ListV1(taskGroupResource, tp.client, metav1.ListOptions{}, "ns", &tasks); err != nil {
				t.Errorf("unexpected Error")
			}

			tnames := []string{}
			for _, t := range tasks.Items {
				tnames = append(tnames, t.Name)
			}
			test.AssertOutput(t, tp.want, tnames)
		})
	}
}

func TestTask_List(t *testing.T) {
	version := "v1"
	clock := test.FakeClock()
	tdata := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
		},
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

	tdata2 := []*v1.Task{
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
			var tasks *v1.TaskList
			if err := actions.ListV1(taskGroupResource, tp.client, metav1.ListOptions{}, "ns", &tasks); err != nil {
				t.Errorf("unexpected Error")
			}

			tnames := []string{}
			fmt.Println(tnames)
			for _, t := range tasks.Items {
				tnames = append(tnames, t.Name)
			}
			test.AssertOutput(t, tp.want, tnames)
		})
	}
}

func TestTask_Get_v1beta1(t *testing.T) {
	version := "v1beta1"
	clock := test.FakeClock()
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
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
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

	var task *v1beta1.Pipeline
	err = actions.GetV1(taskGroupResource, c, "task", "ns", metav1.GetOptions{}, &task)
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "task", task.Name)
}

func TestTask_Get(t *testing.T) {
	version := "v1"
	clock := test.FakeClock()
	tdata := []*v1.Task{
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

	var task *v1.Task
	err = actions.GetV1(taskGroupResource, c, "task", "ns", metav1.GetOptions{}, &task)
	if err != nil {
		t.Errorf("unexpected Error")
	}

	test.AssertOutput(t, "task", task.Name)
}

func TestTask_Create(t *testing.T) {
	version := "v1beta1"
	clock := test.FakeClock()
	tdata := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Create(c, tdata[0], metav1.CreateOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "task", got.Name)
}
