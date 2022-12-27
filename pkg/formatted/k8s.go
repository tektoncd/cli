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

package formatted

import (
	"fmt"
	"sync/atomic"

	"github.com/fatih/color"
	corev1 "k8s.io/api/core/v1"
	v1 "knative.dev/pkg/apis/duck/v1"
)

var ConditionColor = map[string]color.Attribute{
	"Failed":    color.FgHiRed,
	"Succeeded": color.FgHiGreen,
	"Running":   color.FgHiBlue,
	"Cancelled": color.FgHiYellow,
	"Completed": color.FgHiMagenta,
	"Pending":   color.FgHiBlue,
	"Started":   color.FgHiCyan,
}

var stepCounter uint64

// ColorStatus Get a status coloured
func ColorStatus(status string) string {
	return color.New(ConditionColor[status]).Sprint(status)
}

// AutoStepName when our stepName is empty return a generated name as generated
// on pipeLine
func AutoStepName(stepName string) string {
	unnamedStep := fmt.Sprintf("unnamed-%d", stepCounter)
	atomic.AddUint64(&stepCounter, 1)
	if stepName != "" {
		return stepName
	}
	return unnamedStep
}

// Condition returns a human readable text based on the status of the Condition
func Condition(c v1.Conditions) string {
	var status string
	if len(c) == 0 {
		return "---"
	}

	switch c[0].Status {
	case corev1.ConditionFalse:
		status = "Failed"
	case corev1.ConditionTrue:
		status = "Succeeded"
	case corev1.ConditionUnknown:
		status = "Running"
	}

	if c[0].Reason != "" && c[0].Reason != status {
		switch c[0].Reason {
		case "PipelineRunCancelled", "TaskRunCancelled", "Cancelled":
			return ColorStatus("Cancelled") + "(" + c[0].Reason + ")"
		case "PipelineRunStopping", "TaskRunStopping":
			return ColorStatus("Failed") + "(" + c[0].Reason + ")"
		case "CreateContainerConfigError", "ExceededNodeResources", "ExceededResourceQuota":
			return ColorStatus("Pending") + "(" + c[0].Reason + ")"
		default:
			return ColorStatus(status) + "(" + c[0].Reason + ")"
		}
	}
	return ColorStatus(status)
}
