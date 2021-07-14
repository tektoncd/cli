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

package taskrun

import (
	"reflect"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	util_runtime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stest "k8s.io/client-go/testing"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func newDynamicClientOpt(version, taskRunName string, objs ...runtime.Object) testDynamic.Options {
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	var localSchemeBuilder runtime.SchemeBuilder
	if version == "v1alpha1" {
		localSchemeBuilder = runtime.SchemeBuilder{v1alpha1.AddToScheme}
	} else {
		localSchemeBuilder = runtime.SchemeBuilder{v1beta1.AddToScheme}
	}
	v1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	util_runtime.Must(localSchemeBuilder.AddToScheme(scheme))

	o := k8stest.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objs {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	tdc := testDynamic.Options{
		AddReactorRes:  "*",
		AddReactorVerb: "*",
		AddReactorFun:  k8stest.ObjectReaction(o),
		WatchResource:  "*",
		WatchReactionFun: func(action k8stest.Action) (handled bool, ret watch.Interface, err error) {
			gvr := action.GetResource()
			ns := action.GetNamespace()
			watch, err := o.Watch(gvr, ns)
			if err != nil {
				return false, nil, err
			}
			return true, watch, nil
		},
		PrependReactors: []testDynamic.PrependOpt{
			{
				Resource: "taskruns",
				Verb:     "create",
				Action: func(action k8stest.Action) (bool, runtime.Object, error) {
					create := action.(k8stest.CreateActionImpl)
					unstructuredTR := create.GetObject().(*unstructured.Unstructured)
					unstructuredTR.SetName(taskRunName)
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

					if res.Version == "v1alpha1" {
						v1alpha1TR := obj.(*v1alpha1.TaskRun)
						unstructuredTR := cb.UnstructuredTR(v1alpha1TR, versionA1)
						return true, unstructuredTR, nil
					}
					v1beta1TR := obj.(*v1beta1.TaskRun)
					unstructuredTR := cb.UnstructuredV1beta1TR(v1beta1TR, versionB1)
					return true, unstructuredTR, nil
				},
			},
		},
	}

	return tdc
}

func getTask() *v1beta1.Task {
	return &v1beta1.Task{
		ObjectMeta: v1.ObjectMeta{
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
	}
}

func getTaskRun(taskSpec bool, generateName string) *v1beta1.TaskRun {
	clock := clockwork.NewFakeClock()
	if !taskSpec {
		return &v1beta1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    "ns",
				Name:         "tr-1",
				GenerateName: generateName,
				Labels:       map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
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
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
				},
			},
		}
	}

	return &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    "ns",
			Name:         "tr-1",
			GenerateName: generateName,
		},
		Spec: v1beta1.TaskRunSpec{
			TaskSpec: &v1beta1.TaskSpec{
				Steps: []v1beta1.Step{
					{
						Container: corev1.Container{
							Name:    "hello",
							Image:   "busybox",
							Command: []string{"echo"},
							Args:    []string{"hello"},
						},
					},
				},
			},
		},
	}
}

func Test_restart_no_args(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{})
	taskrun := Command(&test.Params{Tekton: cs.Pipeline})

	out, err := test.ExecuteCommand(taskrun, "restart")
	if err == nil {
		t.Errorf("expected an error but got:\n%s", out)
	}

	test.AssertOutput(t, "Error: accepts 1 arg(s), received 0\n", out)
}

func Test_restart_no_taskrun_found(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{})
	taskrun := Command(&test.Params{Tekton: cs.Pipeline})

	out, err := test.ExecuteCommand(taskrun, "restart", "notexist")
	if err == nil {
		t.Errorf("expected an error but got:\n%s", out)
	}

	test.AssertOutput(t, "Error: failed to find TaskRun: notexist\n", out)
}

