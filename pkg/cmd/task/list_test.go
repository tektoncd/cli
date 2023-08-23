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

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTaskList_Invalid_Namespace_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client()

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"task"})

	task := Command(p)
	output, err := test.ExecuteCommand(task, "list", "-n", "foo")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "No Tasks found\n", output)
}

func TestTaskList_Empty_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"task"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	task := Command(p)
	output, err := test.ExecuteCommand(task, "list", "-n", "foo")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, "No Tasks found\n", output)
}

func TestTaskList_Only_Tasks_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bananas",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "apples",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "potatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-514 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "onionss",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-515 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test task to test description of task",
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1T(tasks[1], version),
		cb.UnstructuredV1beta1T(tasks[2], version),
		cb.UnstructuredV1beta1T(tasks[3], version),
		cb.UnstructuredV1beta1T(tasks[4], version),
		cb.UnstructuredV1beta1T(tasks[5], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: tasks, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	task := Command(p)

	output, err := test.ExecuteCommand(task, "list", "-n", "namespace")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskList_Only_Tasks_no_headers_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bananas",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "apples",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "potatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-514 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "onions",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-515 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test task to test description of task",
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1T(tasks[1], version),
		cb.UnstructuredV1beta1T(tasks[2], version),
		cb.UnstructuredV1beta1T(tasks[3], version),
		cb.UnstructuredV1beta1T(tasks[4], version),
		cb.UnstructuredV1beta1T(tasks[5], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: tasks, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	task := Command(p)

	output, err := test.ExecuteCommand(task, "list", "-n", "namespace", "--no-headers")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskList_Only_Tasks_all_namespaces_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bananas",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "apples",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "potatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-514 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "onions",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-515 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test task to test description of task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomates",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangues",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bananes",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pommes",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "patates",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-514 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "oignons",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-515 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test task to test description of task",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1T(tasks[1], version),
		cb.UnstructuredV1beta1T(tasks[2], version),
		cb.UnstructuredV1beta1T(tasks[3], version),
		cb.UnstructuredV1beta1T(tasks[4], version),
		cb.UnstructuredV1beta1T(tasks[5], version),
		cb.UnstructuredV1beta1T(tasks[6], version),
		cb.UnstructuredV1beta1T(tasks[7], version),
		cb.UnstructuredV1beta1T(tasks[8], version),
		cb.UnstructuredV1beta1T(tasks[9], version),
		cb.UnstructuredV1beta1T(tasks[10], version),
		cb.UnstructuredV1beta1T(tasks[11], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: tasks})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	task := Command(p)

	output, err := test.ExecuteCommand(task, "list", "--all-namespaces")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskList_Only_Tasks_all_namespaces_no_headers_v1beta1(t *testing.T) {
	clock := test.FakeClock()
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bananas",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "apples",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "potatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-514 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "onions",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-515 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test task to test description of task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomates",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangues",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bananes",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pommes",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "patates",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-514 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "oignons",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-515 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test task to test description of task",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1T(tasks[1], version),
		cb.UnstructuredV1beta1T(tasks[2], version),
		cb.UnstructuredV1beta1T(tasks[3], version),
		cb.UnstructuredV1beta1T(tasks[4], version),
		cb.UnstructuredV1beta1T(tasks[5], version),
		cb.UnstructuredV1beta1T(tasks[6], version),
		cb.UnstructuredV1beta1T(tasks[7], version),
		cb.UnstructuredV1beta1T(tasks[8], version),
		cb.UnstructuredV1beta1T(tasks[9], version),
		cb.UnstructuredV1beta1T(tasks[10], version),
		cb.UnstructuredV1beta1T(tasks[11], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: tasks})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	task := Command(p)

	output, err := test.ExecuteCommand(task, "list", "--all-namespaces", "--no-headers")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskList_Invalid_Namespace(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client()

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"task"})

	task := Command(p)
	output, err := test.ExecuteCommand(task, "list", "-n", "foo")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "No Tasks found\n", output)
}

func TestTaskList_Empty(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"task"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	task := Command(p)
	output, err := test.ExecuteCommand(task, "list", "-n", "foo")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, "No Tasks found\n", output)
}

