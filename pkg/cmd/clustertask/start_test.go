// Copyright © 2020 The Tekton Authors.
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
	"fmt"
	"io"
	"reflect"
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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	util_runtime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	k8stest "k8s.io/client-go/testing"
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

func Test_ClusterTask_Start(t *testing.T) {
	clock := clockwork.NewFakeClock()
	type clients struct {
		pipelineClient pipelinetest.Clients
		dynamicClient  dynamic.Interface
	}
	seeds := make([]clients, 0)

	clustertasks := []*v1alpha1.ClusterTask{
		tb.ClusterTask("clustertask-1", cb.ClusterTaskCreationTime(clock.Now().Add(-1*time.Minute)),
			tb.ClusterTaskSpec(
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
					tb.StepName("hello")),
				tb.Step("busybox",
					tb.StepName("exit")),
				tb.TaskWorkspace("test", "test workspace", "/workspace/test/file", true),
			),
		),
		tb.ClusterTask("clustertask-2", cb.ClusterTaskCreationTime(clock.Now().Add(-1*time.Minute)),
			tb.ClusterTaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.Step("busybox",
					tb.StepName("hello")),
				tb.Step("busybox",
					tb.StepName("exit")),
			),
		),
	}
	taskruns := []*v1alpha1.TaskRun{
		tb.TaskRun("taskrun-123",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "clustertask-1"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("clustertask-1", tb.TaskRefKind(v1alpha1.ClusterTaskKind)),
				tb.TaskRunServiceAccountName("svc"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("myarg", "value")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("print", "booms", "booms", "booms")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("my-repo", tb.TaskResourceBindingRef("git"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("my-image", tb.TaskResourceBindingRef("image"))),
				tb.TaskRunWorkspaceEmptyDir("test", ""),
			),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(time.Now()),
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

	seedData0, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns, TaskRuns: taskruns, ClusterTasks: clustertasks})
	objs0 := []runtime.Object{clustertasks[0], taskruns[0]}
	tdc0 := newDynamicClientOpt(versionA1, "taskrun-1", objs0...)
	dc0, _ := tdc0.Client(
		cb.UnstructuredCT(clustertasks[0], versionA1),
		cb.UnstructuredTR(taskruns[0], versionA1),
	)
	cs := pipelinetest.Clients{Pipeline: seedData0.Pipeline, Kube: seedData0.Kube}
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"taskrun", "clustertask", "task"})
	//seeds[0]
	seeds = append(seeds, clients{pipelineClient: cs, dynamicClient: dc0})

	//seeds[1] - Same data as seeds[0] but creates a new TaskRun
	seedData1, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns, TaskRuns: taskruns, ClusterTasks: clustertasks})
	objs1 := []runtime.Object{clustertasks[0], taskruns[0]}
	tdc1 := newDynamicClientOpt(versionA1, "taskrun-2", objs1...)
	dc1, _ := tdc1.Client(
		cb.UnstructuredCT(clustertasks[0], versionA1),
		cb.UnstructuredTR(taskruns[0], versionA1),
	)
	cs1 := pipelinetest.Clients{Pipeline: seedData1.Pipeline, Kube: seedData1.Kube}
	cs1.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"task", "taskrun", "clustertask"})
	//seeds[1] - No TaskRun with ClusterTask
	seeds = append(seeds, clients{pipelineClient: cs1, dynamicClient: dc1})

	objs2 := []runtime.Object{clustertasks[0]}
	seedData2, _ := test.SeedTestData(t, pipelinetest.Data{ClusterTasks: clustertasks, Namespaces: ns, TaskRuns: taskruns})
	tdc2 := newDynamicClientOpt(versionA1, "taskrun-2", objs2...)
	dc2, _ := tdc2.Client(
		cb.UnstructuredCT(clustertasks[0], versionA1),
	)
	cs2 := pipelinetest.Clients{
		Pipeline: seedData2.Pipeline,
		Kube:     seedData2.Kube,
	}
	cs2.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"task", "taskrun", "clustertask"})
	//seeds[2]
	seeds = append(seeds, clients{pipelineClient: cs2, dynamicClient: dc2})

	//seeds[3] - For Dry Run tests with v1alpha1
	tdc3 := testDynamic.Options{}
	dc3, _ := tdc3.Client(
		cb.UnstructuredCT(clustertasks[0], versionA1),
		cb.UnstructuredCT(clustertasks[1], versionA1),
	)
	cs3, _ := test.SeedTestData(t, pipelinetest.Data{ClusterTasks: clustertasks, Namespaces: ns})
	cs3.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"task", "taskrun", "clustertask"})
	//seeds[3]
	seeds = append(seeds, clients{pipelineClient: cs3, dynamicClient: dc3})

	//seeds[4] - For Dry Run tests with v1beta1
	tdc4 := testDynamic.Options{}
	dc4, _ := tdc4.Client(
		cb.UnstructuredCT(clustertasks[0], versionB1),
		cb.UnstructuredCT(clustertasks[1], versionB1),
	)
	cs4, _ := test.SeedTestData(t, pipelinetest.Data{ClusterTasks: clustertasks, Namespaces: ns})
	cs4.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun", "clustertask"})
	//seeds[4]
	seeds = append(seeds, clients{pipelineClient: cs4, dynamicClient: dc4})

	//seeds[5] - Same data as seeds[0] but creates a new TaskRun
	seedData5, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns, TaskRuns: taskruns, ClusterTasks: clustertasks})
	objs5 := []runtime.Object{clustertasks[0], taskruns[0]}
	tdc5 := newDynamicClientOpt(versionA1, "taskrun-5", objs5...)
	dc5, _ := tdc5.Client(
		cb.UnstructuredCT(clustertasks[0], versionA1),
		cb.UnstructuredTR(taskruns[0], versionA1),
	)
	cs5 := pipelinetest.Clients{Pipeline: seedData5.Pipeline, Kube: seedData5.Kube}
	cs5.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"task", "taskrun", "clustertask"})
	//seeds[5] - No TaskRun with ClusterTask
	seeds = append(seeds, clients{pipelineClient: cs5, dynamicClient: dc5})

	testParams := []struct {
		name        string
		command     []string
		dynamic     dynamic.Interface
		input       pipelinetest.Clients
		inputStream io.Reader
		wantError   bool
		want        string
		goldenFile  bool
	}{
		{
			name:        "Start with no arguments",
			command:     []string{"start"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "missing ClusterTask name",
		},
		{
			name:        "ClusterTask doesn't exist",
			command:     []string{"start", "notexist"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "ClusterTask name notexist does not exist",
		},
		{
			name:        "Use --last with no taskruns",
			command:     []string{"start", "clustertask-1", "--last"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "no TaskRuns related to ClusterTask clustertask-1 found in namespace ns",
		},
		{
			name: "Start clustertask",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo=git",
				"-i", "my-image=image",
				"-p", "myarg=value1",
				"-p", "print=boom,boom",
				"-l", "key=value",
				"-o", "code-image=output-image",
				"-s=svc1"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "TaskRun started: taskrun-1\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs taskrun-1 -f -n ns\n",
		},
		{
			name: "Start clustertask with different context",
			command: []string{"start", "clustertask-1",
				"--context", "ronaldinho",
				"-i", "my-repo=git",
				"-i", "my-image=image",
				"-p", "myarg=value1",
				"-p", "print=boom,boom",
				"-l", "key=value",
				"-o", "code-image=output-image",
				"-s=svc1"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "TaskRun started: \n\nIn order to track the TaskRun progress run:\ntkn taskrun --context=ronaldinho logs  -f -n ns\n",
		},
		{
			name:        "Start with --last option",
			command:     []string{"start", "clustertask-1", "--last"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "TaskRun started: taskrun-2\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs taskrun-2 -f -n ns\n",
		},
		{
			name: "Invalid input format",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo git",
				"-i", "my-image=image",
				"-p", "myarg=value1",
				"-p", "print=boom,boom",
				"-l", "key=value",
				"-o", "code-image=output-image",
				"-s=svc1"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "invalid input format for resource parameter: my-repo git",
		},
		{
			name: "Invalid output format",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo=git",
				"-i", "my-image=image",
				"-p", "myarg=value1",
				"-p", "print=boom,boom",
				"-l", "key=value",
				"-o", "code-image output-image",
				"-s=svc1"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "invalid input format for resource parameter: code-image output-image",
		},
		{
			name: "Invalid param format",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo=git",
				"-i", "my-image=image",
				"-p", "myarg value1",
				"-p", "print=boom,boom",
				"-l", "key=value",
				"-o", "code-image=output-image",
				"-s=svc1"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "invalid input format for param parameter: myarg value1",
		},
		{
			name: "Invalid label format",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo=git",
				"-i", "my-image=image",
				"-p", "myarg=value1",
				"-p", "print=boom,boom",
				"-l", "key value",
				"-o", "code-image=output-image",
				"-s=svc1"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "invalid input format for label parameter: key value",
		},
		{
			name: "Param not in spec",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo=git",
				"-i", "my-image=image",
				"-p", "myar=value1",
				"-p", "print=boom,boom",
				"-l", "key=value",
				"-o", "code-image=output-image",
				"-s=svc1"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "param 'myar' not present in spec",
		},
		{
			name: "Dry run with invalid output",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo=git",
				"-p", "myarg=value1",
				"-o", "code-image=output-image",
				"-l", "key=value",
				"-s=svc1",
				"--dry-run",
				"--output=invalid",
			},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "output format specified is invalid but must be yaml or json",
		},
		{
			name: "Dry run with no output",
			command: []string{"start", "clustertask-2",
				"-i", "my-repo=git",
				"-o", "code-image=output-image",
				"-l", "key=value",
				"-s=svc1",
				"--dry-run",
			},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: nil,
			wantError:   false,
			goldenFile:  true,
		},
		{
			name: "Dry run with output=json",
			command: []string{"start", "clustertask-2",
				"-i", "my-repo=git",
				"-o", "code-image=output-image",
				"-l", "key=value",
				"-s=svc1",
				"--dry-run",
				"--output=json",
			},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: nil,
			wantError:   false,
			goldenFile:  true,
		},
		{
			name: "Dry run with no output v1beta1",
			command: []string{"start", "clustertask-2",
				"-i", "my-repo=git",
				"-o", "code-image=output-image",
				"-l", "key=value",
				"-s=svc1",
				"--dry-run",
			},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			goldenFile:  true,
		},
		{
			name: "Dry run with --last",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo=git",
				"-o", "code-image=output-image",
				"-l", "key=value",
				"-s=svc1",
				"--dry-run",
				"--last",
			},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			goldenFile:  true,
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube, Dynamic: tp.dynamic}
			p.SetNamespace("ns")
			clustertask := Command(p)
			if tp.inputStream != nil {
				clustertask.SetIn(tp.inputStream)
			}

			got, err := test.ExecuteCommand(clustertask, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("Error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else {
				if err != nil {
					t.Errorf("Unexpected error")
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

func Test_ClusterTask_Start_v1beta1(t *testing.T) {
	clock := clockwork.NewFakeClock()
	type clients struct {
		pipelineClient pipelinev1beta1test.Clients
		dynamicClient  dynamic.Interface
	}
	seeds := make([]clients, 0)
	clustertasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "clustertask-1",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
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
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "clustertask-2",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
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

	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "taskrun-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "clustertask-1"},
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
					Name: "clustertask-1",
					Kind: v1beta1.ClusterTaskKind,
				},
				Workspaces: []v1beta1.WorkspaceBinding{
					{
						Name:     "test",
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: clock.Now()},
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

	seedData0, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: ns, TaskRuns: taskruns, ClusterTasks: clustertasks})
	objs0 := []runtime.Object{clustertasks[0], taskruns[0]}
	tdc0 := newDynamicClientOpt(versionB1, "taskrun-1", objs0...)
	dc0, _ := tdc0.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], versionB1),
		cb.UnstructuredV1beta1TR(taskruns[0], versionB1),
	)
	cs := pipelinev1beta1test.Clients{Pipeline: seedData0.Pipeline, Kube: seedData0.Kube}
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"taskrun", "clustertask", "task"})
	//seeds[0]
	seeds = append(seeds, clients{pipelineClient: cs, dynamicClient: dc0})

	//seeds[1] - Same data as seeds[0] but creates a new TaskRun
	seedData1, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: ns, TaskRuns: taskruns, ClusterTasks: clustertasks})
	objs1 := []runtime.Object{clustertasks[0], taskruns[0]}
	tdc1 := newDynamicClientOpt(versionB1, "taskrun-2", objs1...)
	dc1, _ := tdc1.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], versionB1),
		cb.UnstructuredV1beta1TR(taskruns[0], versionB1),
	)
	cs1 := pipelinev1beta1test.Clients{Pipeline: seedData1.Pipeline, Kube: seedData1.Kube}
	cs1.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun", "clustertask"})
	//seeds[1] - No TaskRun with ClusterTask
	seeds = append(seeds, clients{pipelineClient: cs1, dynamicClient: dc1})

	objs2 := []runtime.Object{clustertasks[0]}
	seedData2, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{ClusterTasks: clustertasks, Namespaces: ns, TaskRuns: taskruns})
	tdc2 := newDynamicClientOpt(versionB1, "taskrun-2", objs2...)
	dc2, _ := tdc2.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], versionB1),
	)
	cs2 := pipelinev1beta1test.Clients{Pipeline: seedData2.Pipeline, Kube: seedData2.Kube}
	cs2.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun", "clustertask"})
	//seeds[2]
	seeds = append(seeds, clients{pipelineClient: cs2, dynamicClient: dc2})

	//seeds[3] - For Dry Run tests with v1alpha1
	tdc3 := testDynamic.Options{}
	dc3, _ := tdc3.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], versionB1),
		cb.UnstructuredV1beta1CT(clustertasks[1], versionB1),
	)
	cs3, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{ClusterTasks: clustertasks, Namespaces: ns})
	cs3.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun", "clustertask"})
	//seeds[3]
	seeds = append(seeds, clients{pipelineClient: cs3, dynamicClient: dc3})

	//seeds[4] - For Dry Run tests with v1beta1
	tdc4 := testDynamic.Options{}
	dc4, _ := tdc4.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], versionB1),
		cb.UnstructuredV1beta1CT(clustertasks[1], versionB1),
	)
	cs4, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{ClusterTasks: clustertasks, Namespaces: ns})
	cs4.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun", "clustertask"})
	//seeds[4]
	seeds = append(seeds, clients{pipelineClient: cs4, dynamicClient: dc4})

	testParams := []struct {
		name        string
		command     []string
		dynamic     dynamic.Interface
		input       pipelinev1beta1test.Clients
		inputStream io.Reader
		wantError   bool
		want        string
		goldenFile  bool
	}{
		{
			name:        "Start with no arguments",
			command:     []string{"start"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "missing ClusterTask name",
		},
		{
			name:        "ClusterTask doesn't exist",
			command:     []string{"start", "notexist"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "ClusterTask name notexist does not exist",
		},
		{
			name:        "Use --last with no taskruns",
			command:     []string{"start", "clustertask-1", "--last"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "no TaskRuns related to ClusterTask clustertask-1 found in namespace ns",
		},
		{
			name: "Start clustertask",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo=git",
				"-i", "my-image=image",
				"-p", "myarg=value1",
				"-p", "print=boom,boom",
				"-l", "key=value",
				"-o", "code-image=output-image",
				"-s=svc1"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "TaskRun started: taskrun-1\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs taskrun-1 -f -n ns\n",
		},
		{
			name:        "Start with --last option",
			command:     []string{"start", "clustertask-1", "--last"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "TaskRun started: taskrun-2\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs taskrun-2 -f -n ns\n",
		},
		{
			name: "Invalid input format",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo git",
				"-i", "my-image=image",
				"-p", "myarg=value1",
				"-p", "print=boom,boom",
				"-l", "key=value",
				"-o", "code-image=output-image",
				"-s=svc1"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "invalid input format for resource parameter: my-repo git",
		},
		{
			name: "Invalid output format",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo=git",
				"-i", "my-image=image",
				"-p", "myarg=value1",
				"-p", "print=boom,boom",
				"-l", "key=value",
				"-o", "code-image output-image",
				"-s=svc1"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "invalid input format for resource parameter: code-image output-image",
		},
		{
			name: "Invalid param format",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo=git",
				"-i", "my-image=image",
				"-p", "myarg value1",
				"-p", "print=boom,boom",
				"-l", "key=value",
				"-o", "code-image=output-image",
				"-s=svc1"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "invalid input format for param parameter: myarg value1",
		},
		{
			name: "Invalid label format",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo=git",
				"-i", "my-image=image",
				"-p", "myarg=value1",
				"-p", "print=boom,boom",
				"-l", "key value",
				"-o", "code-image=output-image",
				"-s=svc1"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "invalid input format for label parameter: key value",
		},
		{
			name: "Param not in spec",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo=git",
				"-i", "my-image=image",
				"-p", "myar=value1",
				"-p", "print=boom,boom",
				"-l", "key=value",
				"-o", "code-image=output-image",
				"-s=svc1"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "param 'myar' not present in spec",
		},
		{
			name: "Dry run with invalid output",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo=git",
				"-p", "myarg=value1",
				"-o", "code-image=output-image",
				"-l", "key=value",
				"-s=svc1",
				"--dry-run",
				"--output=invalid",
			},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "output format specified is invalid but must be yaml or json",
		},
		{
			name: "Dry run with no output",
			command: []string{"start", "clustertask-2",
				"-i", "my-repo=git",
				"-o", "code-image=output-image",
				"-l", "key=value",
				"-s=svc1",
				"--dry-run",
			},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: nil,
			wantError:   false,
			goldenFile:  true,
		},
		{
			name: "Dry run with no output v1beta1",
			command: []string{"start", "clustertask-2",
				"-i", "my-repo=git",
				"-o", "code-image=output-image",
				"-l", "key=value",
				"-s=svc1",
				"--dry-run",
			},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			goldenFile:  true,
		},
		{
			name: "Dry run with --last",
			command: []string{"start", "clustertask-1",
				"-i", "my-repo=git",
				"-o", "code-image=output-image",
				"-l", "key=value",
				"-s=svc1",
				"--dry-run",
				"--last",
			},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			goldenFile:  true,
		},
		{
			name: "Dry run with --timeout v1beta1",
			command: []string{"start", "clustertask-2",
				"-i", "my-repo=git",
				"-o", "code-image=output-image",
				"-l", "key=value",
				"-s=svc1",
				"--dry-run",
				"--timeout", "5s",
			},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			goldenFile:  true,
		},
		{
			name: "Dry run with invalide --timeout v1beta1",
			command: []string{"start", "clustertask-2",
				"-i", "my-repo=git",
				"-o", "code-image=output-image",
				"-l", "key=value",
				"-s=svc1",
				"--dry-run",
				"--timeout", "5d",
			},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "time: unknown unit d in duration 5d",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube, Dynamic: tp.dynamic}
			p.SetNamespace("ns")
			clustertask := Command(p)
			if tp.inputStream != nil {
				clustertask.SetIn(tp.inputStream)
			}

			got, err := test.ExecuteCommand(clustertask, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("Error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else {
				if err != nil {
					t.Errorf("Unexpected error")
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

func Test_mergeResource_v1beta1(t *testing.T) {
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

func Test_parseRes_v1beta1(t *testing.T) {
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
