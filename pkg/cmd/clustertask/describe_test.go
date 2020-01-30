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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinetest "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stest "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
)

func Test_ClusterTaskDescribe(t *testing.T) {
	clock := clockwork.NewFakeClock()
	seeds := make([]pipelinetest.Clients, 0)

	clustertasks := []*v1alpha1.ClusterTask{
		tb.ClusterTask("clustertask-full", cb.ClusterTaskCreationTime(clock.Now().Add(1*time.Minute)),
			tb.ClusterTaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsResource("my-image", v1alpha1.PipelineResourceTypeImage),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString, tb.ParamSpecDefault("default")),
					tb.InputsParamSpec("print", v1alpha1.ParamTypeArray, tb.ParamSpecDefault("booms", "booms", "booms")),
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
		tb.ClusterTask("clustertask-one-everything", cb.ClusterTaskCreationTime(clock.Now().Add(1*time.Minute)),
			tb.ClusterTaskSpec(
				tb.TaskInputs(
					tb.InputsResource("my-repo", v1alpha1.PipelineResourceTypeGit),
					tb.InputsParamSpec("myarg", v1alpha1.ParamTypeString),
				),
				tb.TaskOutputs(
					tb.OutputsResource("code-image", v1alpha1.PipelineResourceTypeImage),
				),
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
				tb.TaskRunTaskRef("clustertask-1", tb.TaskRefKind(v1alpha1.ClusterTaskKind)),
				tb.TaskRunServiceAccountName("svc"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("myarg", "value")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("print", "booms", "booms", "booms")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("my-repo", tb.TaskResourceBindingRef("git"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("my-image", tb.TaskResourceBindingRef("image"))),
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
				tb.TaskRunTaskRef("clustertask-1", tb.TaskRefKind(v1alpha1.ClusterTaskKind)),
				tb.TaskRunServiceAccountName("svc"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("myarg", "value")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("my-repo", tb.TaskResourceBindingRef("git"))),
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
				tb.TaskRunTaskRef("clustertask-1", tb.TaskRefKind(v1alpha1.ClusterTaskKind)),
				tb.TaskRunServiceAccountName("svc"),
				tb.TaskRunInputs(tb.TaskRunInputsParam("myarg", "value")),
				tb.TaskRunInputs(tb.TaskRunInputsParam("print", "booms", "booms", "booms")),
				tb.TaskRunInputs(tb.TaskRunInputsResource("my-repo", tb.TaskResourceBindingRef("git"))),
				tb.TaskRunOutputs(tb.TaskRunOutputsResource("my-image", tb.TaskResourceBindingRef("image"))),
			),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(-10*time.Minute)),
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

	cs, _ := test.SeedTestData(t, pipelinetest.Data{ClusterTasks: clustertasks, Namespaces: ns, TaskRuns: taskruns})
	seeds = append(seeds, cs)

	cs2, _ := test.SeedTestData(t, pipelinetest.Data{ClusterTasks: clustertasks, Namespaces: ns})
	cs2.Pipeline.PrependReactor("list", "taskruns", func(action k8stest.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("fake list taskrun error")
	})

	seeds = append(seeds, cs2)

	testParams := []struct {
		name        string
		command     []string
		input       pipelinetest.Clients
		inputStream io.Reader
		wantError   bool
	}{
		{
			name:        "Describe with no arguments",
			command:     []string{"describe", "notexist"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
		},
		{
			name:        "Describe clustertask with name only",
			command:     []string{"describe", "clustertask-justname"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
		},
		{
			name:        "Describe full clustertask multiple taskruns",
			command:     []string{"describe", "clustertask-full"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
		},
		{
			name:        "Describe clustertask missing param default one of everything",
			command:     []string{"describe", "clustertask-one-everything"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
		},
		{
			name:        "Failure from listing taskruns",
			command:     []string{"describe", "clustertask-full"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   true,
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

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube}
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
