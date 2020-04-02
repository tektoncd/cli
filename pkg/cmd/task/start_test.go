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
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	fakepipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	util_runtime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stest "k8s.io/client-go/testing"
)

func newPipelineClient(objs ...runtime.Object) *fakepipelineclientset.Clientset {
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	localSchemeBuilder := runtime.SchemeBuilder{
		v1alpha1.AddToScheme,
	}
	v1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	util_runtime.Must(localSchemeBuilder.AddToScheme(scheme))

	o := k8stest.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objs {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	c := &fakepipelineclientset.Clientset{}
	c.AddReactor("*", "*", k8stest.ObjectReaction(o))
	c.AddWatchReactor("*", func(action k8stest.Action) (handled bool, ret watch.Interface, err error) {
		gvr := action.GetResource()
		ns := action.GetNamespace()
		watch, err := o.Watch(gvr, ns)
		if err != nil {
			return false, nil, err
		}
		return true, watch, nil
	})

	c.PrependReactor("create", "taskruns", func(action k8stest.Action) (bool, runtime.Object, error) {
		create := action.(k8stest.CreateActionImpl)
		obj := create.GetObject().(*v1alpha1.TaskRun)
		obj.Name = "random"
		rFunc := k8stest.ObjectReaction(o)
		_, o, err := rFunc(action)
		return true, o, err
	})

	return c
}

func Test_start_invalid_namespace(t *testing.T) {

	cs, _ := test.SeedTestData(t, pipelinetest.Data{})
	c := Command(&test.Params{Tekton: cs.Pipeline, Kube: cs.Kube})

	_, err := test.ExecuteCommand(c, "start", "task", "-n", "invalid")

	if err == nil {
		t.Error("Expected an error for invalid namespace")
	}

	test.AssertOutput(t, "namespaces \"invalid\" not found", err.Error())
}

func Test_start_has_no_task_arg(t *testing.T) {
	c := Command(&test.Params{})

	_, err := test.ExecuteCommand(c, "start", "-n", "ns")
	if err == nil {
		t.Error("Expecting an error but it's empty")
	}
	test.AssertOutput(t, "either a task name or a --filename parameter must be supplied", err.Error())
}

func Test_start_has_filename_arg_with_last(t *testing.T) {
	c := Command(&test.Params{})

	_, err := test.ExecuteCommand(c, "start", "-n", "ns", "--filename=./testdata/task.yaml", "--last")
	if err == nil {
		t.Error("Expecting an error but it's empty")
	}
	test.AssertOutput(t, "cannot use --last option with --filename option", err.Error())
}

func Test_start_has_task_filename(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	c := Command(&test.Params{Tekton: cs.Pipeline, Kube: cs.Kube})

	got, err := test.ExecuteCommand(c, "start", "-n", "ns", "--filename=./testdata/task.yaml")
	if err != nil {
		t.Errorf("Not expecting an error, but got %s", err.Error())
	}

	expected := "Taskrun started: \n\nIn order to track the taskrun progress run:\ntkn taskrun logs  -f -n ns\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_not_found(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsResource("my-image", v1alpha1.PipelineResourceTypeImage),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeString),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-2", "-n", "ns")
	expected := "Error: task name task-2 does not exist in namespace ns\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsResource("my-image", v1alpha1.PipelineResourceTypeImage),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeArray),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-i=my-repo=git",
		"-i=my-image=image",
		"-p=myarg=value1",
		"-p=print=boom,boom",
		"-l=key=value",
		"-o=code-image=output-image",
		"-w=name=pvc,claimName=pvc3",
		"-s=svc1",
		"-n=ns")

	expected := "Taskrun started: \n\nIn order to track the taskrun progress run:\ntkn taskrun logs  -f -n ns\n"
	test.AssertOutput(t, expected, got)

	tr, err := cs.Pipeline.TektonV1alpha1().TaskRuns("ns").List(v1.ListOptions{})
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	if tr.Items[0].ObjectMeta.GenerateName != "task-1-run-" {
		t.Errorf("Error taskrun generated is different %+v", tr)
	}

	for _, v := range tr.Items[0].Spec.Inputs.Resources {
		if v.Name == "my-repo" {
			test.AssertOutput(t, "git", v.ResourceRef.Name)
		}

		if v.Name == "my-image" {
			test.AssertOutput(t, "image", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, 2, len(tr.Items[0].Spec.Inputs.Params))

	for _, v := range tr.Items[0].Spec.Inputs.Params {
		if v.Name == "my-arg" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeString, StringVal: "value1"}, v.Value)
		}

		if v.Name == "print" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeArray, ArrayVal: []string{"boom", "boom"}}, v.Value)
		}
	}

	for _, v := range tr.Items[0].Spec.Outputs.Resources {
		if v.Name == "code-image" {
			test.AssertOutput(t, "output-image", v.ResourceRef.Name)
		}
	}

	if d := cmp.Equal(tr.Items[0].ObjectMeta.Labels, map[string]string{"key": "value"}); !d {
		t.Errorf("Error labels generated is different Labels Got: %+v", tr.Items[0].ObjectMeta.Labels)
	}

	test.AssertOutput(t, "pvc", tr.Items[0].Spec.Workspaces[0].Name)
	test.AssertOutput(t, "pvc3", tr.Items[0].Spec.Workspaces[0].PersistentVolumeClaim.ClaimName)

	test.AssertOutput(t, "svc1", tr.Items[0].Spec.ServiceAccountName)
}

