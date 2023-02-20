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
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/cli/pkg/actions"
	traction "github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
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
	localSchemeBuilder := runtime.SchemeBuilder{v1beta1.AddToScheme}

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

					v1beta1TR := obj.(*v1beta1.TaskRun)
					unstructuredTR := cb.UnstructuredV1beta1TR(v1beta1TR, version)
					return true, unstructuredTR, nil
				},
			},
		},
	}
	return nil, dc
}

func Test_start_invalid_namespace(t *testing.T) {
	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{})
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
	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c := Command(&test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc})

	got, err := test.ExecuteCommand(c, "start", "-n", "ns", "--filename=./testdata/task.yaml",
		"-i=docker-source=/path", "-o=build-image=image", "-p=pathToDockerFile=path", "--use-param-defaults")
	if err != nil {
		t.Errorf("Not expecting an error, but got %s", err.Error())
	}

	expected := "TaskRun started: \n\nIn order to track the TaskRun progress run:\ntkn taskrun logs  -f -n ns\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_filename_param_with_invalid_type(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c := Command(&test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc})

	got, err := test.ExecuteCommand(c, "start", "-n", "ns", "--filename=./testdata/task-param-with-invalid-type.yaml", "-i=docker-source=/path", "-o=build-image=image", "-p=pathToDockerFile=path")
	if err == nil {
		t.Errorf("expected an error but didn't get one")
	}

	expected := "Error: params does not have a valid type - 'pathToDockerFile'\n"
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
	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c := Command(&test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc})

	_, err = test.ExecuteCommand(c, "start", "-n", "ns", "--filename=./testdata/task-invalid.yaml")
	if err == nil {
		t.Errorf("expected an error but didn't get one")
	}

	expected := `error unmarshaling JSON: while decoding JSON: json: unknown field "param"`
	test.AssertOutput(t, expected, err.Error())
}

func Test_start_task_not_found(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeString,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version))
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
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version))
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)

	gotConfig, _ := test.ExecuteCommand(task, "start", "task-1",
		"--context=NinjaRabbit",
		"-i=my-repo=git",
		"-i=my-image=image",
		"-p=myarg=value1",
		"-p=print=boom,boom",
		"-l=key=value",
		"-o=code-image=output-image",
		"-w=name=pvc,claimName=pvc3",
		"-s=svc1",
		"-n=ns")

	gcExpected := "TaskRun started: \n\nIn order to track the TaskRun progress run:\ntkn taskrun --context=NinjaRabbit logs  -f -n ns\n"
	test.AssertOutput(t, gcExpected, gotConfig)

}

func Test_start_task(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version))
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

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

	test.AssertOutput(t, "pvc", tr.Items[0].Spec.Workspaces[0].Name)
	test.AssertOutput(t, "pvc3", tr.Items[0].Spec.Workspaces[0].PersistentVolumeClaim.ClaimName)

	test.AssertOutput(t, "svc1", tr.Items[0].Spec.ServiceAccountName)
}

