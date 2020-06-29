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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestTaskDelete(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()

	type clients struct {
		pipelineClient pipelinetest.Clients
		dynamicClient  dynamic.Interface
	}

	tdata := []*v1alpha1.Task{
		tb.Task("task", tb.TaskNamespace("ns"), cb.TaskCreationTime(clock.Now().Add(-1*time.Minute))),
		tb.Task("task2", tb.TaskNamespace("ns"), cb.TaskCreationTime(clock.Now().Add(-1*time.Minute))),
	}

	trdata := []*v1alpha1.TaskRun{
		tb.TaskRun("task-run-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task", tb.TaskRefKind(v1alpha1.NamespacedTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
		tb.TaskRun("task-run-2",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task", tb.TaskRefKind(v1alpha1.NamespacedTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
		// ClusterTask is provided in the TaskRef of TaskRun, so as to verify
		// TaskRun created by ClusterTask is not getting deleted while deleting
		// Task with `--trs` flag and name of Task and ClusterTask is same.
		tb.TaskRun("task-run-3",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task", tb.TaskRefKind(v1alpha1.ClusterTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
	}

	seeds := make([]clients, 0)
	for i := 0; i < 8; i++ {

		cs, _ := test.SeedTestData(t, pipelinetest.Data{
			Tasks:    tdata,
			TaskRuns: trdata,
			Namespaces: []*corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns",
					},
				},
			},
		})

		cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredT(tdata[0], version),
			cb.UnstructuredT(tdata[1], version),
			cb.UnstructuredTR(trdata[0], version),
			cb.UnstructuredTR(trdata[1], version),
			cb.UnstructuredTR(trdata[2], version),
		)
		if err != nil {
			t.Errorf("unable to create dynamic client: %v", err)
		}

		seeds = append(seeds, clients{cs, dc})
	}

	testParams := []struct {
		name        string
		command     []string
		dynamic     dynamic.Interface
		input       pipelinetest.Clients
		inputStream io.Reader
		wantError   bool
		want        string
	}{
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "task", "-n", "ns", "-f"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Tasks deleted: \"task\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "task", "-n", "ns", "--force"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Tasks deleted: \"task\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "task", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting Task(s) \"task\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "task", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete Task(s) \"task\" (y/n): Tasks deleted: \"task\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete task \"nonexistent\": tasks.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete task \"nonexistent\": tasks.tekton.dev \"nonexistent\" not found; failed to delete task \"nonexistent2\": tasks.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Remove multiple non existent resources with --trs flag",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns", "--trs"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete task \"nonexistent\": tasks.tekton.dev \"nonexistent\" not found; failed to delete task \"nonexistent2\": tasks.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "With delete taskruns flag, reply yes",
			command:     []string{"rm", "task", "-n", "ns", "--trs"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete Task(s) \"task\" and related resources (y/n): TaskRuns deleted: \"task-run-1\", \"task-run-2\"\nTasks deleted: \"task\"\n",
		},
		{
			name:        "With --trs and force delete flag",
			command:     []string{"rm", "task", "-n", "ns", "-f", "--trs"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "TaskRuns deleted: \"task-run-1\", \"task-run-2\"\nTasks deleted: \"task\"\n",
		},
		{
			name:        "Try to delete task from invalid namespace",
			command:     []string{"rm", "task", "-n", "invalid", "-f"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "namespaces \"invalid\" not found",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all Tasks in namespace \"ns\" (y/n): All Tasks deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All Tasks deleted in namespace \"ns\"\n",
		},
		{
			name:    "Error from using task name with --all",
			command: []string{"delete", "task", "--all", "-n", "ns"},
			dynamic: seeds[6].dynamicClient,
			input:   seeds[6].pipelineClient,

			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from using --all with --trs",
			command:     []string{"delete", "--all", "--trs", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "With force delete flag (shorthand), multiple pipelines",
			command:     []string{"rm", "task", "task2", "-n", "ns", "-f"},
			dynamic:     seeds[7].dynamicClient,
			input:       seeds[7].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Tasks deleted: \"task\", \"task2\"\n",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube, Dynamic: tp.dynamic}
			task := Command(p)

			if tp.inputStream != nil {
				task.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(task, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}

func TestTaskDelete_v1beta1(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	taskCreated := clock.Now().Add(-1 * time.Minute)

	tdata := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "task",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: taskCreated},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "task2",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: taskCreated},
			},
		},
	}

	type clients struct {
		pipelineClient pipelinev1beta1test.Clients
		dynamicClient  dynamic.Interface
	}

	trdata := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-run-1",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-run-2",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
			},
		},
		// ClusterTask is provided in the TaskRef of TaskRun, so as to verify
		// TaskRun created by ClusterTask is not getting deleted while deleting
		// Task with `--trs` flag and name of Task and ClusterTask is same.
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-run-3",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
			},
		},
	}

	seeds := make([]clients, 0)
	for i := 0; i < 8; i++ {

		cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
			Tasks:    tdata,
			TaskRuns: trdata,
			Namespaces: []*corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns",
					},
				},
			},
		})

		cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredV1beta1T(tdata[0], version),
			cb.UnstructuredV1beta1T(tdata[1], version),
			cb.UnstructuredV1beta1TR(trdata[0], version),
			cb.UnstructuredV1beta1TR(trdata[1], version),
			cb.UnstructuredV1beta1TR(trdata[2], version),
		)
		if err != nil {
			t.Errorf("unable to create dynamic client: %v", err)
		}

		seeds = append(seeds, clients{cs, dc})
	}

	testParams := []struct {
		name        string
		command     []string
		dynamic     dynamic.Interface
		input       pipelinev1beta1test.Clients
		inputStream io.Reader
		wantError   bool
		want        string
	}{
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "task", "-n", "ns", "-f"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Tasks deleted: \"task\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "task", "-n", "ns", "--force"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Tasks deleted: \"task\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "task", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting Task(s) \"task\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "task", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete Task(s) \"task\" (y/n): Tasks deleted: \"task\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete task \"nonexistent\": tasks.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete task \"nonexistent\": tasks.tekton.dev \"nonexistent\" not found; failed to delete task \"nonexistent2\": tasks.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Remove multiple non existent resources with --trs flag",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns", "--trs"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete task \"nonexistent\": tasks.tekton.dev \"nonexistent\" not found; failed to delete task \"nonexistent2\": tasks.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "With delete taskruns flag, reply yes",
			command:     []string{"rm", "task", "-n", "ns", "--trs"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete Task(s) \"task\" and related resources (y/n): TaskRuns deleted: \"task-run-1\", \"task-run-2\"\nTasks deleted: \"task\"\n",
		},
		{
			name:        "With delete all and force delete flag",
			command:     []string{"rm", "task", "-n", "ns", "-f", "--trs"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "TaskRuns deleted: \"task-run-1\", \"task-run-2\"\nTasks deleted: \"task\"\n",
		},
		{
			name:        "Try to delete task from invalid namespace",
			command:     []string{"rm", "task", "-n", "invalid", "-f"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "namespaces \"invalid\" not found",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all Tasks in namespace \"ns\" (y/n): All Tasks deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All Tasks deleted in namespace \"ns\"\n",
		},
		{
			name:    "Error from using task name with --all",
			command: []string{"delete", "task", "--all", "-n", "ns"},
			dynamic: seeds[6].dynamicClient,
			input:   seeds[6].pipelineClient,

			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from using --all with --trs",
			command:     []string{"delete", "--all", "--trs", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "With force delete flag (shorthand), multiple pipelines",
			command:     []string{"rm", "task", "task2", "-n", "ns", "-f"},
			dynamic:     seeds[7].dynamicClient,
			input:       seeds[7].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Tasks deleted: \"task\", \"task2\"\n",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube, Dynamic: tp.dynamic}
			task := Command(p)

			if tp.inputStream != nil {
				task.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(task, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}
