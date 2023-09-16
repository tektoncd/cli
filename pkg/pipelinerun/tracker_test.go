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
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	trh "github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stest "k8s.io/client-go/testing"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestTracker_pipelinerun_complete(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		ns           = "namespace"

		task1Name = "output-task-1"
		tr1Name   = "output-task-1"
		tr1Pods   = "output-task-1-pods-123456"

		task2Name = "output-task-2"
		tr2Name   = "output-task-2"
		tr2Pods   = "output-task-2-pods-123456"

		allTasks  = []string{task1Name, task2Name}
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
		}, {
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
		taskruns := []*v1.TaskRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tr1Name,
					Namespace: ns,
				},
				Spec: v1.TaskRunSpec{
					TaskRef: &v1.TaskRef{
						Name: task1Name,
					},
				},
				Status: v1.TaskRunStatus{
					TaskRunStatusFields: v1.TaskRunStatusFields{
						PodName: tr1Pods,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tr2Name,
					Namespace: ns,
				},
				Spec: v1.TaskRunSpec{
					TaskRef: &v1.TaskRef{
						Name: task2Name,
					},
				},
				Status: v1.TaskRunStatus{
					TaskRunStatusFields: v1.TaskRunStatusFields{
						PodName: tr2Pods,
					},
				},
			},
		}

		initialPR := []*v1.PipelineRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      prName,
					Namespace: ns,
					Labels:    map[string]string{"tekton.dev/pipeline": prName},
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Status: corev1.ConditionUnknown,
								Reason: v1.PipelineRunReasonRunning.String(),
							},
						},
					},
					PipelineRunStatusFields: v1.PipelineRunStatusFields{
						ChildReferences: []v1.ChildStatusReference{
							{
								Name:             tr1Name,
								PipelineTaskName: task1Name,
								TypeMeta: runtime.TypeMeta{
									APIVersion: "tekton.dev/v1",
									Kind:       "TaskRun",
								},
							},
						},
					},
				},
			},
		}

		pr := &v1.PipelineRun{
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
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             tr1Name,
							PipelineTaskName: task1Name,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1",
								Kind:       "TaskRun",
							},
						}, {
							Name:             tr2Name,
							PipelineTaskName: task2Name,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1",
								Kind:       "TaskRun",
							},
						},
					},
				},
			},
		}

		tc := startPipelineRun(t, pipelinetest.Data{PipelineRuns: initialPR, TaskRuns: taskruns}, pr.Status)
		tracker := NewTracker(pipelineName, ns, tc)
		if err := actions.InitializeAPIGroupRes(tc.Tekton.Discovery()); err != nil {
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

func startPipelineRun(t *testing.T, data pipelinetest.Data, prStatus ...v1.PipelineRunStatus) *cli.Clients {
	cs, _ := test.SeedTestData(t, data)

	// to keep pushing the taskrun over the period(simulate watch)
	watcher := watch.NewFake()
	cs.Pipeline.PrependWatchReactor("pipelineruns", k8stest.DefaultWatchReactor(watcher, nil))
	cs.Pipeline.Resources = cb.APIResourceList("v1", []string{"task", "taskrun", "pipeline", "pipelinerun"})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(data.PipelineRuns[0], "v1"),
		cb.UnstructuredTR(data.TaskRuns[0], "v1"),
		cb.UnstructuredTR(data.TaskRuns[1], "v1"),
	)
	if err != nil {
		fmt.Println(err)
	}

	go func() {
		for _, status := range prStatus {
			time.Sleep(time.Second * 2)
			data.PipelineRuns[0].Status = status
			watcher.Modify(data.PipelineRuns[0])
		}
	}()

	return &cli.Clients{
		Tekton:  cs.Pipeline,
		Kube:    cs.Kube,
		Dynamic: dynamic,
	}
}
