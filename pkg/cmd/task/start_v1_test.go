// Copyright © 2023 The Tekton Authors.
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
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	fakepipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	k8stest "k8s.io/client-go/testing"
)

func newPipelineClient(objs ...runtime.Object) (*fakepipelineclientset.Clientset, testDynamic.Options) {
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	localSchemeBuilder := runtime.SchemeBuilder{v1.AddToScheme}

	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	util.Must(localSchemeBuilder.AddToScheme(scheme))
	o := k8stest.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objs {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	dc := testDynamic.Options{
		AddReactorRes:  "*",
		AddReactorVerb: "*",
		AddReactorFun:  k8stest.ObjectReaction(o),
		WatchResource:  "*",
		WatchReactionFun: func(action k8stest.Action) (handled bool, ret watch.Interface, err error) {
			gvr := action.GetResource()
			ns := action.GetNamespace()
			watcher, err := o.Watch(gvr, ns)
			if err != nil {
				return false, nil, err
			}
			return true, watcher, nil
		},
		PrependReactors: []testDynamic.PrependOpt{
			{
				Resource: "taskruns",
				Verb:     "create",
				Action: func(action k8stest.Action) (bool, runtime.Object, error) {
					create := action.(k8stest.CreateActionImpl)
					unstructuredTR := create.GetObject().(*unstructured.Unstructured)
					unstructuredTR.SetName("random")
					rFunc := k8stest.ObjectReaction(o)
					_, o, err := rFunc(action)
					return true, o, err
				},
			},
			{
				Resource: "taskruns",
				Verb:     "get",
				Action: func(action k8stest.Action) (bool, runtime.Object, error) {
					getAction, _ := action.(k8stest.GetActionImpl)
					res := getAction.GetResource()
					ns := getAction.GetNamespace()
					name := getAction.GetName()
					obj, err := o.Get(res, ns, name)
					if err != nil {
						return false, nil, err
					}
					if reflect.TypeOf(obj).String() == "*unstructured.Unstructured" {
						return true, obj, nil
					}

					v1TR := obj.(*v1.TaskRun)
					unstructuredTR := cb.UnstructuredTR(v1TR, version)
					return true, unstructuredTR, nil
				},
			},
		},
	}
	return nil, dc
}

func Test_start_invalid_namespace(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{})
	c := Command(&test.Params{Tekton: cs.Pipeline, Kube: cs.Kube})

	out, err := test.ExecuteCommand(c, "start", "task", "-n", "invalid")
	if err == nil {
		t.Error("Expected an error for invalid namespace")
	}

	test.AssertOutput(t, "Error: Task name task does not exist in namespace invalid\n", out)
}

func Test_start_has_no_task_arg(t *testing.T) {
	c := Command(&test.Params{})

	out, err := test.ExecuteCommand(c, "start", "-n", "ns")
	if err == nil {
		t.Error("Expecting an error but it's empty")
	}
	test.AssertOutput(t, "Error: either a Task name or a --filename argument must be supplied\n", out)
}

func Test_start_has_filename_arg_with_last(t *testing.T) {
	c := Command(&test.Params{})

	_, err := test.ExecuteCommand(c, "start", "-n", "ns", "--filename=./testdata/task-v1.yaml", "--last")
	if err == nil {
		t.Error("Expecting an error but it's empty")
	}
	test.AssertOutput(t, "cannot use --last option with --filename or --image option", err.Error())
}

func Test_start_has_image_arg_no_taskname(t *testing.T) {
	c := Command(&test.Params{})

	_, err := test.ExecuteCommand(c, "start", "-n", "ns", "--image=oci.rad.com/task:0.1", "--last")
	if err == nil {
		t.Error("Expecting an error but it's empty")
	}
	test.AssertOutput(t, "task name required with --image option", err.Error())
}

