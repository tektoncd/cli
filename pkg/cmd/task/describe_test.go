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
	"errors"
	"fmt"
	"strings"
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
	"k8s.io/apimachinery/pkg/runtime"
	k8stest "k8s.io/client-go/testing"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestTaskDescribe_Invalid_Namespace(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client()
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"task"})

	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "bar", "-n", "invalid")
	if err == nil {
		t.Errorf("Error expected here")
	}
	expected := "Error: failed to get Task bar: tasks.tekton.dev \"bar\" not found\n"
	test.AssertOutput(t, expected, out)
}

func TestTaskDescribe_Empty(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"task"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "bar", "-n", "ns")
	if err == nil {
		t.Errorf("Error expected here")
	}
	expected := "Error: failed to get Task bar: tasks.tekton.dev \"bar\" not found\n"
	test.AssertOutput(t, expected, out)
}

func TestTaskDescribe_OnlyName_v1beta1(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns-2",
			},
		},
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-2",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1T(tasks[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_OnlyName(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns-2",
			},
		},
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-2",
			},
		},
	}

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredT(tasks[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_OnlyNameDiffNameSpace_v1beta1(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns-2",
			},
		},
	}
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-2",
			},
		},
	}
	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1T(tasks[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task", "-n", "ns-2")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_OnlyNameDiffNameSpace(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns-2",
			},
		},
	}
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-2",
			},
		},
	}
	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredT(tasks[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task", "-n", "ns-2")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_OnlyNameParams_v1beta1(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns-2",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Params: []v1beta1.ParamSpec{
					{
						Name:        "myarg",
						Type:        v1beta1.ParamTypeString,
						Description: "params without the default value",
					},
					{
						Name: "myprint",
						Type: v1beta1.ParamTypeString,
						Description: "testing very long " +
							"description information in order to test FormatDesc funnction call longlonglonglonglonglonglonglong" +
							"longlonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglong" +
							"longlonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglong",
					},
					{
						Name: "myarray",
						Type: v1beta1.ParamTypeArray,
					},
				},
			},
		},
	}
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-2",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1T(tasks[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_OnlyNameParams(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns-2",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name:        "myarg",
						Type:        v1.ParamTypeString,
						Description: "params without the default value",
					},
					{
						Name: "myprint",
						Type: v1.ParamTypeString,
						Description: "testing very long " +
							"description information in order to test FormatDesc funnction call longlonglonglonglonglonglonglong" +
							"longlonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglong" +
							"longlonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglong",
					},
					{
						Name: "myarray",
						Type: v1.ParamTypeArray,
					},
				},
			},
		},
	}
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-2",
			},
		},
	}

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredT(tasks[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_TaskRunError_v1beta1(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns-2",
			},
		},
	}
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-2",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{
			{Verb: "list", Resource: "taskruns",
				Action: func(_ k8stest.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("fake list taskrun error")
				}}}}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1T(tasks[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")

	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err == nil {
		t.Errorf("Expected error got nil")
	}
	expected := "Error: failed to get TaskRuns for Task task-1: fake list taskrun error\n"
	test.AssertOutput(t, expected, out)
}

func TestTaskDescribe_TaskRunError(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns-2",
			},
		},
	}
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-2",
			},
		},
	}

	version := "v1"
	tdc := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{
			{Verb: "list", Resource: "taskruns",
				Action: func(_ k8stest.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("fake list taskrun error")
				}}}}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredT(tasks[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")

	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err == nil {
		t.Errorf("Expected error got nil")
	}
	expected := "Error: failed to get TaskRuns for Task task-1: fake list taskrun error\n"
	test.AssertOutput(t, expected, out)
}

func TestTaskDescribe_custom_output_v1beta1(t *testing.T) {
	name := "task"
	expected := "task.tekton.dev/" + name

	clock := test.FakeClock()

	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "ns",
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
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	task := Command(p)
	got, err := test.ExecuteCommand(task, "desc", "-o", "name", "-n", "ns", name)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	got = strings.TrimSpace(got)
	if got != expected {
		t.Errorf("Result should be '%s' != '%s'", got, expected)
	}
}

func TestTaskDescribe_custom_output(t *testing.T) {
	name := "task"
	expected := "task.tekton.dev/" + name

	clock := test.FakeClock()

	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "ns",
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	task := Command(p)
	got, err := test.ExecuteCommand(task, "desc", "-o", "name", "-n", "ns", name)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	got = strings.TrimSpace(got)
	if got != expected {
		t.Errorf("Result should be '%s' != '%s'", got, expected)
	}
}

func TestTaskDescribe_WithoutNameIfOnlyOneTaskPresent_v1beta1(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
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
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_WithoutNameIfOnlyOneTaskPresent(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_Full_v1beta1(t *testing.T) {
	clock := test.FakeClock()
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
				Labels:    map[string]string{"app.kubernetes.io/version": "0.1"},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test description",
				Params: []v1beta1.ParamSpec{
					{
						Name: "myarg",
						Type: v1beta1.ParamTypeString,
					},
					{
						Name: "myarray",
						Type: v1beta1.ParamTypeArray,
					},
					{
						Name: "print",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "output",
						Type: v1beta1.ParamTypeArray,
						Default: &v1beta1.ParamValue{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
				Steps: []v1beta1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
					{
						Name:  "exit",
						Image: "busybox",
					},
				},
			},
		},
	}

	taskRuns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-1",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.TaskRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-2",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
					Kind: v1beta1.NamespacedTaskKind,
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
					StartTime:      &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(17 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-3",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.TaskRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
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
		cb.UnstructuredV1beta1TR(taskRuns[0], version),
		cb.UnstructuredV1beta1TR(taskRuns[1], version),
		cb.UnstructuredV1beta1TR(taskRuns[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: tasks, Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	p.SetNamespace("ns")
	task := Command(p)
	clock.Advance(20 * time.Minute)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_Full(t *testing.T) {
	clock := test.FakeClock()
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
				Labels:    map[string]string{"app.kubernetes.io/version": "0.1"},
			},
			Spec: v1.TaskSpec{
				Description: "a test description",
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "myarray",
						Type: v1.ParamTypeArray,
					},
					{
						Name: "print",
						Type: v1.ParamTypeString,
						Default: &v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "output",
						Type: v1.ParamTypeArray,
						Default: &v1.ParamValue{
							Type:     v1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
				Steps: []v1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
					{
						Name:  "exit",
						Image: "busybox",
					},
				},
			},
		},
	}

	taskRuns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-1",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task-1",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.TaskRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-2",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task-1",
					Kind: v1.NamespacedTaskKind,
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
					StartTime:      &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(17 * time.Minute)},
				},
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
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredTR(taskRuns[0], version),
		cb.UnstructuredTR(taskRuns[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	p.SetNamespace("ns")
	task := Command(p)
	clock.Advance(20 * time.Minute)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_With_Results_v1beta1(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test description",
				Steps: []v1beta1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
				},
				Results: []v1beta1.TaskResult{
					{
						Name:        "result-1",
						Description: "This is a description for result 1",
					},
					{
						Name:        "result-2",
						Description: "This is a description for result 2",
					},
					{
						Name: "result-3",
					},
				},
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
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_With_Results(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
			},
			Spec: v1.TaskSpec{
				Description: "a test description",
				Steps: []v1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
				},
				Results: []v1.TaskResult{
					{
						Name:        "result-1",
						Description: "This is a description for result 1",
					},
					{
						Name:        "result-2",
						Description: "This is a description for result 2",
					},
					{
						Name: "result-3",
					},
				},
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_With_Workspaces_v1beta1(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test description",
				Steps: []v1beta1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
				},
				Workspaces: []v1beta1.WorkspaceDeclaration{
					{
						Name:        "test",
						Description: "test workspace",
						MountPath:   "/workspace/test/file",
						ReadOnly:    true,
					},
				},
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
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_With_Workspaces(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
			},
			Spec: v1.TaskSpec{
				Description: "a test description",
				Steps: []v1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
				},
				Workspaces: []v1.WorkspaceDeclaration{
					{
						Name:        "test",
						Description: "test workspace",
						MountPath:   "/workspace/test/file",
						ReadOnly:    true,
					},
				},
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_With_Empty_Labels_v1beta1(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test description",
				Steps: []v1beta1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
				},
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
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_With_Empty_Labels(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
			},
			Spec: v1.TaskSpec{
				Description: "a test description",
				Steps: []v1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
				},
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_With_Different_Labels_v1beta1(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
				Labels:    map[string]string{"tekton.dev/task": "v1"},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test description",
				Steps: []v1beta1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
				},
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
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_With_Different_Labels(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
				Labels:    map[string]string{"tekton.dev/task": "v1"},
			},
			Spec: v1.TaskSpec{
				Description: "a test description",
				Steps: []v1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
				},
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_with_annotations_v1beta1(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
				Annotations: map[string]string{
					corev1.LastAppliedConfigAnnotation: "LastAppliedConfig",
					"tekton.dev/tags":                  "testing",
				},
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
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_with_annotations(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
				Annotations: map[string]string{
					corev1.LastAppliedConfigAnnotation: "LastAppliedConfig",
					"tekton.dev/tags":                  "testing",
				},
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_With_OptionalWorkspaces_v1beta1(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test description",
				Steps: []v1beta1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
				},
				Workspaces: []v1beta1.WorkspaceDeclaration{
					{
						Name:        "test",
						Description: "test workspace",
						MountPath:   "/workspace/test/file",
						ReadOnly:    true,
					},
					{
						Name:        "test-optional",
						Description: "test optional workspace",
						MountPath:   "/workspace/test-optional/file",
						ReadOnly:    true,
						Optional:    true,
					},
				},
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
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_With_OptionalWorkspaces(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
			},
			Spec: v1.TaskSpec{
				Description: "a test description",
				Steps: []v1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
				},
				Workspaces: []v1.WorkspaceDeclaration{
					{
						Name:        "test",
						Description: "test workspace",
						MountPath:   "/workspace/test/file",
						ReadOnly:    true,
					},
					{
						Name:        "test-optional",
						Description: "test optional workspace",
						MountPath:   "/workspace/test-optional/file",
						ReadOnly:    true,
						Optional:    true,
					},
				},
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

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	p.SetNamespace("ns")
	task := Command(p)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}
