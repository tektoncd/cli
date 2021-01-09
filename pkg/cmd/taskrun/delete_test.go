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

	tb "github.com/tektoncd/cli/internal/builder/v1alpha1"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
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

	tasks := []*v1alpha1.Task{
		tb.Task("random",
			tb.TaskNamespace("ns"),
		),
	}

	clustertasks := []*v1alpha1.ClusterTask{
		tb.ClusterTask("random"),
	}

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr0-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("random", tb.TaskRefKind(v1alpha1.NamespacedTaskKind)),
			),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.TaskRunReasonSuccessful.String(),
				}),
			),
		),
		tb.TaskRun("tr0-2",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("random", tb.TaskRefKind(v1alpha1.NamespacedTaskKind)),
			),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.TaskRunReasonSuccessful.String(),
				}),
			),
		),
		tb.TaskRun("tr0-3",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("random", tb.TaskRefKind(v1alpha1.NamespacedTaskKind)),
			),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.TaskRunReasonSuccessful.String(),
				}),
			),
		),
		tb.TaskRun("tr0-4",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/clusterTask", "random"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("random", tb.TaskRefKind(v1alpha1.ClusterTaskKind)),
			),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.TaskRunReasonSuccessful.String(),
				}),
			),
		),
		tb.TaskRun("tr0-5",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/clusterTask", "random"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("random", tb.TaskRefKind(v1alpha1.ClusterTaskKind)),
			),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.TaskRunReasonSuccessful.String(),
				}),
			),
		),
		tb.TaskRun("tr0-6",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/clusterTask", "random"),
			tb.TaskRunSpec(
				tb.TaskRunTaskRef("random", tb.TaskRefKind(v1alpha1.ClusterTaskKind)),
			),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.TaskRunReasonSuccessful.String(),
				}),
			),
		),
	}
	type clients struct {
		pipelineClient pipelinetest.Clients
		dynamicClient  dynamic.Interface
	}

	seeds := make([]clients, 0)
	for i := 0; i < 6; i++ {
		cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Tasks: tasks, ClusterTasks: clustertasks, Namespaces: ns})
		cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredT(tasks[0], version),
			cb.UnstructuredCT(clustertasks[0], version),
			cb.UnstructuredTR(trs[0], version),
			cb.UnstructuredTR(trs[1], version),
			cb.UnstructuredTR(trs[2], version),
			cb.UnstructuredTR(trs[3], version),
			cb.UnstructuredTR(trs[4], version),
			cb.UnstructuredTR(trs[5], version),
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
			name:        "Invalid namespace",
			command:     []string{"rm", "tr0-1", "-n", "invalid"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete TaskRun \"tr0-1\": taskruns.tekton.dev \"tr0-1\" not found",
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
			want:        "canceled deleting TaskRun(s) \"tr0-1\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "tr0-1", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete TaskRun(s) \"tr0-1\" (y/n): TaskRuns deleted: \"tr0-1\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete TaskRun \"nonexistent\": taskruns.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete TaskRun \"nonexistent\": taskruns.tekton.dev \"nonexistent\" not found; failed to delete TaskRun \"nonexistent2\": taskruns.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Attempt remove forgetting to include taskrun names",
			command:     []string{"rm", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "must provide TaskRun name(s) or use --task flag or --all flag to use delete",
		},
		{
			name:        "Remove taskruns of a task",
			command:     []string{"rm", "--task", "random", "-n", "ns"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns related to Task \"random\" (y/n): All TaskRuns associated with Task \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns in namespace \"ns\" (y/n): All TaskRuns deleted in namespace \"ns\"\n",
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
			command:     []string{"delete", "-f", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 2 TaskRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all keeping 1 with --all flag",
			command:     []string{"delete", "-f", "--all", "--keep", "1", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 1 TaskRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Keep -1 is a no go",
			command:     []string{"delete", "-f", "--keep", "-1", "-n", "ns"},
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
		{
			name:        "Error from deleting TaskRun with non-existing Task",
			command:     []string{"delete", "taskrun", "-t", "non-existing-task"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "no TaskRuns associated with Task \"non-existing-task\"",
		},
		{
			name:        "Remove taskruns of a task with --keep",
			command:     []string{"rm", "--task", "random", "-n", "ns", "--keep", "2"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns related to Task \"random\" keeping 2 TaskRuns (y/n): All but 2 TaskRuns associated with Task \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from using argument with --keep",
			command:     []string{"rm", "taskrun", "--keep", "2"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--keep flag should not have any arguments specified with it",
		},
		{
			name:        "Error when using both --task and --clustertask flag together",
			command:     []string{"rm", "--task", "random", "--clustertask", "random"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "cannot use --task and --clustertask option together",
		},
		{
			name:        "Remove taskruns of a clustertask",
			command:     []string{"rm", "--clustertask", "random", "-n", "ns"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns related to ClusterTask \"random\" (y/n): All TaskRuns associated with ClusterTask \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from deleting TaskRun with non-existing ClusterTask",
			command:     []string{"delete", "taskrun", "--clustertask", "non-existing-clustertask"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "no TaskRuns associated with ClusterTask \"non-existing-clustertask\"",
		},
		{
			name:        "Remove taskruns of a ClusterTask with --keep",
			command:     []string{"rm", "--clustertask", "random", "-n", "ns", "--keep", "1"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns related to ClusterTask \"random\" keeping 1 TaskRuns (y/n): All but 1 TaskRuns associated with ClusterTask \"random\" deleted in namespace \"ns\"\n",
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

	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "random",
				Namespace: "ns",
			},
		},
	}

	clustertasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random",
			},
		},
	}

	trdata := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-1",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-2",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-3",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-4",
				Labels:    map[string]string{"tekton.dev/clusterTask": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-5",
				Labels:    map[string]string{"tekton.dev/clusterTask": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-6",
				Labels:    map[string]string{"tekton.dev/clusterTask": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	type clients struct {
		pipelineClient pipelinev1beta1test.Clients
		dynamicClient  dynamic.Interface
	}

	seeds := make([]clients, 0)
	for i := 0; i < 6; i++ {
		trs := trdata
		cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Tasks: tasks, ClusterTasks: clustertasks, Namespaces: ns})
		cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredV1beta1T(tasks[0], version),
			cb.UnstructuredV1beta1CT(clustertasks[0], version),
			cb.UnstructuredV1beta1TR(trdata[0], version),
			cb.UnstructuredV1beta1TR(trdata[1], version),
			cb.UnstructuredV1beta1TR(trdata[2], version),
			cb.UnstructuredV1beta1TR(trdata[3], version),
			cb.UnstructuredV1beta1TR(trdata[4], version),
			cb.UnstructuredV1beta1TR(trdata[5], version),
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
			name:        "Invalid namespace",
			command:     []string{"rm", "tr0-1", "-n", "invalid"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete TaskRun \"tr0-1\": taskruns.tekton.dev \"tr0-1\" not found",
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
			want:        "canceled deleting TaskRun(s) \"tr0-1\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "tr0-1", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete TaskRun(s) \"tr0-1\" (y/n): TaskRuns deleted: \"tr0-1\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete TaskRun \"nonexistent\": taskruns.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete TaskRun \"nonexistent\": taskruns.tekton.dev \"nonexistent\" not found; failed to delete TaskRun \"nonexistent2\": taskruns.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Attempt remove forgetting to include taskrun names",
			command:     []string{"rm", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "must provide TaskRun name(s) or use --task flag or --all flag to use delete",
		},
		{
			name:        "Remove taskruns of a task",
			command:     []string{"rm", "--task", "random", "-n", "ns"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns related to Task \"random\" (y/n): All TaskRuns associated with Task \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns in namespace \"ns\" (y/n): All TaskRuns deleted in namespace \"ns\"\n",
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
		{
			name:        "Error from deleting TaskRun with non-existing Task",
			command:     []string{"delete", "taskrun", "-t", "non-existing-task"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "no TaskRuns associated with Task \"non-existing-task\"",
		},
		{
			name:        "Remove taskruns of a task with --keep",
			command:     []string{"rm", "--task", "random", "-n", "ns", "--keep", "2"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns related to Task \"random\" keeping 2 TaskRuns (y/n): All but 2 TaskRuns associated with Task \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from using argument with --keep",
			command:     []string{"rm", "taskrun", "--keep", "2"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--keep flag should not have any arguments specified with it",
		},
		{
			name:        "Remove taskruns of a ClusterTask",
			command:     []string{"rm", "--clustertask", "random", "-n", "ns"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns related to ClusterTask \"random\" (y/n): All TaskRuns associated with ClusterTask \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from deleting TaskRun with non-existing ClusterTask",
			command:     []string{"delete", "taskrun", "--clustertask", "non-existing-clustertask"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "no TaskRuns associated with ClusterTask \"non-existing-clustertask\"",
		},
		{
			name:        "Remove taskruns of a ClusterTask with --keep",
			command:     []string{"rm", "--clustertask", "random", "-n", "ns", "--keep", "2"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns related to ClusterTask \"random\" keeping 2 TaskRuns (y/n): All but 2 TaskRuns associated with ClusterTask \"random\" deleted in namespace \"ns\"\n",
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

func Test_ClusterTask_TaskRuns_Not_Deleted_With_Task_Option(t *testing.T) {
	version := "v1beta1"
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "random",
				Namespace: "ns",
			},
		},
	}

	clustertasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random",
			},
		},
	}

	trdata := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-1",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-2",
				Labels: map[string]string{
					// TaskRun associated with ClusterTask will
					// have both task and clusterTask labels:
					// https://github.com/tektoncd/pipeline/issues/2533
					"tekton.dev/clusterTask": "random",
					"tekton.dev/task":        "random",
				},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	trs := trdata
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1CT(clustertasks[0], version),
		cb.UnstructuredV1beta1TR(trdata[0], version),
		cb.UnstructuredV1beta1TR(trdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Dynamic: dc}
	taskrun := Command(p)
	// Run tkn taskrun delete --task random to delete on kind Task random's TaskRuns
	_, err = test.ExecuteCommand(taskrun, "delete", "--task", "random", "-n", "ns")
	if err != nil {
		t.Errorf("no error expected but received error: %v", err)
	}

	// Expect TaskRun from kind ClusterTask random remains after deletion
	expected := `NAME    STARTED   DURATION   STATUS
tr0-2   ---       ---        Succeeded
`
	// Run list command to confirm TaskRun still present
	out, err := test.ExecuteCommand(taskrun, "list", "-n", "ns")
	if err != nil {
		t.Errorf("no error expected but received error: %v", err)
	}
	test.AssertOutput(t, expected, out)
}

func Test_Task_TaskRuns_Not_Deleted_With_ClusterTask_Option(t *testing.T) {
	version := "v1beta1"
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	tasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "random",
				Namespace: "ns",
			},
		},
	}

	clustertasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random",
			},
		},
	}

	trdata := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-1",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-2",
				Labels: map[string]string{
					// TaskRun associated with ClusterTask will
					// have both task and clusterTask labels:
					// https://github.com/tektoncd/pipeline/issues/2533
					"tekton.dev/clusterTask": "random",
					"tekton.dev/task":        "random",
				},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	trs := trdata
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], version),
		cb.UnstructuredV1beta1CT(clustertasks[0], version),
		cb.UnstructuredV1beta1TR(trdata[0], version),
		cb.UnstructuredV1beta1TR(trdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Dynamic: dc}
	taskrun := Command(p)
	// Run tkn taskrun delete --task random to delete on kind Task random's TaskRuns
	_, err = test.ExecuteCommand(taskrun, "delete", "--clustertask", "random", "-n", "ns")
	if err != nil {
		t.Errorf("no error expected but received error: %v", err)
	}

	// Expect TaskRun from kind ClusterTask random remains after deletion
	expected := `NAME    STARTED   DURATION   STATUS
tr0-1   ---       ---        Succeeded
`
	// Run list command to confirm TaskRun still present
	out, err := test.ExecuteCommand(taskrun, "list", "-n", "ns")
	if err != nil {
		t.Errorf("no error expected but received error: %v", err)
	}
	test.AssertOutput(t, expected, out)
}