func Test_start_has_image_arg_with_last(t *testing.T) {
	c := Command(&test.Params{})

	_, err := test.ExecuteCommand(c, "start", "rad-task", "-n", "ns", "--image=oci.rad.com/task:0.1", "--last")
	if err == nil {
		t.Error("Expecting an error but it's empty")
	}
	test.AssertOutput(t, "cannot use --last option with --filename or --image option", err.Error())
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
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c := Command(&test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc})

	got, err := test.ExecuteCommand(c, "start", "-n", "ns", "--filename=./testdata/task-v1.yaml",
		"-w=name=temporary,emptyDir=", "-p=foobar=foo")
	if err != nil {
		t.Errorf("Not expecting an error, but got %s", err.Error())
	}

	expected := "TaskRun started: \n\nIn order to track the TaskRun progress run:\ntkn taskrun logs  -f -n ns\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_with_filename_invalid(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c := Command(&test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc})

	_, err = test.ExecuteCommand(c, "start", "-n", "ns", "--filename=./testdata/task-v1-invalid.yaml")
	if err == nil {
		t.Errorf("expected an error but didn't get one")
	}

	expected := `error unmarshaling JSON: while decoding JSON: json: unknown field "param"`
	test.AssertOutput(t, expected, err.Error())
}

func Test_start_task_not_found(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeString,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version))
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-2", "-n", "ns")
	expected := "Error: Task name task-2 does not exist in namespace ns\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_context(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version))
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)

	gotConfig, _ := test.ExecuteCommand(task, "start", "task-1",
		"--context=NinjaRabbit",
		"-p=myarg=value1",
		"-p=print=boom,boom",
		"-l=key=value",
		"-s=svc1",
		"-n=ns")

	gcExpected := "TaskRun started: \n\nIn order to track the TaskRun progress run:\ntkn taskrun --context=NinjaRabbit logs  -f -n ns\n"
	test.AssertOutput(t, gcExpected, gotConfig)

}

func Test_start_task(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version))
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-p=myarg=value1",
		"-p=print=boom,boom",
		"-l=key=value",
		"-s=svc1",
		"-n=ns")

	expected := "TaskRun started: \n\nIn order to track the TaskRun progress run:\ntkn taskrun logs  -f -n ns\n"
	test.AssertOutput(t, expected, got)
	clients, _ := p.Clients()

	var tr *v1.TaskRunList
	if err := actions.ListV1(taskrunGroupResource, clients, metav1.ListOptions{}, "ns", &tr); err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	if tr.Items[0].ObjectMeta.GenerateName != "task-1-run-" {
		t.Errorf("Error taskrun generated is different %+v", tr)
	}

	test.AssertOutput(t, 2, len(tr.Items[0].Spec.Params))

	for _, v := range tr.Items[0].Spec.Params {
		if v.Name == "my-arg" {
			test.AssertOutput(t, v1.ParamValue{Type: v1.ParamTypeString, StringVal: "value1"}, v.Value)
		}

		if v.Name == "print" {
			test.AssertOutput(t, v1.ParamValue{Type: v1.ParamTypeArray, ArrayVal: []string{"boom", "boom"}}, v.Value)
		}
	}

	if d := cmp.Equal(tr.Items[0].ObjectMeta.Labels, map[string]string{"key": "value"}); !d {
		t.Errorf("Error labels generated is different Labels Got: %+v", tr.Items[0].ObjectMeta.Labels)
	}

	test.AssertOutput(t, "svc1", tr.Items[0].Spec.ServiceAccountName)
}

