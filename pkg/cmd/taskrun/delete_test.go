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

package taskrun

import (
	"io"
	"strings"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"knative.dev/pkg/apis"
)

func TestTaskRunDelete(t *testing.T) {
	version := "v1alpha1"
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	trdata := []*v1alpha1.TaskRun{
		tb.TaskRun("tr0-1", "ns",
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
		tb.TaskRun("tr0-2", "ns",
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
		tb.TaskRun("tr0-3", "ns",
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
	}
	type clients struct {
		pipelineClient pipelinetest.Clients
		dynamicClient  dynamic.Interface
	}

	seeds := make([]clients, 0)
	for i := 0; i < 5; i++ {
		trs := trdata
		cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Namespaces: ns})
		cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
		dc, err := testDynamic.Client(
			cb.UnstructuredTR(trdata[0], version),
			cb.UnstructuredTR(trdata[1], version),
			cb.UnstructuredTR(trdata[2], version),
		)
		if err != nil {
			t.Errorf("unable to create dynamic clinet: %v", err)
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
			name:        "Invalid namespace",
			command:     []string{"rm", "tr0-1", "-n", "invalid"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "namespaces \"invalid\" not found",
		},
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "tr0-1", "-n", "ns", "-f"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "TaskRuns deleted: \"tr0-1\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "tr0-1", "-n", "ns", "--force"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "TaskRuns deleted: \"tr0-1\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "tr0-1", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting taskrun \"tr0-1\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "tr0-1", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete taskrun \"tr0-1\" (y/n): TaskRuns deleted: \"tr0-1\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete taskrun \"nonexistent\": taskruns.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete taskrun \"nonexistent\": taskruns.tekton.dev \"nonexistent\" not found; failed to delete taskrun \"nonexistent2\": taskruns.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Attempt remove forgetting to include taskrun names",
			command:     []string{"rm", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "must provide taskrun name(s) or use --task flag or --all flag to use delete",
		},
		{
			name:        "Remove taskruns of a task",
			command:     []string{"rm", "--task", "task", "-n", "ns"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        `Are you sure you want to delete all taskruns related to task "task" (y/n): `,
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all taskruns in namespace \"ns\" (y/n): All TaskRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All TaskRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all keeping 2",
			command:     []string{"delete", "--all", "-f", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 2 TaskRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Keep -1 is a no go",
			command:     []string{"delete", "--all", "-f", "--keep", "-1", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "keep option should not be lower than 0",
		},
		{
			name:        "Error from using taskrun name with --all",
			command:     []string{"delete", "taskrun", "--all", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube, Dynamic: tp.dynamic}
			taskrun := Command(p)

			if tp.inputStream != nil {
				taskrun.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(taskrun, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				} else {
					test.AssertOutput(t, tp.want, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}

func TestTaskRunDelete_v1beta1(t *testing.T) {
	version := "v1beta1"
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	trdata := []*v1alpha1.TaskRun{
		tb.TaskRun("tr0-1", "ns",
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
		tb.TaskRun("tr0-2", "ns",
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
		tb.TaskRun("tr0-3", "ns",
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
	}
	type clients struct {
		pipelineClient pipelinetest.Clients
		dynamicClient  dynamic.Interface
	}

	seeds := make([]clients, 0)
	for i := 0; i < 5; i++ {
		trs := trdata
		cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Namespaces: ns})
		cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
		dc, err := testDynamic.Client(
			cb.UnstructuredTR(trdata[0], version),
			cb.UnstructuredTR(trdata[1], version),
			cb.UnstructuredTR(trdata[2], version),
		)
		if err != nil {
			t.Errorf("unable to create dynamic clinet: %v", err)
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
			name:        "Invalid namespace",
			command:     []string{"rm", "tr0-1", "-n", "invalid"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "namespaces \"invalid\" not found",
		},
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "tr0-1", "-n", "ns", "-f"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "TaskRuns deleted: \"tr0-1\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "tr0-1", "-n", "ns", "--force"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "TaskRuns deleted: \"tr0-1\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "tr0-1", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting taskrun \"tr0-1\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "tr0-1", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete taskrun \"tr0-1\" (y/n): TaskRuns deleted: \"tr0-1\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete taskrun \"nonexistent\": taskruns.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete taskrun \"nonexistent\": taskruns.tekton.dev \"nonexistent\" not found; failed to delete taskrun \"nonexistent2\": taskruns.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Attempt remove forgetting to include taskrun names",
			command:     []string{"rm", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "must provide taskrun name(s) or use --task flag or --all flag to use delete",
		},
		{
			name:        "Remove taskruns of a task",
			command:     []string{"rm", "--task", "task", "-n", "ns"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        `Are you sure you want to delete all taskruns related to task "task" (y/n): `,
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all taskruns in namespace \"ns\" (y/n): All TaskRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All TaskRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all keeping 2",
			command:     []string{"delete", "--all", "-f", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 2 TaskRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Keep -1 is a no go",
			command:     []string{"delete", "--all", "-f", "--keep", "-1", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "keep option should not be lower than 0",
		},
		{
			name:        "Error from using taskrun name with --all",
			command:     []string{"delete", "taskrun", "--all", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube, Dynamic: tp.dynamic}
			taskrun := Command(p)

			if tp.inputStream != nil {
				taskrun.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(taskrun, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				} else {
					test.AssertOutput(t, tp.want, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}
