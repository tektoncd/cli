// Copyright © 2022 The Tekton Authors.
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

package formatted

import (
	"testing"

	"github.com/fatih/color"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestColor(t *testing.T) {
	greenSuccess := color.New(color.FgHiGreen).Sprint("Succeeded")
	cs := ColorStatus("Succeeded")
	if cs != greenSuccess {
		t.Errorf("%s != %s", cs, greenSuccess)
	}
}

func TestAutoStepName(t *testing.T) {
	firstRound := AutoStepName("")
	assert.Equal(t, firstRound, "unnamed-0")

	secondRound := AutoStepName("named")
	assert.Equal(t, secondRound, "named")

	thirdRound := AutoStepName("")
	assert.Equal(t, thirdRound, "unnamed-2")
}

func TestCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition duckv1.Conditions
		want      string
	}{
		{
			name: "Failed status",
			condition: []apis.Condition{{
				Type:   apis.ConditionSucceeded,
				Status: corev1.ConditionFalse,
			}},
			want: "Failed",
		},
		{
			name: "Succeeded status",
			condition: []apis.Condition{{
				Type:   apis.ConditionSucceeded,
				Status: corev1.ConditionTrue,
			}},
			want: "Succeeded",
		},
		{
			name: "Running status",
			condition: []apis.Condition{{
				Type:   apis.ConditionReady,
				Status: corev1.ConditionUnknown,
			}},
			want: "Running",
		},
		{
			name: "PipelineRunCanceled status reason",
			condition: []apis.Condition{{
				Type:   apis.ConditionSucceeded,
				Status: corev1.ConditionTrue,
				Reason: "PipelineRunCancelled",
			}},
			want: "Cancelled(PipelineRunCancelled)",
		},
		{
			name: "TaskRunCanceled status reason",
			condition: []apis.Condition{{
				Type:   apis.ConditionSucceeded,
				Status: corev1.ConditionTrue,
				Reason: "TaskRunCancelled",
			}},
			want: "Cancelled(TaskRunCancelled)",
		},
		{
			name: "Cancelled status reason",
			condition: []apis.Condition{{
				Type:   apis.ConditionSucceeded,
				Status: corev1.ConditionFalse,
				Reason: "Cancelled",
			}},
			want: "Cancelled(Cancelled)",
		},
		{
			name: "PipelineRunTimeout status reason",
			condition: []apis.Condition{{
				Type:   apis.ConditionSucceeded,
				Status: corev1.ConditionFalse,
				Reason: "PipelineRunTimeout",
			}},
			want: "Failed(PipelineRunTimeout)",
		},
		{
			name: "TaskRunTimeout status reason",
			condition: []apis.Condition{{
				Type:   apis.ConditionSucceeded,
				Status: corev1.ConditionFalse,
				Reason: "TaskRunTimeout",
			}},
			want: "Failed(TaskRunTimeout)",
		},
		{
			name: "PipelineRunStopping status reason",
			condition: []apis.Condition{{
				Type:   apis.ConditionSucceeded,
				Status: corev1.ConditionUnknown,
				Reason: "PipelineRunStopping",
			}},
			want: "Failed(PipelineRunStopping)",
		},
		{
			name: "ConfigError status reason",
			condition: []apis.Condition{{
				Type:   apis.ConditionSucceeded,
				Status: corev1.ConditionUnknown,
				Reason: "CreateContainerConfigError",
			}},
			want: "Pending(CreateContainerConfigError)",
		},
		{
			name: "Reason equal Status",
			condition: []apis.Condition{{
				Type:   apis.ConditionSucceeded,
				Status: corev1.ConditionTrue,
				Reason: "Succeeded",
			}},
			want: "Succeeded",
		},
		{
			name: "Reason not equal Status",
			condition: []apis.Condition{{
				Type:   apis.ConditionSucceeded,
				Status: corev1.ConditionTrue,
				Reason: "Ragnarok",
			}},
			want: "Succeeded(Ragnarok)",
		},
		{
			name:      "No Conditions",
			condition: []apis.Condition{},
			want:      "---",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Condition(tt.condition); got != tt.want {
				t.Errorf("Condition() = %v, want %v", got, tt.want)
			}
		})
	}
}
