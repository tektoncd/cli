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

package formatted

import (
	"testing"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"gotest.tools/v3/assert"
)

func TestFormatTaskConditions(t *testing.T) {
	tests := []struct {
		desc       string
		conditions []v1beta1.PipelineTaskCondition
		want       string
	}{
		{
			desc:       "task with no condtions",
			conditions: []v1beta1.PipelineTaskCondition{},
			want:       "---",
		},
		{
			desc: "task with a single condition",
			conditions: []v1beta1.PipelineTaskCondition{
				{ConditionRef: "test"},
			},
			want: "test",
		},
		{
			desc: "task with multiple conditions",
			conditions: []v1beta1.PipelineTaskCondition{
				{ConditionRef: "test-1"},
				{ConditionRef: "test-2"},
				{ConditionRef: "test-3"},
				{ConditionRef: "test-4"},
			},
			want: "test-1, test-2, test-3, test-4",
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(rt *testing.T) {
			got := TaskConditions(test.conditions)
			assert.Equal(rt, got, test.want)
		})
	}
}