func Test_start_task_last(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
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

	timeoutDuration, _ := time.ParseDuration("10s")
	taskruns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1.TaskRunSpec{
				Params: []v1.Param{
					{
						Name:  "myarg",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "value"},
					},
					{
						Name:  "print",
						Value: v1.ParamValue{Type: v1.ParamTypeArray, ArrayVal: []string{"booms", "booms", "booms"}},
					},
				},
				ServiceAccountName: "svc",
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
				},
				Timeout: &metav1.Duration{Duration: timeoutDuration},
				Workspaces: []v1.WorkspaceBinding{
					{
						Name:     "test",
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})
	objs := []runtime.Object{tasks[0], taskruns[0]}
	_, tdc := newPipelineClient(objs...)
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredTR(taskruns[0], version),
	)

	cs := test.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	clients, _ := p.Clients()
	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"--last",
		"-n=ns")

	expected := "TaskRun started: random\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)
	var tr *v1.TaskRun
	if err := actions.GetV1(taskrunGroupResource, clients, "random", "ns", metav1.GetOptions{}, &tr); err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, 2, len(tr.Spec.Params))

	for _, v := range tr.Spec.Params {
		if v.Name == "my-arg" {
			test.AssertOutput(t, v1.ParamValue{Type: v1.ParamTypeString, StringVal: "value"}, v.Value)
		}

		if v.Name == "print" {
			test.AssertOutput(t, v1.ParamValue{Type: v1.ParamTypeArray, ArrayVal: []string{"booms", "booms", "booms"}}, v.Value)
		}
	}

	test.AssertOutput(t, "svc", tr.Spec.ServiceAccountName)
	test.AssertOutput(t, "test", tr.Spec.Workspaces[0].Name)
	test.AssertOutput(t, "", tr.Spec.Workspaces[0].SubPath)
	test.AssertOutput(t, timeoutDuration, tr.Spec.Timeout.Duration)
}

func Test_start_task_last_with_override_timeout(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
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

	// Add timeout to last TaskRun for Task
	timeoutDuration, _ := time.ParseDuration("10s")
	taskruns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1.TaskRunSpec{
				Timeout: &metav1.Duration{Duration: timeoutDuration},
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})
	objs := []runtime.Object{tasks[0], taskruns[0]}
	_, tdc := newPipelineClient(objs...)
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredTR(taskruns[0], version),
	)

	cs := test.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	clients, _ := p.Clients()
	task := Command(p)
	// Specify new timeout value to override previous value
	got, err := test.ExecuteCommand(task, "start", "task", "--last", "--timeout", "1s", "-n=ns")
	if err != nil {
		t.Errorf("Error running task start: %v", err)
	}

	expected := "TaskRun started: random\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)
	var gotTR *v1.TaskRun
	if err := actions.GetV1(taskrunGroupResource, clients, "random", "ns", metav1.GetOptions{}, &gotTR); err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	// Assert newly started TaskRun has new timeout value
	timeoutDuration, _ = time.ParseDuration("1s")
	test.AssertOutput(t, timeoutDuration, gotTR.Spec.Timeout.Duration)
}

func Test_start_use_taskrun(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
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

	timeoutDuration, _ := time.ParseDuration("10s")

	taskruns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "happy",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
				},
			},
		},
		{

			ObjectMeta: metav1.ObjectMeta{
				Name:      "camper",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
				},
				ServiceAccountName: "camper",
				Timeout:            &metav1.Duration{Duration: timeoutDuration},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})

	objs := []runtime.Object{tasks[0], taskruns[0], taskruns[1]}
	_, tdc := newPipelineClient(objs...)

	cs := test.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredTR(taskruns[0], version),
		cb.UnstructuredTR(taskruns[1], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"--use-taskrun", "camper",
		"-n=ns")

	expected := "TaskRun started: random\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)
	clients, _ := p.Clients()
	var tr *v1.TaskRun
	if err := actions.GetV1(taskrunGroupResource, clients, "random", "ns", metav1.GetOptions{}, &tr); err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "camper", tr.Spec.ServiceAccountName)
	test.AssertOutput(t, timeoutDuration, tr.Spec.Timeout.Duration)
}