func Test_start_task_last(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeArray),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
				tb.TaskWorkspace("test", "test workspace", "/workspace/test/file", true),
			),
		),
	}

	taskruns := []*v1alpha1.TaskRun{
		tb.TaskRun("taskrun-123", "ns",
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("task"),
				tb.TaskRunServiceAccountName("svc"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("myarg", "value")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("print", "booms", "booms", "booms")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("my-repo", tb.TaskResourceBindingRef("git"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("code-image", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunWorkspaceEmptyDir("test", ""),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	//Add namespaces to kube client
	seedData, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})

	objs := []runtime.Object{tasks[0], taskruns[0]}
	pClient := newPipelineClient(objs...)

	cs := pipelinetest.Clients{
		Pipeline: pClient,
		Kube:     seedData.Kube,
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"--last",
		"-n=ns")

	expected := "Taskrun started: random\n\nIn order to track the taskrun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	tr, err := cs.Pipeline.TektonV1alpha1().TaskRuns("ns").Get("random", v1.GetOptions{})
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	for _, v := range tr.Spec.Inputs.Resources {
		if v.Name == "my-repo" {
			test.AssertOutput(t, "git", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, 2, len(tr.Spec.Inputs.Params))

	for _, v := range tr.Spec.Inputs.Params {
		if v.Name == "my-arg" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeString, StringVal: "value"}, v.Value)
		}

		if v.Name == "print" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeArray, ArrayVal: []string{"booms", "booms", "booms"}}, v.Value)
		}
	}

	for _, v := range tr.Spec.Outputs.Resources {
		if v.Name == "code-image" {
			test.AssertOutput(t, "image", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, "svc", tr.Spec.ServiceAccountName)
	test.AssertOutput(t, "test", tr.Spec.Workspaces[0].Name)
	test.AssertOutput(t, "", tr.Spec.Workspaces[0].SubPath)
}

func Test_start_use_taskrun(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeArray),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
				tb.TaskWorkspace("test", "test workspace", "/workspace/test/file", true),
			),
		),
	}

	taskruns := []*v1alpha1.TaskRun{
		tb.TaskRun("happy", "ns",
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("task"),
			),
		),
		tb.TaskRun("camper", "ns",
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("task"),
				tb.TaskRunServiceAccountName("camper"),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	//Add namespaces to kube client
	seedData, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})

	objs := []runtime.Object{tasks[0], taskruns[0], taskruns[1]}
	pClient := newPipelineClient(objs...)

	cs := pipelinetest.Clients{
		Pipeline: pClient,
		Kube:     seedData.Kube,
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"--use-taskrun", "camper",
		"-n=ns")

	expected := "Taskrun started: random\n\nIn order to track the taskrun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	tr, err := cs.Pipeline.TektonV1alpha1().TaskRuns("ns").Get("random", v1.GetOptions{})
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "camper", tr.Spec.ServiceAccountName)
}

