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

package pipelinerun

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/cli/pkg/actions"
	trh "github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	k8stest "k8s.io/client-go/testing"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestTracker_pipelinerun_complete(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		ns           = "namespace"

		task1Name = "output-task-1"
		tr1Name   = "output-task-1"
		tr1Pod    = "output-task-1-pod-123456"

		task2Name = "output-task-2"
		tr2Name   = "output-task-2"
		tr2Pod    = "output-task-2-pod-123456"

		allTasks  = []string{}
		onlyTask1 = []string{task1Name}
	)

	scenarios := []struct {
		name               string
		tasks              []string
		fullEmbeddedStatus bool
		expected           []trh.Run
	}{
		{
			name:               "for all tasks, full status",
			tasks:              allTasks,
			fullEmbeddedStatus: true,
			expected: []trh.Run{
				{
					Name: tr1Name,
					Task: task1Name,
				}, {
					Name: tr2Name,
					Task: task2Name,
				},
			},
		}, {
			name:               "for one task, full status",
			tasks:              onlyTask1,
			fullEmbeddedStatus: true,
			expected: []trh.Run{
				{
					Name: tr1Name,
					Task: task1Name,
				},
			},
		}, {
			name:               "for all tasks, minimal status",
			tasks:              allTasks,
			fullEmbeddedStatus: false,
			expected: []trh.Run{
				{
					Name: tr1Name,
					Task: task1Name,
				}, {
					Name: tr2Name,
					Task: task2Name,
				},
			},
		}, {
			name:               "for one task, minimal status",
			tasks:              onlyTask1,
			fullEmbeddedStatus: false,
			expected: []trh.Run{
				{
					Name: tr1Name,
					Task: task1Name,
				},
			},
		},
	}

	for _, s := range scenarios {
		taskruns := []*v1beta1.TaskRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tr1Name,
					Namespace: ns,
				},
				Spec: v1beta1.TaskRunSpec{
					TaskRef: &v1beta1.TaskRef{
						Name: task1Name,
					},
				},
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						PodName: tr1Pod,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tr2Name,
					Namespace: ns,
				},
				Spec: v1beta1.TaskRunSpec{
					TaskRef: &v1beta1.TaskRef{
						Name: task2Name,
					},
				},
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						PodName: tr2Pod,
					},
				},
			},
		}

		initialPR := []*v1beta1.PipelineRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      prName,
					Namespace: ns,
					Labels:    map[string]string{"tekton.dev/pipeline": prName},
				},
				Status: v1beta1.PipelineRunStatus{
					Status: duckv1beta1.Status{
						Conditions: duckv1beta1.Conditions{
							{
								Status: corev1.ConditionUnknown,
								Reason: v1beta1.PipelineRunReasonRunning.String(),
							},
						},
					},
					PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{},
				},
			},
		}
		if s.fullEmbeddedStatus {
			initialPR[0].Status.TaskRuns = map[string]*v1beta1.PipelineRunTaskRunStatus{
				tr1Name: {PipelineTaskName: task1Name, Status: &taskruns[0].Status},
			}
		} else {
			initialPR[0].Status.ChildReferences = []v1beta1.ChildStatusReference{{
				Name:             tr1Name,
				PipelineTaskName: task1Name,
				TypeMeta: runtime.TypeMeta{
					APIVersion: "tekton.dev/v1beta1",
					Kind:       "TaskRun",
				},
			}}
		}

		pr := &v1beta1.PipelineRun{
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					TaskRuns: map[string]*v1beta1.PipelineRunTaskRunStatus{
						tr1Name: {PipelineTaskName: task1Name, Status: &taskruns[0].Status},
						tr2Name: {PipelineTaskName: task2Name, Status: &taskruns[1].Status},
					},
				},
			},
		}
		if s.fullEmbeddedStatus {
			initialPR[0].Status.TaskRuns = map[string]*v1beta1.PipelineRunTaskRunStatus{
				tr1Name: {PipelineTaskName: task1Name, Status: &taskruns[0].Status},
				tr2Name: {PipelineTaskName: task2Name, Status: &taskruns[1].Status},
			}
		} else {
			initialPR[0].Status.ChildReferences = []v1beta1.ChildStatusReference{{
				Name:             tr1Name,
				PipelineTaskName: task1Name,
				TypeMeta: runtime.TypeMeta{
					APIVersion: "tekton.dev/v1beta1",
					Kind:       "TaskRun",
				},
			}, {
				Name:             tr2Name,
				PipelineTaskName: task2Name,
				TypeMeta: runtime.TypeMeta{
					APIVersion: "tekton.dev/v1beta1",
					Kind:       "TaskRun",
				},
			}}
		}

		tc := startPipelineRun(t, pipelinev1beta1test.Data{PipelineRuns: initialPR, TaskRuns: taskruns}, pr.Status)
		tracker := NewTracker(pipelineName, ns, tc)
		if err := actions.InitializeAPIGroupRes(tracker.Tekton.Discovery()); err != nil {
			t.Errorf("failed to initialize APIGroup Resource")
		}
		output := taskRunsFor(s.tasks, tracker)

		if d := cmp.Diff(s.expected, output, cmpopts.SortSlices(func(i, j trh.Run) bool { return i.Name < j.Name })); d != "" {
			t.Errorf("Unexpected output: %s", diff.PrintWantGot(d))
		}

	}
}

func taskRunsFor(onlyTasks []string, tracker *Tracker) []trh.Run {
	output := []trh.Run{}
	for ts := range tracker.Monitor(onlyTasks) {
		output = append(output, ts...)
	}
	return output
}

func startPipelineRun(t *testing.T, data pipelinev1beta1test.Data, prStatus ...v1beta1.PipelineRunStatus) versioned.Interface {
	cs, _ := test.SeedV1beta1TestData(t, data)

	// to keep pushing the taskrun over the period(simulate watch)
	watcher := watch.NewFake()
	cs.Pipeline.PrependWatchReactor("pipelineruns", k8stest.DefaultWatchReactor(watcher, nil))
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"task", "taskrun", "pipeline", "pipelinerun"})
	go func() {
		for _, status := range prStatus {
			time.Sleep(time.Second * 2)
			data.PipelineRuns[0].Status = status
			watcher.Modify(data.PipelineRuns[0])
		}
	}()

	return cs.Pipeline
}
