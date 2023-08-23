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
	"time"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestTaskRunDelete_v1beta1(t *testing.T) {
	clock := test.FakeClock()

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
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now().Add(10 * time.Minute),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now().Add(10 * time.Minute),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now().Add(10 * time.Minute),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now().Add(10 * time.Minute),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now().Add(10 * time.Minute),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now().Add(10 * time.Minute),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
				Name:      "tr0-7",
				Labels:    map[string]string{"tekton.dev/clusterTask": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						// Use real time for testing keep-since
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
				Name:      "tr0-8",
				Labels:    map[string]string{"tekton.dev/clusterTask": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						// Use real time for testing keep-since and keep together
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
				Name:      "tr0-9",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						// for testing keep and keep-since together with resource name
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
				Name:      "tr0-10",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: nil,
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: v1beta1.TaskRunReasonStarted.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-11",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: nil,
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: v1beta1.TaskRunReasonRunning.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-12",
				Labels:    map[string]string{"tekton.dev/clusterTask": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: nil,
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: v1beta1.TaskRunReasonStarted.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-13",
				Labels:    map[string]string{"tekton.dev/clusterTask": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: nil,
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: v1beta1.TaskRunReasonRunning.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-14",
				Labels:    map[string]string{"tekton.dev/clusterTask": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: nil,
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: "ExceededResourceQuota",
						},
					},
				},
			},
		},
	}

	type clients struct {
		pipelineClient test.Clients
		dynamicClient  dynamic.Interface
	}

	seeds := make([]clients, 0)
	for i := 0; i < 17; i++ {
		trs := trdata
		cs, _ := test.SeedV1beta1TestData(t, test.Data{TaskRuns: trs, Tasks: tasks, ClusterTasks: clustertasks, Namespaces: ns})
		cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredV1beta1T(tasks[0], versionv1beta1),
			cb.UnstructuredV1beta1CT(clustertasks[0], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[0], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[1], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[2], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[3], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[4], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[5], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[6], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[7], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[8], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[9], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[10], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[11], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[12], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[13], versionv1beta1),
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
		input       test.Clients
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
			want:        "taskruns.tekton.dev \"tr0-1\" not found",
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
			want:        "taskruns.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "taskruns.tekton.dev \"nonexistent\" not found; taskruns.tekton.dev \"nonexistent2\" not found",
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
			want:        "Are you sure you want to delete all TaskRuns related to Task \"random\" (y/n): All 3 TaskRuns(Completed) associated with Task \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns in namespace \"ns\" (y/n): All 9 TaskRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 9 TaskRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all keeping 2",
			command:     []string{"delete", "--all", "-f", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 2 TaskRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all keeping 1 with --all flag",
			command:     []string{"delete", "-f", "--all", "--keep", "1", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 1 TaskRuns(Completed) deleted in namespace \"ns\"\n",
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
			command:     []string{"delete", "tr0-7", "--all", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from deleting TaskRun with non-existing Task",
			command:     []string{"delete", "tr0-7", "-t", "non-existing-task", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
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
			want:        "Are you sure you want to delete all TaskRuns related to Task \"random\" keeping 2 TaskRuns (y/n): All but 2 TaskRuns(Completed) associated with Task \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from using argument with --keep",
			command:     []string{"rm", "tr0-7", "-n", "ns", "--keep", "2"},
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
			name:        "Remove taskruns of a ClusterTask",
			command:     []string{"rm", "--clustertask", "random", "-n", "ns"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns related to ClusterTask \"random\" (y/n): All 5 TaskRuns(Completed) associated with ClusterTask \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from deleting TaskRun with non-existing ClusterTask",
			command:     []string{"delete", "tr0-7", "--clustertask", "non-existing-clustertask", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
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
			want:        "Are you sure you want to delete all TaskRuns related to ClusterTask \"random\" keeping 2 TaskRuns (y/n): All but 2 TaskRuns(Completed) associated with ClusterTask \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all Taskruns older than 60mn",
			command:     []string{"delete", "-f", "--all", "--keep-since", "60", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "6 expired Taskruns(Completed) has been deleted in namespace \"ns\", kept 3\n",
		},
		{
			name:        "Delete all Taskruns older than 60mn associated with random Task",
			command:     []string{"delete", "-f", "--task", "random", "--keep-since", "60", "-n", "ns"},
			dynamic:     seeds[7].dynamicClient,
			input:       seeds[7].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 3 expired TaskRuns associated with \"Task\" \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error --keep-since less than zero",
			command:     []string{"delete", "-f", "--task", "random", "--keep-since", "-1", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "since option should not be lower than 0",
		},
		{
			name:        "Error --keep-since, --all and --task cannot be used",
			command:     []string{"delete", "-f", "--task", "random", "--keep-since", "1", "--all", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--keep or --keep-since, --all and --task cannot be used together",
		},
		{
			name:        "Delete all TaskRuns older than 60mn and keeping 2 TaskRuns",
			command:     []string{"delete", "-f", "--all", "--keep-since", "60", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[9].dynamicClient,
			input:       seeds[9].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "7 TaskRuns(Completed) has been deleted in namespace \"ns\", kept 2\n",
		},
		{
			name:        "Delete all TaskRuns older than 60mn and keeping 2 TaskRuns associated with random Task",
			command:     []string{"delete", "-f", "--task", "random", "--keep-since", "60", "--keep", "1", "-n", "ns"},
			dynamic:     seeds[10].dynamicClient,
			input:       seeds[10].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "3 TaskRuns(Completed) associated with Task \"random\" has been deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error --keep-since and --keep less than zero",
			command:     []string{"delete", "-f", "--task", "random", "--keep-since", "-1", "--keep", "-1", "-n", "ns"},
			dynamic:     seeds[10].dynamicClient,
			input:       seeds[10].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "keep and keep-since option should not be lower than 0",
		},
		{
			name:        "Error --keep-since, --keep, --all and --task cannot be used",
			command:     []string{"delete", "-f", "--task", "random", "--keep-since", "1", "--keep", "1", "--all", "-n", "ns"},
			dynamic:     seeds[10].dynamicClient,
			input:       seeds[10].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--keep or --keep-since, --all and --task cannot be used together",
		},
		{
			name:        "Delete all with default --ignore-running",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 0 TaskRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with explicit --ignore-running true",
			command:     []string{"delete", "--all", "-f", "-n", "ns", "--ignore-running=true"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 0 TaskRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with --ignore-running false",
			command:     []string{"delete", "--all", "-f", "-n", "ns", "--ignore-running=false"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 5 TaskRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete the Task present and give error for non-existent Task",
			command:     []string{"delete", "nonexistent", "tr0-1", "-n", "ns"},
			dynamic:     seeds[8].dynamicClient,
			input:       seeds[8].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "taskruns.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Attempt to delete TaskRun by keeping equal to existing TaskRun for a Task",
			command:     []string{"delete", "-i", "-f", "--keep", "1", "--task", "random", "-n", "ns"},
			dynamic:     seeds[10].dynamicClient,
			input:       seeds[10].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Associated TaskRun (1) for Task:random is/are equal to keep (1) \n",
		},
		{
			name:        "Attempt to delete TaskRun by keeping more than existing TaskRun for a Task",
			command:     []string{"delete", "-i", "-f", "--keep", "2", "--task", "random", "-n", "ns"},
			dynamic:     seeds[10].dynamicClient,
			input:       seeds[10].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "There is/are only 1 TaskRun(s) associated for Task: random \n",
		},
		{
			name:        "Delete all of task with default --ignore-running",
			command:     []string{"delete", "-t", "random", "-f", "-n", "ns", "--ignore-running"},
			dynamic:     seeds[11].dynamicClient,
			input:       seeds[11].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 4 TaskRuns(Completed) associated with Task \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all of task with explicit --ignore-running true",
			command:     []string{"delete", "-t", "random", "-f", "-n", "ns", "--ignore-running=true"},
			dynamic:     seeds[12].dynamicClient,
			input:       seeds[12].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 4 TaskRuns(Completed) associated with Task \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all of task with --ignore-running false",
			command:     []string{"delete", "-t", "random", "-f", "-n", "ns", "--ignore-running=false"},
			dynamic:     seeds[13].dynamicClient,
			input:       seeds[13].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 6 TaskRuns associated with Task \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all of clustertask with default --ignore-running",
			command:     []string{"delete", "--clustertask", "random", "-f", "-n", "ns", "--ignore-running"},
			dynamic:     seeds[14].dynamicClient,
			input:       seeds[14].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 5 TaskRuns(Completed) associated with ClusterTask \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all of clustertask with explicit --ignore-running true",
			command:     []string{"delete", "--clustertask", "random", "-f", "-n", "ns", "--ignore-running=true"},
			dynamic:     seeds[15].dynamicClient,
			input:       seeds[15].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 5 TaskRuns(Completed) associated with ClusterTask \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all of clustertask with --ignore-running false",
			command:     []string{"delete", "--clustertask", "random", "-f", "-n", "ns", "--ignore-running=false"},
			dynamic:     seeds[16].dynamicClient,
			input:       seeds[16].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 8 TaskRuns associated with ClusterTask \"random\" deleted in namespace \"ns\"\n",
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

func TestTaskRunDelete(t *testing.T) {
	clock := test.FakeClock()

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "random",
				Namespace: "ns",
			},
		},
	}
	trdata := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-1",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now().Add(10 * time.Minute),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
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
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now().Add(10 * time.Minute),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
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
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now().Add(10 * time.Minute),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
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
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: "ClusterTask",
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now().Add(10 * time.Minute),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
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
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: "ClusterTask",
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now().Add(10 * time.Minute),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
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
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: "ClusterTask",
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now().Add(10 * time.Minute),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-7",
				Labels:    map[string]string{"tekton.dev/clusterTask": "random"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: "ClusterTaskKind",
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						// Use real time for testing keep-since
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-8",
				Labels:    map[string]string{"tekton.dev/clusterTask": "random"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: "ClusterTask",
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						// Use real time for testing keep-since and keep together
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr0-9",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						// for testing keep and keep-since together with resource name
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-10",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: nil,
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: v1.TaskRunReasonStarted.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-11",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: nil,
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: v1.TaskRunReasonRunning.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-12",
				Labels:    map[string]string{"tekton.dev/clusterTask": "random"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: "ClusterTask",
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: nil,
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: v1.TaskRunReasonStarted.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-13",
				Labels:    map[string]string{"tekton.dev/clusterTask": "random"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: "ClusterTask",
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: nil,
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: v1.TaskRunReasonRunning.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-14",
				Labels:    map[string]string{"tekton.dev/clusterTask": "random"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: "ClusterTask",
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: nil,
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: "ExceededResourceQuota",
						},
					},
				},
			},
		},
	}

	type clients struct {
		pipelineClient pipelinetest.Clients
		dynamicClient  dynamic.Interface
	}

	seeds := make([]clients, 0)
	for i := 0; i < 17; i++ {
		trs := trdata
		cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Tasks: tasks, Namespaces: ns})
		cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredT(tasks[0], version),
			cb.UnstructuredTR(trdata[0], version),
			cb.UnstructuredTR(trdata[1], version),
			cb.UnstructuredTR(trdata[2], version),
			cb.UnstructuredTR(trdata[3], version),
			cb.UnstructuredTR(trdata[4], version),
			cb.UnstructuredTR(trdata[5], version),
			cb.UnstructuredTR(trdata[6], version),
			cb.UnstructuredTR(trdata[7], version),
			cb.UnstructuredTR(trdata[8], version),
			cb.UnstructuredTR(trdata[9], version),
			cb.UnstructuredTR(trdata[10], version),
			cb.UnstructuredTR(trdata[11], version),
			cb.UnstructuredTR(trdata[12], version),
			cb.UnstructuredTR(trdata[13], version),
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
			want:        "taskruns.tekton.dev \"tr0-1\" not found",
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
			want:        "taskruns.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "taskruns.tekton.dev \"nonexistent\" not found; taskruns.tekton.dev \"nonexistent2\" not found",
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
			want:        "Are you sure you want to delete all TaskRuns related to Task \"random\" (y/n): All 3 TaskRuns(Completed) associated with Task \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns in namespace \"ns\" (y/n): All 9 TaskRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 9 TaskRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all keeping 2",
			command:     []string{"delete", "--all", "-f", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 2 TaskRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all keeping 1 with --all flag",
			command:     []string{"delete", "-f", "--all", "--keep", "1", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 1 TaskRuns(Completed) deleted in namespace \"ns\"\n",
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
			command:     []string{"delete", "tr0-7", "--all", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from deleting TaskRun with non-existing Task",
			command:     []string{"delete", "tr0-7", "-t", "non-existing-task", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
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
			want:        "Are you sure you want to delete all TaskRuns related to Task \"random\" keeping 2 TaskRuns (y/n): All but 2 TaskRuns(Completed) associated with Task \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from using argument with --keep",
			command:     []string{"rm", "tr0-7", "-n", "ns", "--keep", "2"},
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
			name:        "Remove taskruns of a ClusterTask",
			command:     []string{"rm", "--clustertask", "random", "-n", "ns"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns related to ClusterTask \"random\" (y/n): All 5 TaskRuns(Completed) associated with ClusterTask \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from deleting TaskRun with non-existing ClusterTask",
			command:     []string{"delete", "tr0-7", "--clustertask", "non-existing-clustertask", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
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
			want:        "Are you sure you want to delete all TaskRuns related to ClusterTask \"random\" keeping 2 TaskRuns (y/n): All but 2 TaskRuns(Completed) associated with ClusterTask \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all Taskruns older than 60mn",
			command:     []string{"delete", "-f", "--all", "--keep-since", "60", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "6 expired Taskruns(Completed) has been deleted in namespace \"ns\", kept 3\n",
		},
		{
			name:        "Delete all Taskruns older than 60mn associated with random Task",
			command:     []string{"delete", "-f", "--task", "random", "--keep-since", "60", "-n", "ns"},
			dynamic:     seeds[7].dynamicClient,
			input:       seeds[7].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 3 expired TaskRuns associated with \"Task\" \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error --keep-since less than zero",
			command:     []string{"delete", "-f", "--task", "random", "--keep-since", "-1", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "since option should not be lower than 0",
		},
		{
			name:        "Error --keep-since, --all and --task cannot be used",
			command:     []string{"delete", "-f", "--task", "random", "--keep-since", "1", "--all", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--keep or --keep-since, --all and --task cannot be used together",
		},
		{
			name:        "Delete all TaskRuns older than 60mn and keeping 2 TaskRuns",
			command:     []string{"delete", "-f", "--all", "--keep-since", "60", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[9].dynamicClient,
			input:       seeds[9].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "7 TaskRuns(Completed) has been deleted in namespace \"ns\", kept 2\n",
		},
		{
			name:        "Delete all TaskRuns older than 60mn and keeping 2 TaskRuns associated with random Task",
			command:     []string{"delete", "-f", "--task", "random", "--keep-since", "60", "--keep", "1", "-n", "ns"},
			dynamic:     seeds[10].dynamicClient,
			input:       seeds[10].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "3 TaskRuns(Completed) associated with Task \"random\" has been deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error --keep-since and --keep less than zero",
			command:     []string{"delete", "-f", "--task", "random", "--keep-since", "-1", "--keep", "-1", "-n", "ns"},
			dynamic:     seeds[10].dynamicClient,
			input:       seeds[10].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "keep and keep-since option should not be lower than 0",
		},
		{
			name:        "Error --keep-since, --keep, --all and --task cannot be used",
			command:     []string{"delete", "-f", "--task", "random", "--keep-since", "1", "--keep", "1", "--all", "-n", "ns"},
			dynamic:     seeds[10].dynamicClient,
			input:       seeds[10].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--keep or --keep-since, --all and --task cannot be used together",
		},
		{
			name:        "Delete all with default --ignore-running",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 0 TaskRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with explicit --ignore-running true",
			command:     []string{"delete", "--all", "-f", "-n", "ns", "--ignore-running=true"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 0 TaskRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with --ignore-running false",
			command:     []string{"delete", "--all", "-f", "-n", "ns", "--ignore-running=false"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 5 TaskRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete the Task present and give error for non-existent Task",
			command:     []string{"delete", "nonexistent", "tr0-1", "-n", "ns"},
			dynamic:     seeds[8].dynamicClient,
			input:       seeds[8].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "taskruns.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Attempt to delete TaskRun by keeping more than existing TaskRun for a Task",
			command:     []string{"delete", "-i", "-f", "--keep", "2", "--task", "random", "-n", "ns"},
			dynamic:     seeds[10].dynamicClient,
			input:       seeds[10].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "There is/are only 1 TaskRun(s) associated for Task: random \n",
		},
		{
			name:        "Attempt to delete TaskRun by keeping equal to existing TaskRun for a Task",
			command:     []string{"delete", "-i", "-f", "--keep", "1", "--task", "random", "-n", "ns"},
			dynamic:     seeds[10].dynamicClient,
			input:       seeds[10].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Associated TaskRun (1) for Task:random is/are equal to keep (1) \n",
		},
		{
			name:        "Delete all of task with default --ignore-running",
			command:     []string{"delete", "-t", "random", "-f", "-n", "ns", "--ignore-running"},
			dynamic:     seeds[11].dynamicClient,
			input:       seeds[11].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 4 TaskRuns(Completed) associated with Task \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all of task with explicit --ignore-running true",
			command:     []string{"delete", "-t", "random", "-f", "-n", "ns", "--ignore-running=true"},
			dynamic:     seeds[12].dynamicClient,
			input:       seeds[12].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 4 TaskRuns(Completed) associated with Task \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all of task with --ignore-running false",
			command:     []string{"delete", "-t", "random", "-f", "-n", "ns", "--ignore-running=false"},
			dynamic:     seeds[13].dynamicClient,
			input:       seeds[13].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 6 TaskRuns associated with Task \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all of clustertask with default --ignore-running",
			command:     []string{"delete", "--clustertask", "random", "-f", "-n", "ns", "--ignore-running"},
			dynamic:     seeds[14].dynamicClient,
			input:       seeds[14].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 5 TaskRuns(Completed) associated with ClusterTask \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all of clustertask with explicit --ignore-running true",
			command:     []string{"delete", "--clustertask", "random", "-f", "-n", "ns", "--ignore-running=true"},
			dynamic:     seeds[15].dynamicClient,
			input:       seeds[15].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 5 TaskRuns(Completed) associated with ClusterTask \"random\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all of clustertask with --ignore-running false",
			command:     []string{"delete", "--clustertask", "random", "-f", "-n", "ns", "--ignore-running=false"},
			dynamic:     seeds[16].dynamicClient,
			input:       seeds[16].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All 8 TaskRuns associated with ClusterTask \"random\" deleted in namespace \"ns\"\n",
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

func Test_ClusterTask_TaskRuns_Not_Deleted_With_Task_Option_v1beta1(t *testing.T) {
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
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
	cs, _ := test.SeedV1beta1TestData(t, test.Data{TaskRuns: trs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], versionv1beta1),
		cb.UnstructuredV1beta1CT(clustertasks[0], versionv1beta1),
		cb.UnstructuredV1beta1TR(trdata[0], versionv1beta1),
		cb.UnstructuredV1beta1TR(trdata[1], versionv1beta1),
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

func Test_ClusterTask_TaskRuns_Not_Deleted_With_Task_Option(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "random",
				Namespace: "ns",
			},
		},
	}

	trdata := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-1",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
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
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: "ClusterTask",
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	trs := trdata
	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredT(tasks[0], versionv1beta1),
		cb.UnstructuredTR(trdata[0], versionv1beta1),
		cb.UnstructuredTR(trdata[1], versionv1beta1),
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

func Test_Task_TaskRuns_Not_Deleted_With_ClusterTask_Option_v1beta1(t *testing.T) {
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
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
	cs, _ := test.SeedV1beta1TestData(t, test.Data{TaskRuns: trs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1T(tasks[0], versionv1beta1),
		cb.UnstructuredV1beta1CT(clustertasks[0], versionv1beta1),
		cb.UnstructuredV1beta1TR(trdata[0], versionv1beta1),
		cb.UnstructuredV1beta1TR(trdata[1], versionv1beta1),
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

func Test_Task_TaskRuns_Not_Deleted_With_ClusterTask_Option(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "random",
				Namespace: "ns",
			},
		},
	}

	trdata := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-1",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
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
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: "ClusterTask",
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	trs := trdata
	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredT(tasks[0], version),
		cb.UnstructuredTR(trdata[0], version),
		cb.UnstructuredTR(trdata[1], version),
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

func Test_TaskRuns_Delete_With_Running_PipelineRun_v1beta1(t *testing.T) {
	clock := test.FakeClock()

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

	trdata := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-1",
				Labels:    map[string]string{"tekton.dev/task": "random"},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "tekton.dev/v1beta1",
						Kind:               "PipelineRun",
						Name:               "pipeline-run-1",
						UID:                "",
						Controller:         nil,
						BlockOwnerDeletion: nil,
					}},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
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
					"tekton.dev/task": "random",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "tekton.dev/v1beta1",
						Kind:               "PipelineRun",
						Name:               "pipeline-run-1",
						UID:                "",
						Controller:         nil,
						BlockOwnerDeletion: nil,
					}},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: v1beta1.TaskRunReasonRunning.String(),
						},
					},
				},
			},
		},
	}

	prdata := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-1",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonRunning.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					// pipeline run starts now
					StartTime: &metav1.Time{Time: clock.Now()},
				},
			},
		},
	}

	type clients struct {
		pipelineClient test.Clients
		dynamicClient  dynamic.Interface
	}

	seeds := make([]clients, 0)
	for i := 0; i < 5; i++ {
		trs := trdata
		prs := prdata
		cs, _ := test.SeedV1beta1TestData(t, test.Data{TaskRuns: trs, Tasks: tasks, PipelineRuns: prs, Namespaces: ns})
		cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun", "pipelinerun"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredV1beta1T(tasks[0], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[0], versionv1beta1),
			cb.UnstructuredV1beta1TR(trdata[1], versionv1beta1),
			cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
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
		input       test.Clients
		inputStream io.Reader
		wantError   bool
		want        string
	}{
		{
			name:        "Taskrun with running pipelinerun and answer y",
			command:     []string{"rm", "tr0-1", "-n", "ns"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete TaskRun(s) \"tr0-1\" (y/n): TaskRun(s): tr0-1 attached to PipelineRun is still running deleting will restart the completed taskrun. Proceed (y/n): TaskRuns deleted: \"tr0-1\"\n",
		},
		{
			name:        "Taskrun with running pipelinerun force",
			command:     []string{"rm", "tr0-1", "-n", "ns", "-f"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "warning: Taskrun tr0-1 related pipelinerun still running.\nTaskRuns deleted: \"tr0-1\"\n",
		},
		{
			name:        "All taskruns with running pipelinerun ",
			command:     []string{"delete", "-n", "ns", "--all"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns in namespace \"ns\" (y/n): All 0 TaskRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "All taskruns with running pipelinerun ",
			command:     []string{"delete", "-n", "ns", "--all", "--ignore-running-pipelinerun=false"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns in namespace \"ns\" (y/n): All 2 TaskRuns(Completed) deleted in namespace \"ns\"\n",
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

func Test_TaskRuns_Delete_With_Running_PipelineRun(t *testing.T) {
	clock := test.FakeClock()

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	tasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "random",
				Namespace: "ns",
			},
		},
	}

	trdata := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-1",
				Labels:    map[string]string{"tekton.dev/task": "random"},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "tekton.dev/v1beta1",
						Kind:               "PipelineRun",
						Name:               "pipeline-run-1",
						UID:                "",
						Controller:         nil,
						BlockOwnerDeletion: nil,
					}},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
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
					"tekton.dev/task": "random",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "tekton.dev/v1",
						Kind:               "PipelineRun",
						Name:               "pipeline-run-1",
						UID:                "",
						Controller:         nil,
						BlockOwnerDeletion: nil,
					}},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: time.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: v1.TaskRunReasonRunning.String(),
						},
					},
				},
			},
		},
	}

	prdata := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-1",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.PipelineRunReasonRunning.String(),
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					// pipeline run starts now
					StartTime: &metav1.Time{Time: clock.Now()},
				},
			},
		},
	}

	type clients struct {
		pipelineClient pipelinetest.Clients
		dynamicClient  dynamic.Interface
	}

	seeds := make([]clients, 0)
	for i := 0; i < 5; i++ {
		trs := trdata
		prs := prdata
		cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Tasks: tasks, PipelineRuns: prs, Namespaces: ns})
		cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun", "pipelinerun"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredT(tasks[0], version),
			cb.UnstructuredTR(trdata[0], version),
			cb.UnstructuredTR(trdata[1], version),
			cb.UnstructuredPR(prs[0], version),
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
			name:        "Taskrun with running pipelinerun and answer y",
			command:     []string{"rm", "tr0-1", "-n", "ns"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete TaskRun(s) \"tr0-1\" (y/n): TaskRun(s): tr0-1 attached to PipelineRun is still running deleting will restart the completed taskrun. Proceed (y/n): TaskRuns deleted: \"tr0-1\"\n",
		},
		{
			name:        "Taskrun with running pipelinerun force",
			command:     []string{"rm", "tr0-1", "-n", "ns", "-f"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "warning: Taskrun tr0-1 related pipelinerun still running.\nTaskRuns deleted: \"tr0-1\"\n",
		},
		{
			name:        "All taskruns with running pipelinerun ",
			command:     []string{"delete", "-n", "ns", "--all"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns in namespace \"ns\" (y/n): All 0 TaskRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "All taskruns with running pipelinerun ",
			command:     []string{"delete", "-n", "ns", "--all", "--ignore-running-pipelinerun=false"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all TaskRuns in namespace \"ns\" (y/n): All 2 TaskRuns(Completed) deleted in namespace \"ns\"\n",
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

func Test_TaskRuns_Delete_With_Successful_PipelineRun(t *testing.T) {
	clock := test.FakeClock()

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-1",
				Labels:    map[string]string{"tekton.dev/task": "random"},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "tekton.dev/v1",
						Kind:               "PipelineRun",
						Name:               "pipeline-run-1",
						UID:                "",
						Controller:         nil,
						BlockOwnerDeletion: nil,
					}},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
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
					"tekton.dev/task": "random",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "tekton.dev/v1",
						Kind:               "PipelineRun",
						Name:               "pipeline-run-1",
						UID:                "",
						Controller:         nil,
						BlockOwnerDeletion: nil,
					}},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-3",
				Labels: map[string]string{
					"tekton.dev/task": "random",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "tekton.dev/v1",
						Kind:               "PipelineRun",
						Name:               "pipeline-run-1",
						UID:                "",
						Controller:         nil,
						BlockOwnerDeletion: nil,
					}},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr0-4",
				Labels: map[string]string{
					"tekton.dev/task": "random",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "tekton.dev/v1",
						Kind:               "PipelineRun",
						Name:               "pipeline-run-1",
						UID:                "",
						Controller:         nil,
						BlockOwnerDeletion: nil,
					}},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "random",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{
						Time: clock.Now(),
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-1",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					// pipeline run starts now
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now()},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
		cb.UnstructuredTR(trs[1], version),
		cb.UnstructuredTR(trs[2], version),
		cb.UnstructuredTR(trs[3], version),
		cb.UnstructuredPR(prs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Clock: clock}
	p.SetNamespace("ns")

	command := Command(p)
	command.SetIn(strings.NewReader("y"))
	expected := "Are you sure you want to delete all TaskRuns in namespace \"ns\" keeping 1 TaskRuns (y/n): All but 1 TaskRuns(Completed) deleted in namespace \"ns\"\n"
	out, err := test.ExecuteCommand(command, "delete", "--keep", "1")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, expected, out)

	client := &cli.Clients{
		Tekton:  cs.Pipeline,
		Kube:    cs.Kube,
		Dynamic: dc,
	}
	var tr v1.TaskRunList
	err = actions.ListV1(taskrunGroupResource, client, metav1.ListOptions{}, p.Namespace(), &tr)
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, 1, len(tr.Items))
}