func Test_start_task_last_generate_name(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeArray),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
				tb.TaskWorkspace("test", "test workspace", "/workspace/test/file", true),
			),
		),
	}

	taskruns := []*v1alpha1.TaskRun{
		tb.TaskRun("taskrun-123", "ns",
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("task"),
				tb.TaskRunServiceAccountName("svc"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("myarg", "value")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("print", "booms", "booms", "booms")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("my-repo", tb.TaskResourceBindingRef("git"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("code-image", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunWorkspaceEmptyDir("test", ""),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	// Setting GenerateName for test
	taskruns[0].ObjectMeta.GenerateName = "test-generatename-task-run-"

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})

	objs := []runtime.Object{tasks[0], taskruns[0]}
	pClient := newPipelineClient(objs...)

	cs := pipelinetest.Clients{
		Pipeline: pClient,
		Kube:     seedData.Kube,
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"--last",
		"-n=ns")

	expected := "Taskrun started: random\n\nIn order to track the taskrun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	tr, err := cs.Pipeline.TektonV1alpha1().TaskRuns("ns").Get("random", v1.GetOptions{})
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "test-generatename-task-run-", tr.ObjectMeta.GenerateName)
}

func Test_start_task_last_with_prefix_name(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeArray),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
				tb.TaskWorkspace("test", "test workspace", "/workspace/test/file", true),
			),
		),
	}

	taskruns := []*v1alpha1.TaskRun{
		tb.TaskRun("taskrun-123", "ns",
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("task"),
				tb.TaskRunServiceAccountName("svc"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("myarg", "value")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("print", "booms", "booms", "booms")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("my-repo", tb.TaskResourceBindingRef("git"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("code-image", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunWorkspaceEmptyDir("test", ""),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})

	objs := []runtime.Object{tasks[0], taskruns[0]}
	pClient := newPipelineClient(objs...)

	cs := pipelinetest.Clients{
		Pipeline: pClient,
		Kube:     seedData.Kube,
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"--last",
		"-n=ns",
		"--prefix-name=mytrname",
	)

	expected := "Taskrun started: random\n\nIn order to track the taskrun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	tr, err := cs.Pipeline.TektonV1alpha1().TaskRuns("ns").Get("random", v1.GetOptions{})
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "mytrname-", tr.ObjectMeta.GenerateName)
}

func Test_start_task_with_prefix_name(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeArray),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
				tb.TaskWorkspace("test", "test workspace", "/workspace/test/file", true),
			),
		),
	}

	taskruns := []*v1alpha1.TaskRun{
		tb.TaskRun("taskrun-123", "ns",
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("task"),
				tb.TaskRunServiceAccountName("svc"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("myarg", "value")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("print", "booms", "booms", "booms")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("my-repo", tb.TaskResourceBindingRef("git"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("code-image", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunWorkspaceEmptyDir("test", ""),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})

	objs := []runtime.Object{tasks[0], taskruns[0]}
	pClient := newPipelineClient(objs...)

	cs := pipelinetest.Clients{
		Pipeline: pClient,
		Kube:     seedData.Kube,
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"-n=ns",
		"--prefix-name=mytrname",
	)

	expected := "Taskrun started: random\n\nIn order to track the taskrun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	tr, err := cs.Pipeline.TektonV1alpha1().TaskRuns("ns").Get("random", v1.GetOptions{})
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "mytrname-", tr.ObjectMeta.GenerateName)
}

func Test_start_task_last_with_inputs(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeArray),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
			),
		),
	}

	taskruns := []*v1alpha1.TaskRun{
		tb.TaskRun("taskrun-123", "ns",
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("task"),
				tb.TaskRunServiceAccountName("svc"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("myarg", "value")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("print", "booms", "booms", "booms")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("my-repo", tb.TaskResourceBindingRef("git"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("code-image", tb.TaskResourceBindingRef("image"))),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	//Add namespaces to kube client
	seedData, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})

	objs := []runtime.Object{tasks[0], taskruns[0]}
	pClient := newPipelineClient(objs...)

	cs := pipelinetest.Clients{
		Pipeline: pClient,
		Kube:     seedData.Kube,
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"-i=my-repo=git-new",
		"-p=myarg=value1",
		"-p=print=boom,boom",
		"-o=code-image=output-image",
		"-s=svc1",
		"-n=ns",
		"--last")

	expected := "Taskrun started: random\n\nIn order to track the taskrun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	tr, err := cs.Pipeline.TektonV1alpha1().TaskRuns("ns").Get("random", v1.GetOptions{})
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	for _, v := range tr.Spec.Inputs.Resources {
		if v.Name == "my-repo" {
			test.AssertOutput(t, "git-new", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, 2, len(tr.Spec.Inputs.Params))

	for _, v := range tr.Spec.Inputs.Params {
		if v.Name == "my-arg" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeString, StringVal: "value1"}, v.Value)
		}

		if v.Name == "print" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeArray, ArrayVal: []string{"boom", "boom"}}, v.Value)
		}
	}

	for _, v := range tr.Spec.Outputs.Resources {
		if v.Name == "code-image" {
			test.AssertOutput(t, "output-image", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, "svc1", tr.Spec.ServiceAccountName)
}

func Test_start_task_last_without_pipelinerun(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsResource("my-image", v1alpha1.PipelineResourceTypeImage),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeString),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1", "--last", "-n", "ns")
	expected := "Error: no taskruns related to task task-1 found in namespace ns\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_client_error(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsResource("my-image", v1alpha1.PipelineResourceTypeImage),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeString),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.PrependReactor("create", "*", func(_ k8stest.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("cluster not accessible")
	})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1", "-n", "ns")
	expected := "Error: cluster not accessible\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_invalid_input_res(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsResource("my-image", v1alpha1.PipelineResourceTypeImage),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeString),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-i=my-repo git-repo",
		"-n", "ns",
	)
	expected := "Error: invalid input format for resource parameter: my-repo git-repo\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_invalid_workspace(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-w=claimName=pvc3",
		"-n", "ns",
	)
	expected := "Error: Name not found for workspace\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_invalid_output_res(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsResource("my-image", v1alpha1.PipelineResourceTypeImage),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeString),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-o", "code-image image-final",
		"-n", "ns",
	)
	expected := "Error: invalid input format for resource parameter: code-image image-final\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_invalid_param(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsResource("my-image", v1alpha1.PipelineResourceTypeImage),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeString),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-p", "myarg boom",
		"-n", "ns",
	)
	expected := "Error: invalid input format for param parameter: myarg boom\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_invalid_label(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsResource("my-image", v1alpha1.PipelineResourceTypeImage),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeString),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-l", "myarg boom",
		"-n", "ns",
	)
	expected := "Error: invalid input format for label parameter: myarg boom\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_allkindparam(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsResource("my-image", v1alpha1.PipelineResourceTypeImage),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeArray),
					tb.InputsParamSpec("printafter", v1alpha1.ParamTypeArray),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-i=my-repo=git",
		"-i=my-image=image",
		"-p=myarg=value1",
		"-p=print=boom,boom",
		"-p=printafter=booms",
		"-l=key=value",
		"-o=code-image=output-image",
		"-s=svc1",
		"-n=ns")

	expected := "Taskrun started: \n\nIn order to track the taskrun progress run:\ntkn taskrun logs  -f -n ns\n"
	test.AssertOutput(t, expected, got)

	tr, err := cs.Pipeline.TektonV1alpha1().TaskRuns("ns").List(v1.ListOptions{})
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	if tr.Items[0].ObjectMeta.GenerateName != "task-1-run-" {
		t.Errorf("Error taskrun generated is different %+v", tr)
	}

	for _, v := range tr.Items[0].Spec.Inputs.Resources {
		if v.Name == "my-repo" {
			test.AssertOutput(t, "git", v.ResourceRef.Name)
		}

		if v.Name == "my-image" {
			test.AssertOutput(t, "image", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, 3, len(tr.Items[0].Spec.Inputs.Params))

	for _, v := range tr.Items[0].Spec.Inputs.Params {
		if v.Name == "my-arg" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeString, StringVal: "value1"}, v.Value)
		}

		if v.Name == "print" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeArray, ArrayVal: []string{"boom", "boom"}}, v.Value)
		}

		if v.Name == "printafter" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeArray, ArrayVal: []string{"booms"}}, v.Value)
		}
	}

	for _, v := range tr.Items[0].Spec.Outputs.Resources {
		if v.Name == "code-image" {
			test.AssertOutput(t, "output-image", v.ResourceRef.Name)
		}
	}

	if d := cmp.Equal(tr.Items[0].ObjectMeta.Labels, map[string]string{"key": "value"}); !d {
		t.Errorf("Error labels generated is different Labels Got: %+v", tr.Items[0].ObjectMeta.Labels)
	}

	test.AssertOutput(t, "svc1", tr.Items[0].Spec.ServiceAccountName)
}

