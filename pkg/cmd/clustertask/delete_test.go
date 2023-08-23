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

package clustertask

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestClusterTaskDelete(t *testing.T) {
	version := "v1beta1"
	clock := test.FakeClock()
	taskCreated := clock.Now().Add(-1 * time.Minute)

	type clients struct {
		pipelineClient test.Clients
		dynamicClient  dynamic.Interface
	}

	clusterTaskData := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes",
				CreationTimestamp: metav1.Time{Time: taskCreated},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes2",
				CreationTimestamp: metav1.Time{Time: taskCreated},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes3",
				CreationTimestamp: metav1.Time{Time: taskCreated},
			},
		},
	}
	taskRunData := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-run-1",
				Labels:    map[string]string{"tekton.dev/clusterTask": "tomatoes"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "tomatoes",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
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
				Name:      "task-run-2",
				Labels:    map[string]string{"tekton.dev/clusterTask": "tomatoes"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "tomatoes",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
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
		// NamespacedTask (Task) is provided in the TaskRef of TaskRun, so as to
		// verify TaskRun created by Task is not getting deleted while deleting
		// ClusterTask with `--trs` flag and name of Task and ClusterTask is same.
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "task-run-3",
				Labels:    map[string]string{"tekton.dev/task": "tomatoes"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "tomatoes",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
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

	seeds := make([]clients, 0)
	for i := 0; i < 6; i++ {
		cs, _ := test.SeedV1beta1TestData(t, test.Data{
			ClusterTasks: clusterTaskData,
			TaskRuns:     taskRunData,
		})
		cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask", "taskrun"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredV1beta1CT(clusterTaskData[0], version),
			cb.UnstructuredV1beta1CT(clusterTaskData[1], version),
			cb.UnstructuredV1beta1CT(clusterTaskData[2], version),
			cb.UnstructuredV1beta1TR(taskRunData[0], version),
			cb.UnstructuredV1beta1TR(taskRunData[1], version),
			cb.UnstructuredV1beta1TR(taskRunData[2], version),
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
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "tomatoes", "-f"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Command \"delete\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nClusterTasks deleted: \"tomatoes\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "tomatoes", "--force"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Command \"delete\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nClusterTasks deleted: \"tomatoes\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "tomatoes"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting ClusterTask(s) \"tomatoes\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "tomatoes"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Command \"delete\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nAre you sure you want to delete ClusterTask(s) \"tomatoes\" (y/n): ClusterTasks deleted: \"tomatoes\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "clustertasks.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "clustertasks.tekton.dev \"nonexistent\" not found; clustertasks.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Remove multiple non existent resources with --trs flag",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns", "--trs"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "clustertasks.tekton.dev \"nonexistent\" not found; clustertasks.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "With delete taskrun(s) flag, reply yes",
			command:     []string{"rm", "tomatoes", "-n", "ns", "--trs"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Command \"delete\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nAre you sure you want to delete ClusterTask(s) \"tomatoes\" and related resources (y/n): TaskRuns deleted: \"task-run-1\", \"task-run-2\"\nClusterTasks deleted: \"tomatoes\"\n",
		},
		{
			name:        "With force delete flag, reply yes, multiple clustertasks",
			command:     []string{"rm", "tomatoes2", "tomatoes3", "-f"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Command \"delete\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nClusterTasks deleted: \"tomatoes2\", \"tomatoes3\"\n",
		},
		{
			name:        "Without force delete flag, reply yes, multiple clustertasks",
			command:     []string{"rm", "tomatoes2", "tomatoes3"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Command \"delete\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nAre you sure you want to delete ClusterTask(s) \"tomatoes2\", \"tomatoes3\" (y/n): ClusterTasks deleted: \"tomatoes2\", \"tomatoes3\"\n",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Command \"delete\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nAre you sure you want to delete all ClusterTasks (y/n): All ClusterTasks deleted\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Command \"delete\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nAll ClusterTasks deleted\n",
		},
		{
			name:        "Error from using clustertask name with --all",
			command:     []string{"delete", "tomatoes2", "--all"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from using clustertask delete with no names or --all",
			command:     []string{"delete"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "must provide ClusterTask name(s) or use --all flag with delete",
		},
		{
			name:        "Delete the ClusterTask present and give error for non-existent ClusterTask",
			command:     []string{"delete", "nonexistent", "tomatoes2"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "clustertasks.tekton.dev \"nonexistent\" not found",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Dynamic: tp.dynamic}
			clustertask := Command(p)

			if tp.inputStream != nil {
				clustertask.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(clustertask, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("Error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else {
				if err != nil {
					t.Errorf("Unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}
