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

package builder

import (
	"time"

	tb "github.com/tektoncd/cli/internal/builder/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineRunCreationTimestamp sets the creation time of the pipelinerun
func PipelineRunCreationTimestamp(t time.Time) tb.PipelineRunOp {
	return func(p *v1alpha1.PipelineRun) {
		p.CreationTimestamp = metav1.Time{Time: t}
	}
}

// PipelineRunCompletionTime sets the completion time  to the PipelineRunStatus.
func PipelineRunCompletionTime(t time.Time) tb.PipelineRunStatusOp {
	return func(s *v1alpha1.PipelineRunStatus) {
		s.CompletionTime = &metav1.Time{Time: t}
	}
}