func TestTaskList_Only_Tasks(t *testing.T) {
	clock := test.FakeClock()

	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bananas",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "apples",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "potatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-514 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "a test task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "onionss",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-515 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "a test task to test description of task",
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredT(tasks[1], version),
		cb.UnstructuredT(tasks[2], version),
		cb.UnstructuredT(tasks[3], version),
		cb.UnstructuredT(tasks[4], version),
		cb.UnstructuredT(tasks[5], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	task := Command(p)

	output, err := test.ExecuteCommand(task, "list", "-n", "namespace")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskList_Only_Tasks_no_headers(t *testing.T) {
	clock := test.FakeClock()

	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bananas",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "apples",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "potatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-514 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "a test task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "onions",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-515 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "a test task to test description of task",
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredT(tasks[1], version),
		cb.UnstructuredT(tasks[2], version),
		cb.UnstructuredT(tasks[3], version),
		cb.UnstructuredT(tasks[4], version),
		cb.UnstructuredT(tasks[5], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	task := Command(p)

	output, err := test.ExecuteCommand(task, "list", "-n", "namespace", "--no-headers")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskList_Only_Tasks_all_namespaces(t *testing.T) {
	clock := test.FakeClock()

	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bananas",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "apples",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "potatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-514 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "a test task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "onions",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-515 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "a test task to test description of task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomates",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangues",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bananes",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pommes",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "patates",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-514 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "a test task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "oignons",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-515 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "a test task to test description of task",
			},
		},
	}

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredT(tasks[1], version),
		cb.UnstructuredT(tasks[2], version),
		cb.UnstructuredT(tasks[3], version),
		cb.UnstructuredT(tasks[4], version),
		cb.UnstructuredT(tasks[5], version),
		cb.UnstructuredT(tasks[6], version),
		cb.UnstructuredT(tasks[7], version),
		cb.UnstructuredT(tasks[8], version),
		cb.UnstructuredT(tasks[9], version),
		cb.UnstructuredT(tasks[10], version),
		cb.UnstructuredT(tasks[11], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	task := Command(p)

	output, err := test.ExecuteCommand(task, "list", "--all-namespaces")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskList_Only_Tasks_all_namespaces_no_headers(t *testing.T) {
	clock := test.FakeClock()
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bananas",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "apples",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "potatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-514 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "a test task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "onions",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-515 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "a test task to test description of task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomates",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangues",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bananes",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pommes",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "patates",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-514 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "a test task",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "oignons",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-515 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "a test task to test description of task",
			},
		},
	}

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredT(tasks[1], version),
		cb.UnstructuredT(tasks[2], version),
		cb.UnstructuredT(tasks[3], version),
		cb.UnstructuredT(tasks[4], version),
		cb.UnstructuredT(tasks[5], version),
		cb.UnstructuredT(tasks[6], version),
		cb.UnstructuredT(tasks[7], version),
		cb.UnstructuredT(tasks[8], version),
		cb.UnstructuredT(tasks[9], version),
		cb.UnstructuredT(tasks[10], version),
		cb.UnstructuredT(tasks[11], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	task := Command(p)

	output, err := test.ExecuteCommand(task, "list", "--all-namespaces", "--no-headers")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskList_in_all_namespaces_with_output_yaml_flag(t *testing.T) {
	clock := test.FakeClock()

	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bananas",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomates",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangues",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bananes",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pommes",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "oignons",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-515 * time.Hour)},
			},
			Spec: v1.TaskSpec{
				Description: "a test task to test description of task",
			},
		},
	}

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredT(tasks[1], version),
		cb.UnstructuredT(tasks[2], version),
		cb.UnstructuredT(tasks[3], version),
		cb.UnstructuredT(tasks[4], version),
		cb.UnstructuredT(tasks[5], version),
		cb.UnstructuredT(tasks[6], version),
		cb.UnstructuredT(tasks[7], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	task := Command(p)

	output, err := test.ExecuteCommand(task, "list", "--all-namespaces", "--output=yaml")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}