func Test_start_task_last(t *testing.T) {
	// TODO: this should be fixed with start command
	t.Skip()
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	timeoutDuration, _ := time.ParseDuration("10s")
	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				Params: []v1beta1.Param{
					{
						Name:  "myarg",
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "value"},
					},
					{
						Name:  "print",
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"booms", "booms", "booms"}},
					},
				},
				Resources: &v1beta1.TaskRunResources{
					Inputs: []v1beta1.TaskResourceBinding{
						{
							PipelineResourceBinding: v1beta1.PipelineResourceBinding{
								Name: "my-repo",
								ResourceRef: &v1beta1.PipelineResourceRef{
									Name: "git",
								},
							},
						},
					},
					Outputs: []v1beta1.TaskResourceBinding{
						{
							PipelineResourceBinding: v1beta1.PipelineResourceBinding{
								Name: "code-image",
								ResourceRef: &v1beta1.PipelineResourceRef{
									Name: "image",
								},
							},
						},
					},
				},
				ServiceAccountName: "svc",
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
				},
				Timeout: &metav1.Duration{Duration: timeoutDuration},
				Workspaces: []v1beta1.WorkspaceBinding{
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

	seedData, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})
	objs := []runtime.Object{tasks[0], taskruns[0]}
	_, tdc := newPipelineClient(objs...)
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1TR(taskruns[0], version),
	)

	cs := pipelinetest.Clients{
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
	tr, err := traction.Get(clients, "random", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	for _, v := range tr.Spec.Resources.Inputs {
		if v.Name == "my-repo" {
			test.AssertOutput(t, "git", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, 2, len(tr.Spec.Params))

	for _, v := range tr.Spec.Params {
		if v.Name == "my-arg" {
			test.AssertOutput(t, v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "value"}, v.Value)
		}

		if v.Name == "print" {
			test.AssertOutput(t, v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"booms", "booms", "booms"}}, v.Value)
		}
	}

	for _, v := range tr.Spec.Resources.Outputs {
		if v.Name == "code-image" {
			test.AssertOutput(t, "image", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, "svc", tr.Spec.ServiceAccountName)
	test.AssertOutput(t, "test", tr.Spec.Workspaces[0].Name)
	test.AssertOutput(t, "", tr.Spec.Workspaces[0].SubPath)
	test.AssertOutput(t, timeoutDuration, tr.Spec.Timeout.Duration)
}

func Test_start_task_last_with_override_timeout(t *testing.T) {
	// TODO: this should be fixed with start command
	t.Skip()
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
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

	// Add timeout to last TaskRun for Task
	timeoutDuration, _ := time.ParseDuration("10s")
	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				Timeout: &metav1.Duration{Duration: timeoutDuration},
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
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

	seedData, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})
	objs := []runtime.Object{tasks[0], taskruns[0]}
	_, tdc := newPipelineClient(objs...)
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1TR(taskruns[0], version),
	)

	cs := pipelinetest.Clients{
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
	gotTR, err := traction.Get(clients, "random", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	// Assert newly started TaskRun has new timeout value
	timeoutDuration, _ = time.ParseDuration("1s")
	test.AssertOutput(t, timeoutDuration, gotTR.Spec.Timeout.Duration)
}

func Test_start_use_taskrun(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	timeoutDuration, _ := time.ParseDuration("10s")

	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "happy",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
		},
		{

			ObjectMeta: metav1.ObjectMeta{
				Name:      "camper",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
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

	seedData, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})

	objs := []runtime.Object{tasks[0], taskruns[0], taskruns[1]}
	_, tdc := newPipelineClient(objs...)

	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1TR(taskruns[0], version),
		cb.UnstructuredV1beta1TR(taskruns[1], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"--use-taskrun", "camper",
		"-n=ns")

	expected := "TaskRun started: random\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)
	clients, _ := p.Clients()
	tr, err := traction.Get(clients, "random", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "camper", tr.Spec.ServiceAccountName)
	test.AssertOutput(t, timeoutDuration, tr.Spec.Timeout.Duration)
}

func Test_start_use_taskrun_cancelled_status(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	timeoutDuration, _ := time.ParseDuration("10s")

	taskruns := []*v1beta1.TaskRun{
		{

			ObjectMeta: metav1.ObjectMeta{
				Name:      "camper",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
				},
				ServiceAccountName: "camper",
				Timeout:            &metav1.Duration{Duration: timeoutDuration},
				Status:             v1beta1.TaskRunSpecStatus(v1beta1.TaskRunSpecStatusCancelled),
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

	seedData, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})

	objs := []runtime.Object{tasks[0], taskruns[0]}
	_, tdc := newPipelineClient(objs...)

	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1TR(taskruns[0], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"--use-taskrun", "camper",
		"-n=ns")

	expected := "TaskRun started: random\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)
	clients, _ := p.Clients()
	tr, err := traction.Get(clients, "random", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "camper", tr.Spec.ServiceAccountName)
	test.AssertOutput(t, timeoutDuration, tr.Spec.Timeout.Duration)
	// Assert that new TaskRun does not contain cancelled status of previous run
	test.AssertOutput(t, v1beta1.TaskRunSpecStatus(""), tr.Spec.Status)
}

