// Copyright © 2026 The Tekton Authors.
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

package taskrun

import (
	"testing"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		name   string
		trs    []Run
		filter []string
		want   int
	}{
		{name: "empty filter returns all", trs: []Run{{Task: "build"}, {Task: "test"}}, filter: nil, want: 2},
		{name: "exact match", trs: []Run{{Task: "build"}, {Task: "test"}}, filter: []string{"build"}, want: 1},
		{name: "no match", trs: []Run{{Task: "build"}}, filter: []string{"nonexistent"}, want: 0},
		{name: "PinP exact", trs: []Run{{Task: "call-child" + ChildTaskSeparator + "greet"}}, filter: []string{"call-child" + ChildTaskSeparator + "greet"}, want: 1},
		{name: "PinP parent prefix", trs: []Run{{Task: "call-child" + ChildTaskSeparator + "greet"}}, filter: []string{"call-child"}, want: 1},
		{name: "PinP child suffix", trs: []Run{{Task: "call-child" + ChildTaskSeparator + "greet"}}, filter: []string{"greet"}, want: 1},
		{name: "PinP deep nesting", trs: []Run{{Task: "a" + ChildTaskSeparator + "b" + ChildTaskSeparator + "c"}}, filter: []string{"c"}, want: 1},
		{name: "PinP mixed direct and child", trs: []Run{{Task: "build"}, {Task: "deploy" + ChildTaskSeparator + "greet"}, {Task: "test"}}, filter: []string{"build", "greet"}, want: 2},
		{name: "PinP multiple filters", trs: []Run{{Task: "a" + ChildTaskSeparator + "x"}, {Task: "b" + ChildTaskSeparator + "y"}, {Task: "c" + ChildTaskSeparator + "z"}}, filter: []string{"x", "y"}, want: 2},
		{name: "partial prefix does not match", trs: []Run{{Task: "call-child" + ChildTaskSeparator + "greet"}}, filter: []string{"child"}, want: 0},
		{name: "filter with separator exact only", trs: []Run{{Task: "a" + ChildTaskSeparator + "b" + ChildTaskSeparator + "c"}}, filter: []string{"a" + ChildTaskSeparator + "b"}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Filter(tt.trs, tt.filter)
			if len(result) != tt.want {
				t.Errorf("expected %d results, got %d", tt.want, len(result))
			}
		})
	}
}

func TestIsFiltered(t *testing.T) {
	if IsFiltered(Run{Task: "build"}, []string{"build"}) {
		t.Error("expected IsFiltered to be false for matching task")
	}
	if !IsFiltered(Run{Task: "build"}, []string{"test"}) {
		t.Error("expected IsFiltered to be true for non-matching task")
	}
	if IsFiltered(Run{Task: "call-child" + ChildTaskSeparator + "greet"}, []string{"greet"}) {
		t.Error("expected IsFiltered to be false for suffix-matching task")
	}
}