func Test_start_use_taskrun_cancelled_status(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
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

	timeoutDuration, _ := time.ParseDuration("10s")

	taskruns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "camper",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
				},
				ServiceAccountName: "camper",
				Timeout:            &metav1.Duration{Duration: timeoutDuration},
				Status:             v1.TaskRunSpecStatus(v1.TaskRunSpecStatusCancelled),
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})

	objs := []runtime.Object{tasks[0], taskruns[0]}
	_, tdc := newPipelineClient(objs...)

	cs := test.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredTR(taskruns[0], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"--use-taskrun", "camper",
		"-n=ns")

	expected := "TaskRun started: random\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)
	clients, _ := p.Clients()
	var tr *v1.TaskRun
	if err := actions.GetV1(taskrunGroupResource, clients, "random", "ns", metav1.GetOptions{}, &tr); err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "camper", tr.Spec.ServiceAccountName)
	test.AssertOutput(t, timeoutDuration, tr.Spec.Timeout.Duration)
	// Assert that new TaskRun does not contain cancelled status of previous run
	test.AssertOutput(t, v1.TaskRunSpecStatus(""), tr.Spec.Status)
}

func Test_start_task_last_generate_name(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
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

	timeoutDuration, _ := time.ParseDuration("10s")

	taskruns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1.TaskRunSpec{
				Params: []v1.Param{
					{
						Name:  "myarg",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "value"},
					},
					{
						Name:  "print",
						Value: v1.ParamValue{Type: v1.ParamTypeArray, ArrayVal: []string{"booms", "booms", "booms"}},
					},
				},
				ServiceAccountName: "svc",
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
				},
				Timeout: &metav1.Duration{Duration: timeoutDuration},
				Workspaces: []v1.WorkspaceBinding{
					{
						Name:     "test",
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
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

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})

	objs := []runtime.Object{tasks[0], taskruns[0]}
	_, tdc := newPipelineClient(objs...)

	cs := test.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredTR(taskruns[0], version))
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"--last",
		"-n=ns")

	expected := "TaskRun started: random\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)
	clients, _ := p.Clients()
	var tr *v1.TaskRun
	if err := actions.GetV1(taskrunGroupResource, clients, "random", "ns", metav1.GetOptions{}, &tr); err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "test-generatename-task-run-", tr.ObjectMeta.GenerateName)
}

func Test_start_task_last_with_prefix_name(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
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

	timeoutDuration, _ := time.ParseDuration("10s")

	taskruns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1.TaskRunSpec{
				Params: []v1.Param{
					{
						Name:  "myarg",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "value"},
					},
					{
						Name:  "print",
						Value: v1.ParamValue{Type: v1.ParamTypeArray, ArrayVal: []string{"booms", "booms", "booms"}},
					},
				},
				ServiceAccountName: "svc",
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
				},
				Timeout: &metav1.Duration{Duration: timeoutDuration},
				Workspaces: []v1.WorkspaceBinding{
					{
						Name:     "test",
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})

	objs := []runtime.Object{tasks[0], taskruns[0]}
	_, tdc := newPipelineClient(objs...)

	cs := test.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredTR(taskruns[0], version))
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"--last",
		"-n=ns",
		"--prefix-name=mytrname",
	)

	expected := "TaskRun started: random\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)
	clients, _ := p.Clients()
	var tr *v1.TaskRun
	if err := actions.GetV1(taskrunGroupResource, clients, "random", "ns", metav1.GetOptions{}, &tr); err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "mytrname-", tr.ObjectMeta.GenerateName)
}

func Test_start_task_with_prefix_name(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
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

	timeoutDuration, _ := time.ParseDuration("10s")

	taskruns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1.TaskRunSpec{
				Params: []v1.Param{
					{
						Name:  "myarg",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "value"},
					},
					{
						Name:  "print",
						Value: v1.ParamValue{Type: v1.ParamTypeArray, ArrayVal: []string{"booms", "booms", "booms"}},
					},
				},
				ServiceAccountName: "svc",
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
				},
				Timeout: &metav1.Duration{Duration: timeoutDuration},
				Workspaces: []v1.WorkspaceBinding{
					{
						Name:     "test",
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})
	objs := []runtime.Object{tasks[0], taskruns[0]}
	_, tdc := newPipelineClient(objs...)
	cs := test.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredTR(taskruns[0], version))
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"-n=ns",
		"--prefix-name=mytrname",
		"--last",
	)

	expected := "TaskRun started: random\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	clients, _ := p.Clients()
	var tr *v1.TaskRun
	if err := actions.GetV1(taskrunGroupResource, clients, "random", "ns", metav1.GetOptions{}, &tr); err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "mytrname-", tr.ObjectMeta.GenerateName)
}