func Test_start_task_last_generate_name(t *testing.T) {
	// TODO: this should be fixed with start command
	t.Skip()
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	timeoutDuration, _ := time.ParseDuration("10s")

	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				Params: []v1beta1.Param{
					{
						Name:  "myarg",
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "value"},
					},
					{
						Name:  "print",
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"booms", "booms", "booms"}},
					},
				},
				Resources: &v1beta1.TaskRunResources{
					Inputs: []v1beta1.TaskResourceBinding{
						{
							PipelineResourceBinding: v1beta1.PipelineResourceBinding{
								Name: "my-repo",
								ResourceRef: &v1beta1.PipelineResourceRef{
									Name: "git",
								},
							},
						},
					},
					Outputs: []v1beta1.TaskResourceBinding{
						{
							PipelineResourceBinding: v1beta1.PipelineResourceBinding{
								Name: "code-image",
								ResourceRef: &v1beta1.PipelineResourceRef{
									Name: "image",
								},
							},
						},
					},
				},
				ServiceAccountName: "svc",
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
				},
				Timeout: &metav1.Duration{Duration: timeoutDuration},
				Workspaces: []v1beta1.WorkspaceBinding{
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

	seedData, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})

	objs := []runtime.Object{tasks[0], taskruns[0]}
	_, tdc := newPipelineClient(objs...)

	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1TR(taskruns[0], version))
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"--last",
		"-n=ns")

	expected := "TaskRun started: random\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)
	clients, _ := p.Clients()
	tr, err := traction.Get(clients, "random", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "test-generatename-task-run-", tr.ObjectMeta.GenerateName)
}

func Test_start_task_last_with_prefix_name(t *testing.T) {
	// TODO: this should be fixed with start command
	t.Skip()
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	timeoutDuration, _ := time.ParseDuration("10s")

	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				Params: []v1beta1.Param{
					{
						Name:  "myarg",
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "value"},
					},
					{
						Name:  "print",
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"booms", "booms", "booms"}},
					},
				},
				Resources: &v1beta1.TaskRunResources{
					Inputs: []v1beta1.TaskResourceBinding{
						{
							PipelineResourceBinding: v1beta1.PipelineResourceBinding{
								Name: "my-repo",
								ResourceRef: &v1beta1.PipelineResourceRef{
									Name: "git",
								},
							},
						},
					},
					Outputs: []v1beta1.TaskResourceBinding{
						{
							PipelineResourceBinding: v1beta1.PipelineResourceBinding{
								Name: "code-image",
								ResourceRef: &v1beta1.PipelineResourceRef{
									Name: "image",
								},
							},
						},
					},
				},
				ServiceAccountName: "svc",
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
				},
				Timeout: &metav1.Duration{Duration: timeoutDuration},
				Workspaces: []v1beta1.WorkspaceBinding{
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

	seedData, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})

	objs := []runtime.Object{tasks[0], taskruns[0]}
	_, tdc := newPipelineClient(objs...)

	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1TR(taskruns[0], version))
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
	tr, err := traction.Get(clients, "random", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "mytrname-", tr.ObjectMeta.GenerateName)
}

func Test_start_task_with_prefix_name(t *testing.T) {
	// TODO: this should be fixed with start command
	t.Skip()
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	timeoutDuration, _ := time.ParseDuration("10s")

	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				Params: []v1beta1.Param{
					{
						Name:  "myarg",
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "value"},
					},
					{
						Name:  "print",
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"booms", "booms", "booms"}},
					},
				},
				Resources: &v1beta1.TaskRunResources{
					Inputs: []v1beta1.TaskResourceBinding{
						{
							PipelineResourceBinding: v1beta1.PipelineResourceBinding{
								Name: "my-repo",
								ResourceRef: &v1beta1.PipelineResourceRef{
									Name: "git",
								},
							},
						},
					},
					Outputs: []v1beta1.TaskResourceBinding{
						{
							PipelineResourceBinding: v1beta1.PipelineResourceBinding{
								Name: "code-image",
								ResourceRef: &v1beta1.PipelineResourceRef{
									Name: "image",
								},
							},
						},
					},
				},
				ServiceAccountName: "svc",
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
				},
				Timeout: &metav1.Duration{Duration: timeoutDuration},
				Workspaces: []v1beta1.WorkspaceBinding{
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

	seedData, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})
	objs := []runtime.Object{tasks[0], taskruns[0]}
	_, tdc := newPipelineClient(objs...)
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1TR(taskruns[0], version))
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"-n=ns",
		"--prefix-name=mytrname",
		"--last",
	)

	expected := "TaskRun started: random\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	clients, _ := p.Clients()
	tr, err := traction.Get(clients, "random", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "mytrname-", tr.ObjectMeta.GenerateName)
}

