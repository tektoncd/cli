// Copyright © 2021 The Tekton Authors.
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

package clustertask

import (
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreate_TaskNotExist(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "t-1",
			},
			Spec: v1beta1.TaskSpec{
				Description: "a task",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: tasks})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("default")
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	clustertask := Command(p)

	out, err := test.ExecuteCommand(clustertask, "create", "ct", "--from", "t-1")
	if err == nil {
		t.Errorf("Expected error got nil")
	}
	expected := "Command \"create\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nError: Task t-1 does not exist in namespace default\n"
	test.AssertOutput(t, expected, out)
}

func TestCreate(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "t-1",
				Namespace: "default",
			},
			Spec: v1beta1.TaskSpec{
				Description: "a task",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: tasks})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("default")
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "clustertask"})

	clustertask := Command(p)

	out, err := test.ExecuteCommand(clustertask, "create", "ct-1", "--from", "t-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := "Command \"create\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nClusterTask ct-1 created from Task t-1 present in namespace default\n"
	test.AssertOutput(t, expected, out)
}

func TestCreate_InNamespace(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "t-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Description: "a task",
			},
		},
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: tasks, Namespaces: namespaces})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "clustertask"})

	clustertask := Command(p)

	out, err := test.ExecuteCommand(clustertask, "create", "--from", "t-1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := "Command \"create\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nClusterTask t-1 created from Task t-1 present in namespace ns\n"
	test.AssertOutput(t, expected, out)
}

func TestCreate_WithoutFlag(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "t-1",
			},
			Spec: v1beta1.TaskSpec{
				Description: "a task",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: tasks})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "clustertask"})

	clustertask := Command(p)

	out, err := test.ExecuteCommand(clustertask, "create", "ct-1")
	if err == nil {
		t.Errorf("Expected error got nil")
	}
	expected := "Command \"create\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nError: --from flag not passed\n"
	test.AssertOutput(t, expected, out)
}

func TestCreate_taskToClusterTask(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
			},
		},
	}
	ctask := taskToClusterTask(tasks[0], "ctask-1")
	test.AssertOutput(t, "ClusterTask", ctask.Kind)
}