func Test_start_task_last_with_inputs(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
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

	timeoutDuration, _ := time.ParseDuration("10s")

	taskruns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1.TaskRunSpec{
				Params: []v1.Param{
					{
						Name:  "myarg",
						Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "value"},
					},
					{
						Name:  "print",
						Value: v1.ParamValue{Type: v1.ParamTypeArray, ArrayVal: []string{"booms", "booms", "booms"}},
					},
				},
				ServiceAccountName: "svc",
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
				},
				Timeout: &metav1.Duration{Duration: timeoutDuration},
				Workspaces: []v1.WorkspaceBinding{
					{
						Name:     "test",
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})
	objs := []runtime.Object{tasks[0], taskruns[0]}
	_, tdc := newPipelineClient(objs...)
	cs := test.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredTR(taskruns[0], version))
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"-p=myarg=value1",
		"-p=print=boom,boom",
		"-s=svc1",
		"-n=ns",
		"--last")

	expected := "TaskRun started: random\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	clients, _ := p.Clients()
	var tr *v1.TaskRun
	if err := actions.GetV1(taskrunGroupResource, clients, "random", "ns", metav1.GetOptions{}, &tr); err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, 2, len(tr.Spec.Params))

	for _, v := range tr.Spec.Params {
		if v.Name == "my-arg" {
			test.AssertOutput(t, v1.ParamValue{Type: v1.ParamTypeString, StringVal: "value1"}, v.Value)
		}

		if v.Name == "print" {
			test.AssertOutput(t, v1.ParamValue{Type: v1.ParamTypeArray, ArrayVal: []string{"boom", "boom"}}, v.Value)
		}
	}

	test.AssertOutput(t, "svc1", tr.Spec.ServiceAccountName)
}

func Test_start_task_last_without_taskrun(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	objs := []runtime.Object{tasks[0]}
	_, tdc := newPipelineClient(objs...)
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1", "--last", "-n", "ns")
	expected := "Error: no TaskRuns related to Task task-1 found in namespace ns\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_client_error(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{
			{
				Resource: "*",
				Verb:     "create",
				Action: func(_ k8stest.Action) (bool, runtime.Object, error) {
					return true, nil, fmt.Errorf("cluster not accessible")
				},
			},
		},
	}
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task,
		"start", "task-1", "-n", "ns",
		"-p=myarg=value1",
		"-p=print=boom,boom",
	)
	expected := "Error: cluster not accessible\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_invalid_workspace(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
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
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-w=claimName=pvc3",
		"-n", "ns",
	)
	expected := "Error: Name not found for workspace\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_invalid_param(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-p", "myarg boom",
		"-n", "ns",
	)
	expected := "Error: invalid input format for param parameter: myarg boom\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_invalid_label(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-l", "myarg boom",
		"-p=param=param",
		"-n", "ns",
		"-p=myarg=abc",
		"-p=print=xyz",
	)
	expected := "Error: invalid input format for label parameter: myarg boom\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_allkindparam(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
					},
					{
						Name: "printafter",
						Type: v1.ParamTypeArray,
					},
					{
						Name: "printlast",
						Type: v1.ParamTypeObject,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-p=myarg=value1",
		"-p=print=boom,boom",
		"-p=printafter=booms",
		"-p=printlast=a:b, c:d",
		"-l=key=value",
		"-s=svc1",
		"-n=ns")

	expected := "TaskRun started: \n\nIn order to track the TaskRun progress run:\ntkn taskrun logs  -f -n ns\n"
	test.AssertOutput(t, expected, got)
	clients, _ := p.Clients()

	var tr *v1.TaskRunList
	if err := actions.ListV1(taskrunGroupResource, clients, metav1.ListOptions{}, "ns", &tr); err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	if tr.Items[0].ObjectMeta.GenerateName != "task-1-run-" {
		t.Errorf("Error taskrun generated is different %+v", tr)
	}

	test.AssertOutput(t, 4, len(tr.Items[0].Spec.Params))

	for _, v := range tr.Items[0].Spec.Params {
		if v.Name == "my-arg" {
			test.AssertOutput(t, v1.ParamValue{Type: v1.ParamTypeString, StringVal: "value1"}, v.Value)
		}

		if v.Name == "print" {
			test.AssertOutput(t, v1.ParamValue{Type: v1.ParamTypeArray, ArrayVal: []string{"boom", "boom"}}, v.Value)
		}

		if v.Name == "printafter" {
			test.AssertOutput(t, v1.ParamValue{Type: v1.ParamTypeArray, ArrayVal: []string{"booms"}}, v.Value)
		}

		if v.Name == "printlast" {
			test.AssertOutput(t, v1.ParamValue{Type: v1.ParamTypeObject, ObjectVal: map[string]string{"a": "b", "c": "d"}}, v.Value)
		}
	}

	if d := cmp.Equal(tr.Items[0].ObjectMeta.Labels, map[string]string{"key": "value"}); !d {
		t.Errorf("Error labels generated is different Labels Got: %+v", tr.Items[0].ObjectMeta.Labels)
	}

	test.AssertOutput(t, "svc1", tr.Items[0].Spec.ServiceAccountName)
}