func Test_restart_with_no_generate_name(t *testing.T) {
	taskRun1 := getTaskRun(false, "")
	taskRuns := []*v1beta1.TaskRun{
		taskRun1,
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(taskRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	taskrun := Command(p)

	out, err := test.ExecuteCommand(taskrun, "restart", taskRun1.Name, "-n", "ns")
	if err == nil {
		t.Errorf("expected an error but got:\n%s", out)
	}

	test.AssertOutput(t, "Error: generateName field required for TaskRun with restart\nUse --prefix-name option if TaskRun does not have generateName set\n", out)
}

func Test_restart_taskrun_successfully(t *testing.T) {
	task1 := getTask()
	tasks := []*v1beta1.Task{
		task1,
	}

	taskRun1 := getTaskRun(false, "tr-")
	taskRuns := []*v1beta1.TaskRun{
		taskRun1,
	}

	version := "v1beta1"
	objs := []runtime.Object{tasks[0], taskRuns[0]}
	tdc := newDynamicClientOpt(version, "tr-2", objs...)
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1TR(taskRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: taskRuns, Tasks: tasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun", "task"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	taskrun := Command(p)
	out, err := test.ExecuteCommand(taskrun, "restart", "tr-1", "-n", "ns")
	if err != nil {
		t.Errorf("error occurred: %v", err)
	}

	expected := "TaskRun started: tr-2\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs tr-2 -f -n ns\n"
	test.AssertOutput(t, expected, out)
}

func Test_restart_taskrun_with_taskspec_successfully(t *testing.T) {
	task1 := getTask()
	tasks := []*v1beta1.Task{
		task1,
	}

	taskRun1 := getTaskRun(true, "tr-")
	taskRuns := []*v1beta1.TaskRun{
		taskRun1,
	}

	version := "v1beta1"
	objs := []runtime.Object{tasks[0], taskRuns[0]}
	tdc := newDynamicClientOpt(version, "tr-2", objs...)
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1TR(taskRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: taskRuns, Tasks: tasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun", "task"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	taskrun := Command(p)
	out, err := test.ExecuteCommand(taskrun, "restart", "tr-1", "-n", "ns")
	if err != nil {
		t.Errorf("error occurred: %v", err)
	}

	expected := "TaskRun started: tr-2\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs tr-2 -f -n ns\n"
	test.AssertOutput(t, expected, out)
}

func Test_restart_taskrun_with_prefix_name(t *testing.T) {
	task1 := getTask()
	tasks := []*v1beta1.Task{
		task1,
	}

	taskRun1 := getTaskRun(true, "")
	taskRuns := []*v1beta1.TaskRun{
		taskRun1,
	}

	version := "v1beta1"
	objs := []runtime.Object{tasks[0], taskRuns[0]}
	tdc := newDynamicClientOpt(version, "prefix-test-12345", objs...)
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1TR(taskRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: taskRuns, Tasks: tasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun", "task"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	taskrun := Command(p)
	out, err := test.ExecuteCommand(taskrun, "restart", "tr-1", "-n", "ns", "--prefix-name", "prefix-test")
	if err != nil {
		t.Errorf("error occurred: %v", err)
	}

	expected := "TaskRun started: prefix-test-12345\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs prefix-test-12345 -f -n ns\n"
	test.AssertOutput(t, expected, out)
}

func Test_restart_taskrun_with_context(t *testing.T) {
	task1 := getTask()
	tasks := []*v1beta1.Task{
		task1,
	}

	taskRun1 := getTaskRun(true, "tr-")
	taskRuns := []*v1beta1.TaskRun{
		taskRun1,
	}

	version := "v1beta1"
	objs := []runtime.Object{tasks[0], taskRuns[0]}
	tdc := newDynamicClientOpt(version, "tr-2", objs...)
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1TR(taskRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: taskRuns, Tasks: tasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun", "task"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	taskrun := Command(p)
	out, err := test.ExecuteCommand(taskrun, "restart", "tr-1", "-n", "ns", "--context", "minikube")
	if err != nil {
		t.Errorf("error occurred: %v", err)
	}

	expected := "TaskRun started: tr-2\n\nIn order to track the TaskRun progress run:\ntkn taskrun --context=minikube logs tr-2 -f -n ns\n"
	test.AssertOutput(t, expected, out)
}
