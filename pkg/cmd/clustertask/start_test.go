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

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	if version == "v1beta1" {
		localSchemeBuilder = runtime.SchemeBuilder{v1beta1.AddToScheme}
	}
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
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
	clock := test.FakeClock()
	type clients struct {
		pipelineClient test.Clients
		dynamicClient  dynamic.Interface
	}
	seeds := make([]clients, 0)
	clustertasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "clustertask-1",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
			Spec: v1beta1.TaskSpec{
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "clustertask-2",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
			Spec: v1beta1.TaskSpec{
				Params: []v1beta1.ParamSpec{
					{
						Name: "myarg",
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "clustertask-3",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
			Spec: v1beta1.TaskSpec{
				Params: []v1beta1.ParamSpec{
					{
						Name: "myarg",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "arg1",
						},
					},
					{
						Name: "print",
						Type: v1beta1.ParamTypeArray,
						Default: &v1beta1.ParamValue{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"boom", "boom"},
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "clustertask-4",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
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

	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/clusterTask": "clustertask-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				Params: []v1beta1.Param{
					{
						Name:  "myarg",
						Value: v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "value"},
					},
					{
						Name:  "print",
						Value: v1beta1.ParamValue{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"booms", "booms", "booms"}},
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

	seedData0, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns, TaskRuns: taskruns, ClusterTasks: clustertasks})
	objs0 := []runtime.Object{clustertasks[0], taskruns[0]}
	tdc0 := newDynamicClientOpt(versionB1, "taskrun-1", objs0...)
	dc0, _ := tdc0.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], versionB1),
		cb.UnstructuredV1beta1TR(taskruns[0], versionB1),
	)
	cs := test.Clients{Pipeline: seedData0.Pipeline, Kube: seedData0.Kube}
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"taskrun", "clustertask", "task"})
	// seeds[0]
	seeds = append(seeds, clients{pipelineClient: cs, dynamicClient: dc0})

	// seeds[1] - Same data as seeds[0] but creates a new TaskRun
	seedData1, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns, TaskRuns: taskruns, ClusterTasks: clustertasks})
	objs1 := []runtime.Object{clustertasks[0], taskruns[0]}
	tdc1 := newDynamicClientOpt(versionB1, "taskrun-2", objs1...)
	dc1, _ := tdc1.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], versionB1),
		cb.UnstructuredV1beta1TR(taskruns[0], versionB1),
	)
	cs1 := test.Clients{Pipeline: seedData1.Pipeline, Kube: seedData1.Kube}
	cs1.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun", "clustertask"})
	// seeds[1] - No TaskRun with ClusterTask
	seeds = append(seeds, clients{pipelineClient: cs1, dynamicClient: dc1})

	objs2 := []runtime.Object{clustertasks[0]}
	seedData2, _ := test.SeedV1beta1TestData(t, test.Data{ClusterTasks: clustertasks, Namespaces: ns, TaskRuns: taskruns})
	tdc2 := newDynamicClientOpt(versionB1, "taskrun-2", objs2...)
	dc2, _ := tdc2.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], versionB1),
	)
	cs2 := test.Clients{Pipeline: seedData2.Pipeline, Kube: seedData2.Kube}
	cs2.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun", "clustertask"})
	// seeds[2]
	seeds = append(seeds, clients{pipelineClient: cs2, dynamicClient: dc2})

	// seeds[3] - For Dry Run tests with v1beta1
	tdc3 := testDynamic.Options{}
	dc3, _ := tdc3.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], versionB1),
		cb.UnstructuredV1beta1CT(clustertasks[1], versionB1),
	)
	cs3, _ := test.SeedV1beta1TestData(t, test.Data{ClusterTasks: clustertasks, Namespaces: ns})
	cs3.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun", "clustertask"})
	// seeds[3]
	seeds = append(seeds, clients{pipelineClient: cs3, dynamicClient: dc3})

	// seeds[4] - For Dry Run tests with v1beta1
	tdc4 := testDynamic.Options{}
	dc4, _ := tdc4.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], versionB1),
		cb.UnstructuredV1beta1CT(clustertasks[1], versionB1),
	)
	cs4, _ := test.SeedV1beta1TestData(t, test.Data{ClusterTasks: clustertasks, Namespaces: ns})
	cs4.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun", "clustertask"})
	// seeds[4]
	seeds = append(seeds, clients{pipelineClient: cs4, dynamicClient: dc4})

	seedData5, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns, TaskRuns: taskruns, ClusterTasks: clustertasks})
	objs5 := []runtime.Object{clustertasks[2], taskruns[0]}
	tdc5 := newDynamicClientOpt(versionB1, "taskrun-3", objs5...)
	dc5, _ := tdc5.Client(
		cb.UnstructuredV1beta1CT(clustertasks[2], versionB1),
		cb.UnstructuredV1beta1TR(taskruns[0], versionB1),
	)
	cs5 := pipelinev1beta1test.Clients{Pipeline: seedData5.Pipeline, Kube: seedData5.Kube}
	cs5.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"taskrun", "clustertask", "task"})
	// seeds[5]
	seeds = append(seeds, clients{pipelineClient: cs, dynamicClient: dc5})

	// seeds[6] - Same data as seeds[0] but creates a new TaskRun
	seedData6, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns, TaskRuns: taskruns, ClusterTasks: clustertasks})
	objs6 := []runtime.Object{clustertasks[0], taskruns[0]}
	tdc6 := newDynamicClientOpt(versionB1, "taskrun-4", objs6...)
	dc6, _ := tdc6.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], versionB1),
		cb.UnstructuredV1beta1TR(taskruns[0], versionB1),
	)
	cs6 := test.Clients{Pipeline: seedData6.Pipeline, Kube: seedData6.Kube}
	cs6.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun", "clustertask"})
	// seeds[6] - For starting ClusterTask with use-taskrun flag
	seeds = append(seeds, clients{pipelineClient: cs6, dynamicClient: dc6})

	// seeds[7] - Same data as seeds[0] but creates a new TaskRun
	seedData7, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns, TaskRuns: taskruns, ClusterTasks: clustertasks})
	objs7 := []runtime.Object{clustertasks[3], taskruns[0]}
	tdc7 := newDynamicClientOpt(versionB1, "taskrun-5", objs7...)
	dc7, _ := tdc7.Client(
		cb.UnstructuredV1beta1CT(clustertasks[3], versionB1),
		cb.UnstructuredV1beta1TR(taskruns[0], versionB1),
	)
	cs7 := test.Clients{Pipeline: seedData7.Pipeline, Kube: seedData7.Kube}
	cs7.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun", "clustertask"})
	// seeds[7] - For starting ClusterTask with skip-optional-workspaces flag
	seeds = append(seeds, clients{pipelineClient: cs7, dynamicClient: dc7})

	testParams := []struct {
		name        string
		command     []string
		dynamic     dynamic.Interface
		input       test.Clients
		inputStream io.Reader
		wantError   bool
		hasPrefix   bool
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
				"-p", "myarg=value1",
				"-p", "print=boom,boom",
				"-l", "key=value",
				"-w", "name=test,claimName=pvc3",
				"-s=svc1"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want: "Command \"start\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\n" +
				"TaskRun started: taskrun-1\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs taskrun-1 -f -n ns\n",
		},
		{
			name:        "Start with --last option",
			command:     []string{"start", "clustertask-1", "--last"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want: "Command \"start\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\n" +
				"TaskRun started: taskrun-2\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs taskrun-2 -f -n ns\n",
		},
		{
			name:        "Start with --use-taskrun option",
			command:     []string{"start", "clustertask-1", "--use-taskrun", "taskrun-123"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want: "Command \"start\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\n" +
				"TaskRun started: taskrun-4\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs taskrun-4 -f -n ns\n",
		},
		{
			name: "Invalid param format",
			command: []string{"start", "clustertask-1",
				"-p", "myarg value1",
				"-p", "print=boom,boom",
				"-l", "key=value",
				"-w", "name=test,claimName=pvc3",
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
				"-p", "myarg=value1",
				"-p", "print=boom,boom",
				"-l", "key value",
				"-w", "name=test,claimName=pvc3",
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
				"-p", "myar=value1",
				"-p", "myarg=value1",
				"-p", "print=boom,boom",
				"-l", "key=value",
				"-w", "name=test,claimName=pvc3",
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
				"-p", "myarg=value1",
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
			name: "Dry run with no output v1beta1",
			command: []string{"start", "clustertask-2",
				"-p", "myarg=value1",
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
			name: "Dry run with --last",
			command: []string{"start", "clustertask-1",
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
				"-p", "myarg=value1",
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
			name: "Dry run with invalid --timeout v1beta1",
			command: []string{"start", "clustertask-2",
				"-p", "myarg=value1",
				"-l", "key=value",
				"-s=svc1",
				"--dry-run",
				"--timeout", "5d",
			},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			hasPrefix:   true,
			want:        `time: unknown unit`,
		},
		{
			name: "Dry run with PodTemplate",
			command: []string{"start", "clustertask-2",
				"-p", "myarg=value1",
				"-p", "print=boom",
				"-l", "key=value",
				"-l", "key=value",
				"-s=svc1",
				"--dry-run",
				"--pod-template", "./testdata/podtemplate.yaml",
			},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			goldenFile:  true,
		},
		{
			name: "Dry Run with --use-param-defaults and specified params",
			command: []string{"start", "clustertask-3",
				"-p", "myarg=value1",
				"-p", "print=boom",
				"-l", "key=value",
				"-w", "name=test,emptyDir=",
				"-s=svc1",
				"--dry-run",
				"--use-param-defaults"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   false,
			goldenFile:  true,
		},
		{
			name: "Dry Run with --use-param-defaults and no specified params",
			command: []string{"start", "clustertask-3",
				"-l", "key=value",
				"-w", "name=test,emptyDir=",
				"-s=svc1",
				"--dry-run",
				"--use-param-defaults"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   false,
			goldenFile:  true,
		},
		{
			name: "Dry Run with --use-param-defaults and --last",
			command: []string{"start", "clustertask-3",
				"-l", "key=value",
				"-w", "name=test,emptyDir=",
				"-s=svc1",
				"--dry-run",
				"--use-param-defaults",
				"--last"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "cannot use --last or --use-taskrun options with --use-param-defaults option",
		},
		{
			name: "Dry Run with --use-param-defaults and --use-taskrun",
			command: []string{"start", "clustertask-3",
				"-l", "key=value",
				"-w", "name=test,emptyDir=",
				"-s=svc1",
				"--dry-run",
				"--use-param-defaults",
				"--use-taskrun", "dummy-taskrun"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "cannot use --last or --use-taskrun options with --use-param-defaults option",
		},
		{
			name: "Dry Run with --prefix-name v1beta1",
			command: []string{"start", "clustertask-3",
				"-p", "myarg=value1",
				"-p", "print=boom",
				"-l", "key=value",
				"-w", "name=test,emptyDir=",
				"-s=svc1",
				"--dry-run",
				"--prefix-name", "customname"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   false,
			goldenFile:  true,
		},
		{
			name: "Dry Run with --prefix-name and --last v1beta1",
			command: []string{"start", "clustertask-1",
				"-l", "key=value",
				"-s=svc1",
				"--dry-run",
				"--last",
				"--prefix-name", "customname"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			goldenFile:  true,
		},
		{
			name: "Start clustertask with skip-optional-workspaces flag",
			command: []string{"start", "clustertask-4",
				"--skip-optional-workspace",
				"-s=svc1"},
			dynamic:     seeds[7].dynamicClient,
			input:       seeds[7].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want: "Command \"start\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\n" +
				"TaskRun started: taskrun-5\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs taskrun-5 -f -n ns\n",
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

				if tp.hasPrefix {
					test.AssertOutputPrefix(t, tp.want, err.Error())
				} else {
					test.AssertOutput(t, tp.want, err.Error())
				}
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

func Test_start_use_taskrun_cancelled_status(t *testing.T) {
	ctasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask",
			},
			Spec: v1beta1.TaskSpec{
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

	timeoutDuration, _ := time.ParseDuration("10s")
	trName := "ct-run"
	taskruns := []*v1beta1.TaskRun{
		{

			ObjectMeta: metav1.ObjectMeta{
				Name:      trName,
				Labels:    map[string]string{"tekton.dev/clustertask": "clustertask"},
				Namespace: "ns",
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "clustertask",
					Kind: v1beta1.ClusterTaskKind,
				},
				ServiceAccountName: "serviceaccount",
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

	seedData, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns, ClusterTasks: ctasks, TaskRuns: taskruns})
	objs := []runtime.Object{ctasks[0], taskruns[0]}
	trName2 := trName + "-2"
	tdc := newDynamicClientOpt(versionB1, trName2, objs...)

	cs := pipelinev1beta1test.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"clustertask", "taskrun"})
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1CT(ctasks[0], versionB1),
		cb.UnstructuredV1beta1TR(taskruns[0], versionB1),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	clustertask := Command(p)
	got, _ := test.ExecuteCommand(clustertask, "start", "clustertask", "--use-taskrun", trName, "-n", "ns")

	expected := "Command \"start\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nTaskRun started: ct-run-2\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs ct-run-2 -f -n ns\n"
	test.AssertOutput(t, expected, got)
	clients, _ := p.Clients()
	var tr *v1beta1.TaskRun
	if err := actions.GetV1(taskrunGroupResource, clients, trName2, "ns", metav1.GetOptions{}, &tr); err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	test.AssertOutput(t, "serviceaccount", tr.Spec.ServiceAccountName)
	test.AssertOutput(t, timeoutDuration, tr.Spec.Timeout.Duration)
	// Assert that new TaskRun does not contain cancelled status of previous run
	test.AssertOutput(t, v1beta1.TaskRunSpecStatus(""), tr.Spec.Status)
}

func Test_start_clustertask_last_override_timeout(t *testing.T) {
	ctasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask",
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

	// Add timeout to last TaskRun for ClusterTask
	timeoutDuration, _ := time.ParseDuration("10s")
	trName := "ct-run"
	taskruns := []*v1beta1.TaskRun{
		{

			ObjectMeta: metav1.ObjectMeta{
				Name:      trName,
				Labels:    map[string]string{"tekton.dev/clusterTask": "clustertask"},
				Namespace: "ns",
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "clustertask",
					Kind: v1beta1.ClusterTaskKind,
				},
				Timeout: &metav1.Duration{Duration: timeoutDuration},
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

	seedData, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns, ClusterTasks: ctasks, TaskRuns: taskruns})
	objs := []runtime.Object{ctasks[0], taskruns[0]}
	trName2 := trName + "-2"
	tdc := newDynamicClientOpt(versionB1, trName2, objs...)

	cs := pipelinev1beta1test.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"clustertask", "taskrun"})
	dc, _ := tdc.Client(
		cb.UnstructuredV1beta1CT(ctasks[0], versionB1),
		cb.UnstructuredV1beta1TR(taskruns[0], versionB1),
	)
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	clustertask := Command(p)
	// Specify new timeout value to override previous value
	got, err := test.ExecuteCommand(clustertask, "start", "clustertask", "--use-taskrun", trName, "--timeout", "1s", "-n", "ns")
	if err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	expected := "Command \"start\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nTaskRun started: ct-run-2\n\nIn order to track the TaskRun progress run:\ntkn taskrun logs ct-run-2 -f -n ns\n"
	test.AssertOutput(t, expected, got)
	clients, _ := p.Clients()
	var tr *v1beta1.TaskRun
	if err := actions.GetV1(taskrunGroupResource, clients, trName2, "ns", metav1.GetOptions{}, &tr); err != nil {
		t.Errorf("Error listing taskruns %s", err.Error())
	}

	// Assert newly started TaskRun has new timeout value
	timeoutDuration, _ = time.ParseDuration("1s")
	test.AssertOutput(t, timeoutDuration, tr.Spec.Timeout.Duration)
}