func Test_start_task_wrong_param(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsResource("my-image", v1alpha1.PipelineResourceTypeImage),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeArray),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-i=my-repo=git",
		"-i=my-image=image",
		"-p=myar=value1",
		"-p=print=boom,boom",
		"-l=key=value",
		"-o=code-image=output-image",
		"-s=svc1",
		"-n=ns")

	expected := "Error: param 'myar' not present in spec\n"
	test.AssertOutput(t, expected, got)
}

func Test_mergeResource(t *testing.T) {
	res := []v1alpha1.TaskResourceBinding{{
		PipelineResourceBinding: v1alpha1.PipelineResourceBinding{
			Name: "source",
			ResourceRef: &v1alpha1.PipelineResourceRef{
				Name: "git",
			},
		},
	}}

	_, err := mergeRes(res, []string{"test"})
	if err == nil {
		t.Errorf("Expected error")
	}

	res, err = mergeRes(res, []string{})
	if err != nil {
		t.Errorf("Did not expect error")
	}
	test.AssertOutput(t, 1, len(res))

	res, err = mergeRes(res, []string{"image=test-1"})
	if err != nil {
		t.Errorf("Did not expect error")
	}
	test.AssertOutput(t, 2, len(res))

	res, err = mergeRes(res, []string{"image=test-new", "image-2=test-2"})
	if err != nil {
		t.Errorf("Did not expect error")
	}
	test.AssertOutput(t, 3, len(res))
}

