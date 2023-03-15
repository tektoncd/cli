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

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func TestResult(t *testing.T) {
	tt := []struct {
		name  string
		value v1.ParamValue
		want  string
	}{
		{
			name: "Empty value in the result",
			value: v1.ParamValue{
				Type:      v1.ParamTypeString,
				StringVal: "",
			},
			want: "",
		},
		{
			name: "Proper value in the result",
			value: v1.ParamValue{
				Type:      v1.ParamTypeString,
				StringVal: "sha: 2r17r2176r7\n",
			},
			want: "sha: 2r17r2176r7",
		},
		{
			name: "Empty line in the result",
			value: v1.ParamValue{
				Type:      v1.ParamTypeString,
				StringVal: "\n",
			},
			want: "",
		},
		{
			name: "Empty array in the result",
			value: v1.ParamValue{
				Type:     v1.ParamTypeArray,
				ArrayVal: []string{},
			},
			want: "",
		},
		{
			name: "Array in the result",
			value: v1.ParamValue{
				Type:     v1.ParamTypeArray,
				ArrayVal: []string{"foo", "bar"},
			},
			want: "foo, bar",
		},
		{
			name: "Empty object in the result",
			value: v1.ParamValue{
				Type:      v1.ParamTypeObject,
				ObjectVal: map[string]string{},
			},
			want: "{}",
		},
		{
			name: "Object in the result",
			value: v1.ParamValue{
				Type:      v1.ParamTypeObject,
				ObjectVal: map[string]string{"foo": "bar", "baz": "crazy"},
			},
			want: `{"baz":"crazy","foo":"bar"}`,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			got := Result(test.value)
			if got != test.want {
				t.Fatalf("Result failed: got %s, want %s", got, test.want)
			}
		})
	}
}
