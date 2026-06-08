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

package taskrun

import (
	"testing"
)

func TestFilter_Empty_ReturnsAll(t *testing.T) {
	trs := []Run{{Task: "build"}, {Task: "test"}}
	result := Filter(trs, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
}

func TestFilter_ExactMatch(t *testing.T) {
	trs := []Run{{Task: "build"}, {Task: "test"}}
	result := Filter(trs, []string{"build"})
	if len(result) != 1 || result[0].Task != "build" {
		t.Errorf("expected [build], got %v", result)
	}
}

func TestFilter_NoMatch(t *testing.T) {
	trs := []Run{{Task: "build"}}
	result := Filter(trs, []string{"nonexistent"})
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestFilter_PinP_Exact(t *testing.T) {
	trs := []Run{{Task: "call-child" + ChildTaskSeparator + "greet"}}
	result := Filter(trs, []string{"call-child" + ChildTaskSeparator + "greet"})
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

func TestFilter_PinP_ParentPrefix(t *testing.T) {
	trs := []Run{{Task: "call-child" + ChildTaskSeparator + "greet"}}
	result := Filter(trs, []string{"call-child"})
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

func TestFilter_PinP_ChildSuffix(t *testing.T) {
	trs := []Run{{Task: "call-child" + ChildTaskSeparator + "greet"}}
	result := Filter(trs, []string{"greet"})
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

func TestFilter_PinP_DeepNesting(t *testing.T) {
	trs := []Run{{Task: "a" + ChildTaskSeparator + "b" + ChildTaskSeparator + "c"}}
	result := Filter(trs, []string{"c"})
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

func TestFilter_PinP_MixedDirectAndChild(t *testing.T) {
	trs := []Run{{Task: "build"}, {Task: "deploy" + ChildTaskSeparator + "greet"}, {Task: "test"}}
	result := Filter(trs, []string{"build", "greet"})
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
}

func TestFilter_PinP_MultipleFilters(t *testing.T) {
	trs := []Run{{Task: "a" + ChildTaskSeparator + "x"}, {Task: "b" + ChildTaskSeparator + "y"}, {Task: "c" + ChildTaskSeparator + "z"}}
	result := Filter(trs, []string{"x", "y"})
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
}

func TestFilter_PinP_PrefixDoesNotMatchPartial(t *testing.T) {
	// "child" should NOT match "call-child>greet" (partial prefix)
	trs := []Run{{Task: "call-child" + ChildTaskSeparator + "greet"}}
	result := Filter(trs, []string{"child"})
	if len(result) != 0 {
		t.Errorf("expected 0 results for partial prefix match, got %d", len(result))
	}
}

func TestFilter_PinP_FilterWithSeparatorExactOnly(t *testing.T) {
	// filter value containing ChildTaskSeparator should only match exact
	trs := []Run{{Task: "a" + ChildTaskSeparator + "b" + ChildTaskSeparator + "c"}}
	result := Filter(trs, []string{"a" + ChildTaskSeparator + "b"})
	if len(result) != 0 {
		t.Errorf("expected 0 results (exact only for filter with >), got %d", len(result))
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
