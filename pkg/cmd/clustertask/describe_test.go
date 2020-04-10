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
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stest "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
)

func Test_ClusterTaskDescribe(t *testing.T) {
	clock := clockwork.NewFakeClock()

	clustertasks := []*v1alpha1.ClusterTask{
		tb.ClusterTask("clustertask-full", cb.ClusterTaskCreationTime(clock.Now().Add(1*time.Minute)),
			tb.ClusterTaskSpec(
				tb.TaskResources(
					tb.TaskResourcesInput("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.TaskResourcesInput("my-image", v1alpha1.PipelineResourceTypeImage),
					tb.TaskResourcesOutput("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.TaskParam("myarg", v1alpha1.ParamTypeString, tb.ParamSpecDefault("default")),
				tb.TaskParam("print", v1alpha1.ParamTypeArray, tb.ParamSpecDefault("booms", "booms", "booms")),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
				tb.Step("busybox",
					tb.StepName("exit"),
				),
			),
		),
		tb.ClusterTask("clustertask-one-everything", cb.ClusterTaskCreationTime(clock.Now().Add(1*time.Minute)),
			tb.ClusterTaskSpec(
				tb.TaskDescription("a test description"),
				tb.TaskResources(
					tb.TaskResourcesInput("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.TaskResourcesOutput("code-image", v1alpha1.PipelineResourceTypeImage),
				),
				tb.TaskParam("myarg", v1alpha1.ParamTypeString),
				tb.Step("busybox",
					tb.StepName("hello"),
				),
			),
		),
		tb.ClusterTask("clustertask-justname", cb.ClusterTaskCreationTime(clock.Now().Add(-1*time.Minute))),
	}

	taskruns := []*v1alpha1.TaskRun{
		tb.TaskRun("taskrun-1", "ns",
			tb.TaskRunLabel("tekton.dev/task", "clustertask-full"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("clustertask-full", tb.TaskRefKind(v1alpha1.ClusterTaskKind)),
				tb.TaskRunServiceAccountName("svc"),
				tb.TaskRunParam("myarg", "value"),
				tb.TaskRunParam("print", "booms", "booms", "booms"),
				tb.TaskRunResources(tb.TaskRunResourcesInput("my-repo", tb.TaskResourceBindingRef("git"))),
				tb.TaskRunResources(tb.TaskRunResourcesOutput("my-image", tb.TaskResourceBindingRef("image"))),
			),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(-10*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(17*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
		tb.TaskRun("taskrun-2", "ns",
			tb.TaskRunLabel("tekton.dev/task", "clustertask-one-everything"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("clustertask-one-everything", tb.TaskRefKind(v1alpha1.ClusterTaskKind)),
				tb.TaskRunServiceAccountName("svc"),
				tb.TaskRunParam("myarg", "value"),
				tb.TaskRunResources(tb.TaskRunResourcesInput("my-repo", tb.TaskResourceBindingRef("git"))),
			),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(-10*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(17*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
		tb.TaskRun("taskrun-3", "ns",
			tb.TaskRunLabel("tekton.dev/task", "clustertask-full"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("clustertask-full", tb.TaskRefKind(v1alpha1.ClusterTaskKind)),
				tb.TaskRunServiceAccountName("svc"),
				tb.TaskRunParam("myarg", "value"),
				tb.TaskRunParam("print", "booms", "booms", "booms"),
				tb.TaskRunResources(tb.TaskRunResourcesInput("my-repo", tb.TaskResourceBindingRef("git"))),
				tb.TaskRunResources(tb.TaskRunResourcesOutput("my-image", tb.TaskResourceBindingRef("image"))),
			),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(-12*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionUnknown,
					Reason: resources.ReasonRunning,
				}),
			),
		),
		tb.TaskRun("taskrun-4", "ns",
			tb.TaskRunLabel("tekton.dev/task", "clustertask-full"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("clustertask-full", tb.TaskRefKind(v1alpha1.NamespacedTaskKind)),
				tb.TaskRunServiceAccountName("svc"),
				tb.TaskRunParam("myarg", "value"),
				tb.TaskRunParam("print", "booms", "booms", "booms"),
				tb.TaskRunResources(tb.TaskRunResourcesInput("my-repo", tb.TaskResourceBindingRef("git"))),
				tb.TaskRunResources(tb.TaskRunResourcesOutput("my-image", tb.TaskResourceBindingRef("image"))),
			),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(-12*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionUnknown,
					Reason: resources.ReasonRunning,
				}),
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

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredCT(clustertasks[0], version),
		cb.UnstructuredCT(clustertasks[1], version),
		cb.UnstructuredCT(clustertasks[2], version),
		cb.UnstructuredTR(taskruns[0], version),
		cb.UnstructuredTR(taskruns[1], version),
		cb.UnstructuredTR(taskruns[2], version),
		cb.UnstructuredTR(taskruns[3], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{ClusterTasks: clustertasks, Namespaces: ns, TaskRuns: taskruns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	p.SetNamespace("ns")

	tdc2 := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{{
			Verb:     "list",
			Resource: "taskruns",
			Action: func(action k8stest.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("fake list taskrun error")
			}}}}
	dynamic2, err := tdc2.Client(
		cb.UnstructuredCT(clustertasks[0], version),
		cb.UnstructuredCT(clustertasks[1], version),
		cb.UnstructuredCT(clustertasks[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs2, _ := test.SeedTestData(t, pipelinetest.Data{ClusterTasks: clustertasks, Namespaces: ns})
	cs2.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask", "taskrun"})
	p2 := &test.Params{Tekton: cs2.Pipeline, Kube: cs2.Kube, Dynamic: dynamic2, Clock: clock}
	p2.SetNamespace("ns")

	testParams := []struct {
		name        string
		command     []string
		param       *test.Params
		inputStream io.Reader
		wantError   bool
	}{
		{
			name:        "Describe with no arguments",
			command:     []string{"describe", "notexist"},
			param:       p,
			inputStream: nil,
			wantError:   true,
		},
		{
			name:        "Describe clustertask with name only",
			command:     []string{"describe", "clustertask-justname"},
			param:       p,
			inputStream: nil,
			wantError:   false,
		},
		{
			name:        "Describe full clustertask multiple taskruns",
			command:     []string{"describe", "clustertask-full"},
			param:       p,
			inputStream: nil,
			wantError:   false,
		},
		{
			name:        "Describe clustertask missing param default one of everything",
			command:     []string{"describe", "clustertask-one-everything"},
			param:       p,
			inputStream: nil,
			wantError:   false,
		},
		{
			name:        "Failure from listing taskruns",
			command:     []string{"describe", "clustertask-full"},
			param:       p2,
			inputStream: nil,
			wantError:   true,
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			clustertask := Command(tp.param)

			if tp.inputStream != nil {
				clustertask.SetIn(tp.inputStream)
			}

			got, err := test.ExecuteCommand(clustertask, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("Error expected here")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error")
				}
				golden.Assert(t, got, strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
			}
		})
	}
}

func TestClusterTask_custom_output(t *testing.T) {
	name := "clustertask"
	expected := "clustertask.tekton.dev/" + name

	clock := clockwork.NewFakeClock()

	cstasks := []*v1alpha1.ClusterTask{
		tb.ClusterTask(name),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		ClusterTasks: cstasks,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "",
				},
			},
		},
	})

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredCT(cstasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	clustertask := Command(p)
	got, err := test.ExecuteCommand(clustertask, "desc", "-o", "name", name)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	got = strings.TrimSpace(got)
	if got != expected {
		t.Errorf("Result should be '%s' != '%s'", got, expected)
	}
}

func TestClusterTaskV1beta1_custom_output(t *testing.T) {
	name := "clustertask"
	expected := "clustertask.tekton.dev/" + name

	clock := clockwork.NewFakeClock()

	cstasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		ClusterTasks: cstasks,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "",
				},
			},
		},
	})

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CT(cstasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	clustertask := Command(p)
	got, err := test.ExecuteCommand(clustertask, "desc", "-o", "name", name)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	got = strings.TrimSpace(got)
	if got != expected {
		t.Errorf("Result should be '%s' != '%s'", got, expected)
	}
}
