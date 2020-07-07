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

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stest "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestTaskDescribe_Invalid_Namespace(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client()
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"task"})

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
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"task"})
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

func TestTaskDescribe_OnlyName(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", tb.TaskNamespace("ns")),
		tb.Task("task", tb.TaskNamespace("ns-2")),
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

	version := "v1alpha1"
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

func TestTaskDescribe_OnlyNameDiffNameSpace(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", tb.TaskNamespace("ns")),
		tb.Task("task", tb.TaskNamespace("ns-2")),
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
	version := "v1alpha1"
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

func TestTaskDescribe_OnlyNameParams(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1",
			tb.TaskNamespace("ns"),
			tb.TaskSpec(
				tb.TaskParam("myarg", v1alpha1.ParamTypeString, tb.ParamSpecDescription("params without the default value")),
				tb.TaskParam("myprint", v1alpha1.ParamTypeString, tb.ParamSpecDescription("testing very long "+
					"description information in order to test FormatDesc funnction call longlonglonglonglonglonglonglong"+
					"longlonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglong"+
					"longlonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglong")),
				tb.TaskParam("myarray", v1alpha1.ParamTypeArray),
			),
		),
		tb.Task("task", tb.TaskNamespace("ns-2")),
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

	version := "v1alpha1"
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

func TestTaskDescribe_Full(t *testing.T) {
	clock := clockwork.NewFakeClock()
	tasks := []*v1alpha1.Task{
		tb.Task("task-1",
			tb.TaskNamespace("ns"),
			tb.TaskSpec(
				tb.TaskDescription("a test description"),
				tb.TaskResources(
					tb.TaskResourcesInput("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.TaskResourcesInput("my-image", v1alpha1.PipelineResourceTypeImage),
					tb.TaskResourcesInput("source-repo", v1alpha1.PipelineResourceTypeGit),
					tb.TaskResourcesOutput("code-image", v1alpha1.PipelineResourceTypeImage),
					tb.TaskResourcesOutput("artifact-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.TaskParam("myarg", v1alpha1.ParamTypeString),
				tb.TaskParam("myarray", v1alpha1.ParamTypeArray),
				tb.TaskParam("print", v1alpha1.ParamTypeString, tb.ParamSpecDescription("params with sting type"),
					tb.ParamSpecDefault("somethingdifferent")),
				tb.TaskParam("output", v1alpha1.ParamTypeArray, tb.ParamSpecDefault("booms", "booms", "booms")),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
			),
		),
	}
	taskRuns := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task-1"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task-1", tb.TaskRefKind(v1alpha1.NamespacedTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionFalse,
					Reason: v1beta1.TaskRunReasonFailed.String(),
				}),
				tb.TaskRunStartTime(clock.Now()),
				cb.TaskRunCompletionTime(clock.Now().Add(5*time.Minute)),
			),
		),
		tb.TaskRun("tr-2",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task-1"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task-1", tb.TaskRefKind(v1alpha1.NamespacedTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.TaskRunReasonSuccessful.String(),
				}),
				tb.TaskRunStartTime(clock.Now().Add(10*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(17*time.Minute)),
			),
		),
		tb.TaskRun("tr-3",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task-1"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task-1", tb.TaskRefKind(v1alpha1.ClusterTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.TaskRunReasonSuccessful.String(),
				}),
				tb.TaskRunStartTime(clock.Now().Add(10*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(17*time.Minute)),
			),
		),
	}
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredTR(taskRuns[0], version),
		cb.UnstructuredTR(taskRuns[1], version),
		cb.UnstructuredTR(taskRuns[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	task := Command(p)
	clock.Advance(20 * time.Minute)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskDescribe_TaskRunError(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", tb.TaskNamespace("ns")),
		tb.Task("task", tb.TaskNamespace("ns-2")),
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

	version := "v1alpha1"
	tdc := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{
			{Verb: "list", Resource: "taskruns",
				Action: func(action k8stest.Action) (bool, runtime.Object, error) {
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

func TestTaskDescribe_custom_output(t *testing.T) {
	name := "task"
	expected := "task.tekton.dev/" + name

	clock := clockwork.NewFakeClock()

	tasks := []*v1alpha1.Task{
		tb.Task(name, tb.TaskNamespace("ns")),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
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

func TestTaskV1beta1Describe_Full(t *testing.T) {
	clock := clockwork.NewFakeClock()
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-1",
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test description",
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1alpha1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1alpha1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1alpha1.ResourceDeclaration{
								Name: "my-image",
								Type: v1alpha1.PipelineResourceTypeImage,
							},
						},
						{
							ResourceDeclaration: v1alpha1.ResourceDeclaration{
								Name: "source-repo",
								Type: v1alpha1.PipelineResourceTypeGit,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1alpha1.ResourceDeclaration{
								Name: "code-image",
								Type: v1alpha1.PipelineResourceTypeImage,
							},
						},
						{
							ResourceDeclaration: v1alpha1.ResourceDeclaration{
								Name: "artifact-image",
								Type: v1alpha1.PipelineResourceTypeImage,
							},
						},
					},
				},
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
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "output",
						Type: v1beta1.ParamTypeArray,
						Default: &v1beta1.ArrayOrString{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
				Steps: []v1beta1.Step{
					{
						Container: corev1.Container{
							Name:  "hello",
							Image: "busybox",
						},
					},
					{
						Container: corev1.Container{
							Name:  "exit",
							Image: "busybox",
						},
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
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
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
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
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
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
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

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Tasks: tasks, Namespaces: namespaces, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	task := Command(p)
	clock.Advance(20 * time.Minute)
	out, err := test.ExecuteCommand(task, "desc", "task-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}