func Test_start_task_wrong_param(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
					},
					{
						Name: "print",
						Type: v1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-p=myar=value1",
		"-p=myarg=value1",
		"-p=print=boom,boom",
		"-l=key=value",
		"-s=svc1",
		"-n=ns")

	expected := "Error: param 'myar' not present in spec\n"
	test.AssertOutput(t, expected, got)
}

func TestTaskStart_ExecuteCommand(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
				Params: []v1.ParamSpec{
					{
						Name: "myarg",
						Type: v1.ParamTypeString,
						Default: &v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "arg1",
						},
					},
					{
						Name: "task-param",
						Type: v1.ParamTypeString,
						Default: &v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "my-param",
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskruns"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
	)

	testParams := []struct {
		name       string
		command    []string
		namespace  string
		dynamic    dynamic.Interface
		input      pipelinetest.Clients
		wantError  bool
		hasPrefix  bool
		want       string
		goldenFile bool
	}{
		{
			name: "Dry Run with invalid output",
			command: []string{"start", "task-1",
				"-p=myarg=arg",
				"-p=task-param=arg",
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
				"-p=myarg=arg",
				"-p=task-param=arg",
				"-s=svc1",
				"-n", "ns",
				"--dry-run"},
			namespace:  "",
			dynamic:    dc,
			input:      cs,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with --use-param-defaults and specified params",
			command: []string{"start", "task-1",
				"-p=myarg=arg",
				"-s=svc1",
				"-n", "ns",
				"--dry-run",
				"--use-param-defaults"},
			namespace:  "",
			dynamic:    dc,
			input:      cs,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with --use-param-defaults and no specified params",
			command: []string{"start", "task-1",
				"-s=svc1",
				"-n", "ns",
				"--dry-run",
				"--use-param-defaults"},
			namespace:  "",
			dynamic:    dc,
			input:      cs,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with --use-param-defaults, --last and --use-taskrun",
			command: []string{"start", "task-1",
				"-s=svc1",
				"-n", "ns",
				"--dry-run",
				"--use-param-defaults",
				"--last",
				"--use-taskrun", "dummy-taskrun"},
			namespace: "",
			dynamic:   dc,
			input:     cs,
			wantError: true,
			want:      "cannot use --last or --use-taskrun options with --use-param-defaults option",
		},
		{
			name: "Dry Run with --use-param-defaults and --use-taskrun",
			command: []string{"start", "task-1",
				"-s=svc1",
				"-n", "ns",
				"--dry-run",
				"--use-param-defaults",
				"--use-taskrun", "dummy-taskrun"},
			namespace: "",
			dynamic:   dc,
			input:     cs,
			wantError: true,
			want:      "cannot use --last or --use-taskrun options with --use-param-defaults option",
		},
		{
			name: "Dry Run with --use-param-defaults and --last",
			command: []string{"start", "task-1",
				"-s=svc1",
				"-n", "ns",
				"--dry-run",
				"--use-param-defaults",
				"--last"},
			namespace: "",
			dynamic:   dc,
			input:     cs,
			wantError: true,
			want:      "cannot use --last or --use-taskrun options with --use-param-defaults option",
		},
		{
			name: "Dry Run with output=json",
			command: []string{"start", "task-1",
				"-p=myarg=arg",
				"-p=task-param=arg",
				"-s=svc1",
				"-n", "ns",
				"--dry-run",
				"--output=json"},
			namespace:  "",
			dynamic:    dc,
			input:      cs,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with -f",
			command: []string{"start",
				"-f", "./testdata/task-v1.yaml",
				"-n", "ns",
				"-s=svc1",
				"-p=foobar=context",
				"-w=name=temporary,emptyDir=",
				"--dry-run",
				"--output=yaml"},
			namespace:  "",
			dynamic:    dc,
			input:      cs,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with --timeout specified",
			command: []string{"start", "task-1",
				"-p=myarg=arg",
				"-p=task-param=arg",
				"-s=svc1",
				"-n", "ns",
				"--dry-run",
				"--timeout", "1s"},
			namespace:  "",
			dynamic:    dc,
			input:      cs,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with invalid --timeout specified",
			command: []string{"start", "task-1",
				"-p=myarg=arg",
				"-p=task-param=arg",
				"-s=svc1",
				"-n", "ns",
				"--dry-run",
				"--timeout", "5d"},
			namespace: "",
			dynamic:   dc,
			input:     cs,
			wantError: true,
			hasPrefix: true,
			want:      `time: unknown unit`,
		},
		{
			name: "Dry Run with output=json -f",
			command: []string{"start",
				"-f", "./testdata/task-v1.yaml",
				"-n", "ns",
				"-s=svc1",
				"-p=foobar=context",
				"-w=name=temporary,emptyDir=",
				"--dry-run",
				"--output=json"},
			namespace:  "",
			dynamic:    dc,
			input:      cs,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with --param -f",
			command: []string{"start",
				"-f", "./testdata/task-v1.yaml",
				"-n", "ns",
				"-s=svc1",
				"--dry-run",
				"-p=foobar=context",
				"-w=name=temporary,emptyDir=",
			},
			namespace:  "",
			dynamic:    dc,
			input:      cs,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with PodTemplate",
			command: []string{"start", "task-1",
				"-p=myarg=arg",
				"-p=task-param=arg",
				"-s=svc1",
				"-n", "ns",
				"--pod-template", "./testdata/podtemplate.yaml",
				"--dry-run",
			},
			namespace:  "",
			dynamic:    dc,
			input:      cs,
			wantError:  false,
			goldenFile: true,
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube, Dynamic: tp.dynamic}
			c := Command(p)

			got, err := test.ExecuteCommand(c, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				}

				if tp.hasPrefix {
					test.AssertOutputPrefix(t, tp.want, err.Error())
				} else {
					test.AssertOutput(t, tp.want, err.Error())
				}
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

func Test_start_task_with_skip_optional_workspace_flag(t *testing.T) {
	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1.TaskSpec{
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
						Optional:    true,
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version))
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"--skip-optional-workspace",
		"-n", "ns",
	)

	expected := "TaskRun started: \n\nIn order to track the TaskRun progress run:\ntkn taskrun logs  -f -n ns\n"
	test.AssertOutput(t, expected, got)
}