func Test_start_task_last_with_inputs(t *testing.T) {
	// TODO: this should be fixed with start command
	t.Skip()
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	timeoutDuration, _ := time.ParseDuration("10s")

	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				Params: []v1beta1.Param{
					{
						Name:  "myarg",
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "value"},
					},
					{
						Name:  "print",
						Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"booms", "booms", "booms"}},
					},
				},
				Resources: &v1beta1.TaskRunResources{
					Inputs: []v1beta1.TaskResourceBinding{
						{
							PipelineResourceBinding: v1beta1.PipelineResourceBinding{
								Name: "my-repo",
								ResourceRef: &v1beta1.PipelineResourceRef{
									Name: "git",
								},
							},
						},
					},
					Outputs: []v1beta1.TaskResourceBinding{
						{
							PipelineResourceBinding: v1beta1.PipelineResourceBinding{
								Name: "code-image",
								ResourceRef: &v1beta1.PipelineResourceRef{
									Name: "image",
								},
							},
						},
					},
				},
				ServiceAccountName: "svc",
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
				},
				Timeout: &metav1.Duration{Duration: timeoutDuration},
				Workspaces: []v1beta1.WorkspaceBinding{
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

	seedData, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks, TaskRuns: taskruns})
	objs := []runtime.Object{tasks[0], taskruns[0]}
	_, tdc := newPipelineClient(objs...)
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1TR(taskruns[0], version))
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task",
		"-i=my-repo=git-new",
		"-p=myarg=value1",
		"-p=print=boom,boom",
		"-o=code-image=output-image",
		"-s=svc1",
		"-n=ns",
		"--last")

	expected := "TaskRun started: random\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	clients, _ := p.Clients()
	tr, err := traction.Get(clients, "random", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	for _, v := range tr.Spec.Resources.Inputs {
		if v.Name == "my-repo" {
			test.AssertOutput(t, "git-new", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, 2, len(tr.Spec.Params))

	for _, v := range tr.Spec.Params {
		if v.Name == "my-arg" {
			test.AssertOutput(t, v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "value1"}, v.Value)
		}

		if v.Name == "print" {
			test.AssertOutput(t, v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"boom", "boom"}}, v.Value)
		}
	}

	for _, v := range tr.Spec.Resources.Outputs {
		if v.Name == "code-image" {
			test.AssertOutput(t, "output-image", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, "svc1", tr.Spec.ServiceAccountName)
}

func Test_start_task_last_without_taskrun(t *testing.T) {
	// TODO: this should be fixed with start command
	t.Skip()
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	objs := []runtime.Object{tasks[0]}
	_, tdc := newPipelineClient(objs...)
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
	)

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1", "--last", "-n", "ns")
	expected := "Error: no TaskRuns related to Task task-1 found in namespace ns\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_client_error(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
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
		cb.UnstructuredV1beta1T(tasks[0], version),
	)

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task,
		"start", "task-1", "-n", "ns",
		"-i=my-repo=git",
		"-i=my-image=image",
		"-p=myarg=value1",
		"-p=print=boom,boom",
		"-o=code-image=image",
	)
	expected := "Error: cluster not accessible\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_invalid_input_res(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
	)

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-i=my-repo git-repo",
		"-i=my-image=lkadjf",
		"-o=some=some",
		"-p=some=some",
		"-n", "ns",
		"-p=myarg=abc",
		"-p=print=xyz",
	)
	expected := "Error: invalid input format for resource parameter: my-repo git-repo\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_invalid_workspace(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
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
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns, Tasks: tasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-w=claimName=pvc3",
		"-i=my-image=lkadjf",
		"-o=code-image=some",
		"-n", "ns",
	)
	expected := "Error: Name not found for workspace\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_invalid_output_res(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-o", "code-image image-final",
		"-i=my-repo=something",
		"-p=print=cat",
		"-n", "ns",
		"-p=myarg=abc",
		"-p=print=xyz",
	)
	expected := "Error: invalid input format for resource parameter: code-image image-final\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_invalid_param(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-p", "myarg boom",
		"-i=my-repo=repo",
		"-o=out=out",
		"-n", "ns",
	)
	expected := "Error: invalid input format for param parameter: myarg boom\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_invalid_label(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-l", "myarg boom",
		"-i=input=input",
		"-o=out=out",
		"-p=param=param",
		"-n", "ns",
		"-p=myarg=abc",
		"-p=print=xyz",
	)
	expected := "Error: invalid input format for label parameter: myarg boom\n"
	test.AssertOutput(t, expected, got)
}

