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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinetest "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestTaskDelete(t *testing.T) {
	clock := clockwork.NewFakeClock()

	seeds := make([]pipelinetest.Clients, 0)
	for i := 0; i < 8; i++ {
		cs, _ := test.SeedTestData(t, pipelinetest.Data{
			Tasks: []*v1alpha1.Task{
				tb.Task("task", "ns", cb.TaskCreationTime(clock.Now().Add(-1*time.Minute))),
				tb.Task("task2", "ns", cb.TaskCreationTime(clock.Now().Add(-1*time.Minute))),
			},
			TaskRuns: []*v1alpha1.TaskRun{
				tb.TaskRun("task-run-1", "ns",
					tb.TaskRunLabel("tekton.dev/task", "task"),
					tb.TaskRunSpec(tb.TaskRunTaskRef("task")),
					tb.TaskRunStatus(
						tb.StatusCondition(apis.Condition{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						}),
					),
				),
				tb.TaskRun("task-run-2", "ns",
					tb.TaskRunLabel("tekton.dev/task", "task"),
					tb.TaskRunSpec(tb.TaskRunTaskRef("task")),
					tb.TaskRunStatus(
						tb.StatusCondition(apis.Condition{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						}),
					),
				),
			},
			Namespaces: []*corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns",
					},
				},
			},
		})
		seeds = append(seeds, cs)
	}

	testParams := []struct {
		name        string
		command     []string
		input       pipelinetest.Clients
		inputStream io.Reader
		wantError   bool
		want        string
	}{
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "task", "-n", "ns", "-f"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "Tasks deleted: \"task\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "task", "-n", "ns", "--force"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   false,
			want:        "Tasks deleted: \"task\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "task", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting task \"task\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "task", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete task \"task\" (y/n): Tasks deleted: \"task\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   true,
			want:        "failed to delete task \"nonexistent\": tasks.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "With delete all flag, reply yes",
			command:     []string{"rm", "task", "-n", "ns", "--trs"},
			input:       seeds[3],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete task and related resources \"task\" (y/n): TaskRuns deleted: \"task-run-1\", \"task-run-2\"\nTasks deleted: \"task\"\n",
		},
		{
			name:        "With delete all and force delete flag",
			command:     []string{"rm", "task", "-n", "ns", "-f", "--trs"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   false,
			want:        "TaskRuns deleted: \"task-run-1\", \"task-run-2\"\nTasks deleted: \"task\"\n",
		},
		{
			name:        "Try to delete task from invalid namespace",
			command:     []string{"rm", "task", "-n", "invalid", "-f"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   true,
			want:        "namespaces \"invalid\" not found",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			input:       seeds[5],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all tasks in namespace \"ns\" (y/n): All Tasks deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			input:       seeds[6],
			inputStream: nil,
			wantError:   false,
			want:        "All Tasks deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from using task name with --all",
			command:     []string{"delete", "task", "--all", "-n", "ns"},
			input:       seeds[6],
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from using --all with --trs",
			command:     []string{"delete", "--all", "--trs", "-n", "ns"},
			input:       seeds[6],
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "With force delete flag (shorthand), multiple pipelines",
			command:     []string{"rm", "task", "task2", "-n", "ns", "-f"},
			input:       seeds[7],
			inputStream: nil,
			wantError:   false,
			want:        "Tasks deleted: \"task\", \"task2\"\n",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube}
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
