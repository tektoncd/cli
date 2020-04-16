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

func newPipelineClient(taskRunName string, objs ...runtime.Object) *fakepipelineclientset.Clientset {
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
		obj.Name = taskRunName
		rFunc := k8stest.ObjectReaction(o)
		_, o, err := rFunc(action)
		return true, o, err
	})

	return c
}

func Test_ClusterTask_Start(t *testing.T) {
	clock := clockwork.NewFakeClock()
	seeds := make([]pipelinetest.Clients, 0)

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
		tb.TaskRun("taskrun-123", "ns",
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

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{ClusterTasks: clustertasks, Namespaces: ns, TaskRuns: taskruns})

	objs := []runtime.Object{clustertasks[0], taskruns[0]}
	pClient := newPipelineClient("taskrun-1", objs...)

	cs := pipelinetest.Clients{
		Pipeline: pClient,
		Kube:     seedData.Kube,
	}

	seeds = append(seeds, cs)

	//seeds[1] - Same data as seeds[0] but creates a new TaskRun
	objs2 := []runtime.Object{clustertasks[0], taskruns[0]}
	pClient2 := newPipelineClient("taskrun-2", objs2...)

	cs2 := pipelinetest.Clients{
		Pipeline: pClient2,
		Kube:     seedData.Kube,
	}

	//seeds[2] - No TaskRun with ClusterTask
	seeds = append(seeds, cs2)

	objs3 := []runtime.Object{clustertasks[0]}
	pClient3 := newPipelineClient("taskrun-2", objs3...)

	cs3 := pipelinetest.Clients{
		Pipeline: pClient3,
		Kube:     seedData.Kube,
	}

	seeds = append(seeds, cs3)

	//seeds[3] - For Dry Run tests with v1alpha1
	cs4, _ := test.SeedTestData(t, pipelinetest.Data{ClusterTasks: clustertasks, Namespaces: ns})
	cs4.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"clustertask"})
	seeds = append(seeds, cs4)

	//seeds[4] - For Dry Run tests with v1beta1
	cs5, _ := test.SeedTestData(t, pipelinetest.Data{ClusterTasks: clustertasks, Namespaces: ns})
	cs5.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"clustertask"})
	seeds = append(seeds, cs5)

	testParams := []struct {
		name        string
		command     []string
		input       pipelinetest.Clients
		inputStream io.Reader
		wantError   bool
		want        string
		goldenFile  bool
	}{
		{
			name:        "Start with no arguments",
			command:     []string{"start"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "missing clustertask name",
		},
		{
			name:        "ClusterTask doesn't exist",
			command:     []string{"start", "notexist"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "clustertask name notexist does not exist",
		},
		{
			name:        "Use --last with no taskruns",
			command:     []string{"start", "clustertask-1", "--last"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "no taskruns related to ClusterTask clustertask-1 found in namespace ns",
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
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "Taskrun started: taskrun-1\n\nIn order to track the taskrun progress run:\ntkn taskrun logs taskrun-1 -f -n ns\n",
		},
		{
			name:        "Start with --last option",
			command:     []string{"start", "clustertask-1", "--last"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   false,
			want:        "Taskrun started: taskrun-2\n\nIn order to track the taskrun progress run:\ntkn taskrun logs taskrun-2 -f -n ns\n",
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
			input:       seeds[0],
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
			input:       seeds[0],
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
			input:       seeds[0],
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
			input:       seeds[0],
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
			input:       seeds[0],
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
			input:       seeds[0],
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
			input:       seeds[3],
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
			input:       seeds[3],
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
			input:       seeds[4],
			inputStream: nil,
			wantError:   false,
			goldenFile:  true,
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube}
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