func Test_parseRes(t *testing.T) {
	type args struct {
		res []string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]v1alpha1.TaskResourceBinding
		wantErr bool
	}{{
		name: "Test_parseRes No Err",
		args: args{
			res: []string{"source=git", "image=docker2"},
		},
		want: map[string]v1alpha1.TaskResourceBinding{"source": {
			PipelineResourceBinding: v1alpha1.PipelineResourceBinding{
				Name: "source",
				ResourceRef: &v1alpha1.PipelineResourceRef{
					Name: "git",
				},
			},
		}, "image": {
			PipelineResourceBinding: v1alpha1.PipelineResourceBinding{
				Name: "image",
				ResourceRef: &v1alpha1.PipelineResourceRef{
					Name: "docker2",
				},
			},
		}},
		wantErr: false,
	}, {
		name: "Test_parseRes Err",
		args: args{
			res: []string{"value1", "value2"},
		},
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRes(tt.args.res)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskStart_ExecuteCommand(t *testing.T) {
	tasks := []*v1alpha1.Task{
		tb.Task("task-1", "ns",
			tb.TaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox", tb.StepName("hello")),
				tb.Step("busybox", tb.StepName("exit")),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})

	testParams := []struct {
		name       string
		command    []string
		namespace  string
		input      pipelinetest.Clients
		wantError  bool
		want       string
		goldenFile bool
	}{
		{
			name: "Dry Run with invalid output",
			command: []string{"start", "task-1",
				"-i=my-repo=git-repo",
				"-o=code-image=output-image",
				"-s=svc1",
				"-n", "ns",
				"--dry-run",
				"--output", "invalid"},
			namespace: "",
			input:     cs,
			wantError: true,
			want:      "output format specified is invalid but must be yaml or json",
		},
		{
			name: "Dry Run with only --dry-run specified",
			command: []string{"start", "task-1",
				"-i=my-repo=git-repo",
				"-o=code-image=output-image",
				"-s=svc1",
				"-n", "ns",
				"--dry-run"},
			namespace:  "",
			input:      cs,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with output=json",
			command: []string{"start", "task-1",
				"-i=my-repo=git-repo",
				"-o=code-image=output-image",
				"-s=svc1",
				"-n", "ns",
				"--dry-run",
				"--output=json"},
			namespace:  "",
			input:      cs,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with -f",
			command: []string{"start",
				"-f", "./testdata/task.yaml",
				"-n", "ns",
				"-s=svc1",
				"-i=docker-source=git",
				"-o=builtImage=image",
				"--dry-run",
				"--output=yaml"},
			namespace:  "",
			input:      cs,
			wantError:  false,
			goldenFile: true,
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube}
			c := Command(p)

			got, err := test.ExecuteCommand(c, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
				if tp.goldenFile {
					golden.Assert(t, got, strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
				} else {
					test.AssertOutput(t, tp.want, got)
				}
			}
		})
	}
}
