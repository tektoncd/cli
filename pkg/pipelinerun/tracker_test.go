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

	"github.com/tektoncd/cli/pkg/actions"
	trh "github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/cli/pkg/test"
	clitest "github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"

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
		name     string
		tasks    []string
		expected []trh.Run
	}{
		{
			name:  "for all tasks",
			tasks: allTasks,
			expected: []trh.Run{
				{
					Name: tr1Name,
					Task: task1Name,
				}, {
					Name: tr2Name,
					Task: task2Name,
				},
			},
		},
		{
			name:  "for one task",
			tasks: onlyTask1,
			expected: []trh.Run{
				{
					Name: tr1Name,
					Task: task1Name,
				},
			},
		},
	}

	for _, s := range scenarios {
		taskruns := []*v1alpha1.TaskRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tr1Name,
					Namespace: ns,
				},
				Spec: v1alpha1.TaskRunSpec{
					TaskRef: &v1beta1.TaskRef{
						Name: task1Name,
					},
				},
				Status: v1alpha1.TaskRunStatus{
					TaskRunStatusFields: v1alpha1.TaskRunStatusFields{
						PodName: tr1Pod,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tr2Name,
					Namespace: ns,
				},
				Spec: v1alpha1.TaskRunSpec{
					TaskRef: &v1beta1.TaskRef{
						Name: task2Name,
					},
				},
				Status: v1alpha1.TaskRunStatus{
					TaskRunStatusFields: v1alpha1.TaskRunStatusFields{
						PodName: tr2Pod,
					},
				},
			},
		}

		initialPR := []*v1alpha1.PipelineRun{
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
					PipelineRunStatusFields: v1alpha1.PipelineRunStatusFields{
						TaskRuns: map[string]*v1alpha1.PipelineRunTaskRunStatus{
							tr1Name: {PipelineTaskName: task1Name, Status: &taskruns[0].Status},
						},
					},
				},
			},
		}
		pr := &v1alpha1.PipelineRun{
			Status: v1alpha1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1alpha1.PipelineRunStatusFields{
					TaskRuns: map[string]*v1alpha1.PipelineRunTaskRunStatus{
						tr1Name: {PipelineTaskName: task1Name, Status: &taskruns[0].Status},
						tr2Name: {PipelineTaskName: task2Name, Status: &taskruns[1].Status},
					},
				},
			},
		}

		tc := startPipelineRun(t, pipelinetest.Data{PipelineRuns: initialPR, TaskRuns: taskruns}, pr.Status)
		tracker := NewTracker(pipelineName, ns, tc)
		if err := actions.InitializeAPIGroupRes(tracker.Tekton.Discovery()); err != nil {
			t.Errorf("failed to initialize APIGroup Resource")
		}
		output := taskRunsFor(s.tasks, tracker)

		clitest.AssertOutput(t, s.expected, output)
	}
}

func taskRunsFor(onlyTasks []string, tracker *Tracker) []trh.Run {
	output := []trh.Run{}
	for ts := range tracker.Monitor(onlyTasks) {
		output = append(output, ts...)
	}
	return output
}

func startPipelineRun(t *testing.T, data pipelinetest.Data, prStatus ...v1alpha1.PipelineRunStatus) versioned.Interface {
	cs, _ := test.SeedTestData(t, data)

	// to keep pushing the taskrun over the period(simulate watch)
	watcher := watch.NewFake()
	cs.Pipeline.PrependWatchReactor("pipelineruns", k8stest.DefaultWatchReactor(watcher, nil))
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"task", "taskrun", "pipeline", "pipelinerun"})
	go func() {
		for _, status := range prStatus {
			time.Sleep(time.Second * 2)
			data.PipelineRuns[0].Status = status
			watcher.Modify(data.PipelineRuns[0])
		}
	}()

	return cs.Pipeline
}
