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

import "testing"

func TestResult(t *testing.T) {
	tt := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "Empty value in the result",
			value: "",
			want:  "",
		},
		{
			name:  "Proper value in the result",
			value: "sha: 2r17r2176r7\n",
			want:  "sha: 2r17r2176r7",
		},
		{
			name:  "Empty line in the result",
			value: "\n",
			want:  "",
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
