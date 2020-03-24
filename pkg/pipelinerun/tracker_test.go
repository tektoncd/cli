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

	trh "github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/cli/pkg/test"
	clitest "github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinetest "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	k8stest "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
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

	taskruns := []*v1alpha1.TaskRun{
		tb.TaskRun(tr1Name, ns,
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(task1Name),
			),
			tb.TaskRunStatus(
				tb.PodName(tr1Pod),
			),
		),
		tb.TaskRun(tr2Name, ns,
			tb.TaskRunSpec(
				tb.TaskRunTaskRef(task2Name),
			),
			tb.TaskRunStatus(
				tb.PodName(tr2Pod),
			),
		),
	}

	initialPR := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName, ns,
			tb.PipelineRunLabel("tekton.dev/pipeline", prName),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionUnknown,
					Reason: resources.ReasonRunning,
				}),
				tb.PipelineRunTaskRunsStatus(tr1Name, &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: task1Name,
					Status:           &taskruns[0].Status,
				}),
			),
		),
	}

	prStatusFn := tb.PipelineRunStatus(
		tb.PipelineRunStatusCondition(apis.Condition{
			Status: corev1.ConditionTrue,
			Reason: resources.ReasonSucceeded,
		}),
		tb.PipelineRunTaskRunsStatus(tr1Name, &v1alpha1.PipelineRunTaskRunStatus{
			PipelineTaskName: task1Name,
			Status:           &taskruns[0].Status,
		}),
		tb.PipelineRunTaskRunsStatus(tr2Name, &v1alpha1.PipelineRunTaskRunStatus{
			PipelineTaskName: task2Name,
			Status:           &taskruns[1].Status,
		}),
	)
	pr := &v1alpha1.PipelineRun{}
	prStatusFn(pr)
	data1 := pipelinetest.Data{PipelineRuns: initialPR, TaskRuns: taskruns}

	// define conditional pipelinerun
	tr2 := []*v1alpha1.TaskRun{
		tb.TaskRun("condition-taskrun", "ns",
			tb.TaskRunSpec(tb.TaskRunTaskRef("condition-task")),
			tb.TaskRunStatus(tb.PodName("condition-taskrun-pod-123")),
		),
	}

	prccs := make(map[string]*v1alpha1.PipelineRunConditionCheckStatus)
	prccs["condition-pod-123"] = &v1alpha1.PipelineRunConditionCheckStatus{
		ConditionName: "cond-1",
		Status: &v1alpha1.ConditionCheckStatus{
			ConditionCheckStatusFields: v1alpha1.ConditionCheckStatusFields{
				PodName: "condition-pod-123",
			},
		},
	}

	initialPR2 := []*v1alpha1.PipelineRun{
		tb.PipelineRun("condition-pipeline-run", "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", "condition-pipeline-run"),
			tb.PipelineRunSpec("", tb.PipelineRunPipelineSpec(
				tb.PipelineTask("condition-task", "condition-task",
					tb.PipelineTaskCondition("cond-1")),
			)),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionUnknown,
					Reason: resources.ReasonRunning,
				}),
				tb.PipelineRunTaskRunsStatus("condition-taskrun", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "condition-task",
					Status:           &tr2[0].Status,
					ConditionChecks:  prccs,
				}),
			),
		),
	}

	prStatusFn2 := tb.PipelineRunStatus(
		tb.PipelineRunStatusCondition(apis.Condition{
			Status: corev1.ConditionTrue,
			Reason: resources.ReasonSucceeded,
		}),
		tb.PipelineRunTaskRunsStatus("condition-taskrun", &v1alpha1.PipelineRunTaskRunStatus{
			PipelineTaskName: "condition-task",
			Status:           &tr2[0].Status,
			ConditionChecks:  prccs,
		}),
	)
	pr2 := &v1alpha1.PipelineRun{}
	prStatusFn2(pr2)
	data2 := pipelinetest.Data{PipelineRuns: initialPR2, TaskRuns: tr2}

	scenarios := []struct {
		name         string
		pipelineName string
		tasks        []string
		namespace    string
		data         pipelinetest.Data
		prStatus     v1alpha1.PipelineRunStatus
		expected     []trh.Run
	}{
		{
			name:         "for all tasks",
			pipelineName: pipelineName,
			tasks:        allTasks,
			namespace:    ns,
			data:         data1,
			prStatus:     pr.Status,
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
			name:         "for one task",
			pipelineName: pipelineName,
			tasks:        onlyTask1,
			namespace:    ns,
			data:         data1,
			prStatus:     pr.Status,
			expected: []trh.Run{
				{
					Name: tr1Name,
					Task: task1Name,
				},
			},
		},
		{
			name:         "conditonal pipeline",
			pipelineName: "condition-pipeline",
			tasks:        allTasks,
			namespace:    "ns",
			data:         data2,
			prStatus:     pr2.Status,
			expected: []trh.Run{
				{
					Name: "condition-pod-123",
					Task: "cond-1",
				}, {
					Name: "condition-taskrun",
					Task: "condition-task",
				},
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			tc := startPipelineRun(t, s.data, s.prStatus)

			tracker := NewTracker(s.pipelineName, s.namespace, tc)
			output := taskRunsFor(s.tasks, tracker)

			clitest.AssertOutput(t, s.expected, output)
		})
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

	go func() {
		for _, status := range prStatus {
			time.Sleep(time.Second * 2)
			data.PipelineRuns[0].Status = status
			if !watcher.IsStopped() {
				watcher.Modify(data.PipelineRuns[0])
			}
		}
	}()

	return cs.Pipeline
}