func Test_start_task_allkindparam(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
					},
					{
						Name: "printafter",
						Type: v1beta1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

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

	test.AssertOutput(t, 3, len(tr.Items[0].Spec.Params))

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
	}

	if d := cmp.Equal(tr.Items[0].ObjectMeta.Labels, map[string]string{"key": "value"}); !d {
		t.Errorf("Error labels generated is different Labels Got: %+v", tr.Items[0].ObjectMeta.Labels)
	}

	test.AssertOutput(t, "svc1", tr.Items[0].Spec.ServiceAccountName)
}

func Test_start_task_wrong_param(t *testing.T) {
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
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
						Name: "print",
						Type: v1beta1.ParamTypeArray,
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	task := Command(p)
	got, _ := test.ExecuteCommand(task, "start", "task-1",
		"-i=my-repo=git",
		"-i=my-image=image",
		"-p=myar=value1",
		"-p=myarg=value1",
		"-p=print=boom,boom",
		"-l=key=value",
		"-o=code-image=output-image",
		"-s=svc1",
		"-n=ns")

	expected := "Error: param 'myar' not present in spec\n"
	test.AssertOutput(t, expected, got)
}

func Test_mergeResource(t *testing.T) {
	res := []v1beta1.TaskResourceBinding{{
		PipelineResourceBinding: v1beta1.PipelineResourceBinding{
			Name: "source",
			ResourceRef: &v1beta1.PipelineResourceRef{
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
		want    map[string]v1beta1.TaskResourceBinding
		wantErr bool
	}{{
		name: "Test_parseRes No Err",
		args: args{
			res: []string{"source=git", "image=docker2"},
		},
		want: map[string]v1beta1.TaskResourceBinding{"source": {
			PipelineResourceBinding: v1beta1.PipelineResourceBinding{
				Name: "source",
				ResourceRef: &v1beta1.PipelineResourceRef{
					Name: "git",
				},
			},
		}, "image": {
			PipelineResourceBinding: v1beta1.PipelineResourceBinding{
				Name: "image",
				ResourceRef: &v1beta1.PipelineResourceRef{
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
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
				Resources: &v1beta1.TaskResources{
					Inputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "my-repo",
								Type: v1beta1.PipelineResourceTypeGit,
							},
						},
					},
					Outputs: []v1beta1.TaskResource{
						{
							ResourceDeclaration: v1beta1.ResourceDeclaration{
								Name: "code-image",
								Type: v1beta1.PipelineResourceTypeImage,
							},
						},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "myarg",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "arg1",
						},
					},
					{
						Name: "task-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "my-param",
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

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
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
				"-i=my-repo=git-repo",
				"-o=code-image=output-image",
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
				"-i=my-repo=git-repo",
				"-o=code-image=output-image",
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
				"-i=my-repo=git-repo",
				"-o=code-image=output-image",
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
				"-i=my-repo=git-repo",
				"-o=code-image=output-image",
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
				"-i=my-repo=git-repo",
				"-o=code-image=output-image",
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
				"-i=my-repo=git-repo",
				"-o=code-image=output-image",
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
				"-i=my-repo=git-repo",
				"-o=code-image=output-image",
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
				"-i=my-repo=git-repo",
				"-o=code-image=output-image",
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
				"-f", "./testdata/task.yaml",
				"-n", "ns",
				"-s=svc1",
				"-i=docker-source=git",
				"-o=builtImage=image",
				"-p=pathToContext=/context",
				"-p=pathToDockerFile=/path",
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
				"-i=my-repo=git-repo",
				"-o=code-image=output-image",
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
				"-i=my-repo=git-repo",
				"-o=code-image=output-image",
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
				"-f", "./testdata/task.yaml",
				"-n", "ns",
				"-s=svc1",
				"-p=pathToContext=/context",
				"-p=pathToDockerFile=/path",
				"-i=docker-source=git",
				"-o=builtImage=image",
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
				"-f", "./testdata/task.yaml",
				"-n", "ns",
				"-s=svc1",
				"-i=docker-source=git",
				"-o=builtImage=image",
				"--dry-run",
				"-p=pathToContext=/context",
				"-p=pathToDockerFile=/path",
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
				"-i=my-repo=git-repo",
				"-o=code-image=output-image",
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
	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task-1",
				Namespace: "ns",
			},
			Spec: v1beta1.TaskSpec{
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

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Tasks: tasks, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version))
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
