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
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
)

// If description is longer than 20 char then it will return
// initial 20 chars suffixed by ...
func FormatDesc(desc string) string {
	if len(desc) > 20 {
		return desc[0:19] + "..."
	}
	return desc
}

func RemoveLastAppliedConfig(annotations map[string]string) map[string]string {
	removed := map[string]string{}
	for k, v := range annotations {
		if k != corev1.LastAppliedConfigAnnotation {
			removed[k] = v
		}
	}
	return removed
}

// Check if PipelineRef exists on a PipelineRunSpec. Returns empty string if not present.
func PipelineRefExists(spec v1.PipelineRunSpec) string {
	if spec.PipelineRef == nil {
		return ""
	}

	return spec.PipelineRef.Name
}

// Check if TaskRef exists on a TaskRunSpec. Returns empty string if not present.
func TaskRefExists(spec v1.TaskRunSpec) string {
	if spec.TaskRef == nil {
		return ""
	}

	return spec.TaskRef.Name
}
