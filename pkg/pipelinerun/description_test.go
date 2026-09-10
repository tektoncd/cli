// Copyright © 2020 The Tekton Authors.
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
	"sort"
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestHasFailed_PipelineAndTaskRunFailedMessage(t *testing.T) {
	taskRunWithStatusList := TaskRunWithStatusList{
		{
			TaskRunName:      "tr-1",
			PipelineTaskName: "task",
			Status: &v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Type:    apis.ConditionSucceeded,
							Reason:  v1.TaskRunReasonCancelled.String(),
							Message: "TaskRun \"tr-1\" was cancelled",
						},
					},
				},
			},
		},
	}

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pipeline-run",
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "test",
						},
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  "PipelineRunCancelled",
							Message: "PipelineRun \"pipeline-run\" was cancelled",
						},
					},
				},
			},
		},
	}

	message := hasFailed(pipelineRuns[0], taskRunWithStatusList)
	test.AssertOutput(t, "PipelineRun \"pipeline-run\" was cancelled\nTaskRun(s) cancelled: tr-1", message)
}

func TestHasFailed_PipelineFailedMessage(t *testing.T) {
	taskRunWithStatusList := TaskRunWithStatusList{
		{
			TaskRunName:      "tr-1",
			PipelineTaskName: "t-1",
			Status: &v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
							Reason: v1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	pipelineRuns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pipeline-run",
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             "tr-1",
							PipelineTaskName: "t-1",
						},
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  "PipelineRunCancelled",
							Message: "PipelineRun \"pipeline-run\" was cancelled",
						},
					},
				},
			},
		},
	}

	message := hasFailed(pipelineRuns[0], taskRunWithStatusList)
	test.AssertOutput(t, "PipelineRun \"pipeline-run\" was cancelled", message)
}

func withStartTime(name string, start *metav1.Time) TaskRunWithStatus {
	return TaskRunWithStatus{
		TaskRunName: name,
		Status: &v1.TaskRunStatus{
			Status: duckv1.Status{Conditions: duckv1.Conditions{{
				Status: corev1.ConditionTrue, Type: apis.ConditionSucceeded}}},
			TaskRunStatusFields: v1.TaskRunStatusFields{StartTime: start},
		},
	}
}

func TestTaskRunWithStatusList_SortDeterministic(t *testing.T) {
	start := test.FakeClock().Now().Add(2 * time.Minute)
	// TaskRuns sharing an identical StartTime must order by name, regardless of
	// the input order handed over from map iteration.
	list := TaskRunWithStatusList{
		withStartTime("tr-b", &metav1.Time{Time: start}),
		withStartTime("tr-a", &metav1.Time{Time: start}),
	}
	sort.Sort(list)
	test.AssertOutput(t, "tr-a", list[0].TaskRunName)
	test.AssertOutput(t, "tr-b", list[1].TaskRunName)

	// Distinct StartTimes keep the existing latest-first ordering.
	later := test.FakeClock().Now().Add(5 * time.Minute)
	list = TaskRunWithStatusList{
		withStartTime("tr-a", &metav1.Time{Time: start}),
		withStartTime("tr-b", &metav1.Time{Time: later}),
	}
	sort.Sort(list)
	test.AssertOutput(t, "tr-b", list[0].TaskRunName)
	test.AssertOutput(t, "tr-a", list[1].TaskRunName)

	// Both TaskRuns without StartTime order by name as well.
	list = TaskRunWithStatusList{
		withStartTime("tr-b", nil),
		withStartTime("tr-a", nil),
	}
	sort.Sort(list)
	test.AssertOutput(t, "tr-a", list[0].TaskRunName)
	test.AssertOutput(t, "tr-b", list[1].TaskRunName)
}
