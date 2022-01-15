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

package description

import (
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestPipelineRefExists_Present(t *testing.T) {
	spec := v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{
			Name: "Pipeline",
		},
	}

	output := pipelineRefExists(spec)
	test.AssertOutput(t, "Pipeline", output)
}

func TestPipelineRefExists_Not_Present(t *testing.T) {
	spec := v1beta1.PipelineRunSpec{
		PipelineRef: nil,
	}

	output := pipelineRefExists(spec)
	test.AssertOutput(t, "", output)
}

func TestPipelineResourceRefExists_Present(t *testing.T) {
	spec := v1beta1.PipelineResourceBinding{
		ResourceRef: &v1beta1.PipelineResourceRef{
			Name: "Pipeline",
		},
	}

	output := pipelineResourceRefExists(spec)
	test.AssertOutput(t, "Pipeline", output)
}

func TestPipelineResourceRefExists_Not_Present(t *testing.T) {
	spec := v1beta1.PipelineResourceBinding{
		ResourceRef: nil,
	}

	output := pipelineResourceRefExists(spec)
	test.AssertOutput(t, "", output)
}

func TestHasFailed_PipelineAndTaskRunFailedMessage(t *testing.T) {

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Type:    apis.ConditionSucceeded,
							Reason:  v1beta1.TaskRunReasonCancelled.String(),
							Message: "TaskRun \"tr-1\" was cancelled",
						},
					},
				},
			},
		},
	}

	pipelineRuns := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pipeline-run",
			},
			Status: v1beta1.PipelineRunStatus{
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					TaskRuns: map[string]*v1beta1.PipelineRunTaskRunStatus{
						"tr-1": {PipelineTaskName: "t-1", Status: &trs[0].Status},
					},
				},
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
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

	message := hasFailed(pipelineRuns[0])
	test.AssertOutput(t, "PipelineRun \"pipeline-run\" was cancelled\nTaskRun(s) cancelled: tr-1", message)
}

func TestHasFailed_PipelineFailedMessage(t *testing.T) {

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tr-1",
				Namespace: "ns",
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
							Reason: string(v1beta1.TaskRunReasonSuccessful),
						},
					},
				},
			},
		},
	}

	pipelineRuns := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pipeline-run",
			},
			Status: v1beta1.PipelineRunStatus{
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					TaskRuns: map[string]*v1beta1.PipelineRunTaskRunStatus{
						"tr-1": {PipelineTaskName: "t-1", Status: &trs[0].Status},
					},
				},
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
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

	message := hasFailed(pipelineRuns[0])
	test.AssertOutput(t, "PipelineRun \"pipeline-run\" was cancelled", message)
}
