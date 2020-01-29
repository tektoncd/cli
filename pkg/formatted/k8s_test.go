package formatted

import (
	"testing"

	"github.com/fatih/color"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck/v1beta1"
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
		condition v1beta1.Conditions
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
