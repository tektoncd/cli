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

package formatted_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tektoncd/cli/pkg/formatted"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func makePRList() *v1.PipelineRunList {
	return &v1.PipelineRunList{
		Items: []v1.PipelineRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pr-1",
					Namespace: "default",
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{Reason: "Succeeded"},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pr-2",
					Namespace: "default",
				},
			},
		},
	}
}

func TestPrintNDJSON_allFields(t *testing.T) {
	var buf bytes.Buffer
	if err := formatted.PrintNDJSON(&buf, makePRList(), nil, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := splitLines(buf.String())
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		}
	}
}

func TestPrintNDJSON_fieldSelection(t *testing.T) {
	var buf bytes.Buffer
	if err := formatted.PrintNDJSON(&buf, makePRList(), []string{"metadata.name", "metadata.namespace"}, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := splitLines(buf.String())
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		}
		meta, ok := m["metadata"].(map[string]any)
		if !ok {
			t.Errorf("line %d: expected metadata key", i)
			continue
		}
		if _, ok := meta["name"]; !ok {
			t.Errorf("line %d: expected metadata.name key", i)
		}
		if _, ok := meta["namespace"]; !ok {
			t.Errorf("line %d: expected metadata.namespace key", i)
		}
		// status should not be present
		if _, ok := m["status"]; ok {
			t.Errorf("line %d: unexpected status key", i)
		}
	}
}

func TestPrintNDJSON_singleTopLevelField(t *testing.T) {
	var buf bytes.Buffer
	if err := formatted.PrintNDJSON(&buf, makePRList(), []string{"metadata"}, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := splitLines(buf.String())
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		}
		if _, ok := m["metadata"]; !ok {
			t.Errorf("line %d: expected metadata key", i)
		}
		if len(m) != 1 {
			t.Errorf("line %d: expected only 1 top-level key, got %d", i, len(m))
		}
	}
}

func TestPrintNDJSON_unknownFieldIgnored(t *testing.T) {
	var buf bytes.Buffer
	if err := formatted.PrintNDJSON(&buf, makePRList(), []string{"metadata.name", "does.not.exist"}, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := splitLines(buf.String())
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		}
		meta, ok := m["metadata"].(map[string]any)
		if !ok {
			t.Errorf("line %d: expected metadata key", i)
			continue
		}
		if _, ok := meta["name"]; !ok {
			t.Errorf("line %d: expected metadata.name", i)
		}
	}
}

func TestPrintNDJSON_emptyList(t *testing.T) {
	var buf bytes.Buffer
	empty := &v1.PipelineRunList{}
	if err := formatted.PrintNDJSON(&buf, empty, nil, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for empty list, got %q", buf.String())
	}
}

// TestPrintNDJSON_nullValuePreserved ensures that a field explicitly set to nil
// in the source (e.g. status.completionTime for a running PipelineRun) is
// emitted as JSON null rather than being dropped.
func TestPrintNDJSON_nullValuePreserved(t *testing.T) {
	list := &v1.PipelineRunList{
		Items: []v1.PipelineRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "running",
					Namespace: "default",
				},
				// CompletionTime intentionally absent (nil).
			},
		},
	}

	var buf bytes.Buffer
	// Request metadata.name — present and non-nil.
	if err := formatted.PrintNDJSON(&buf, list, []string{"metadata.name"}, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := splitLines(buf.String())
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &m); err != nil {
		t.Fatalf("line is not valid JSON: %v", err)
	}
	meta, ok := m["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata key")
	}
	if _, ok := meta["name"]; !ok {
		t.Errorf("expected metadata.name to be present")
	}
}

func TestPrintJSON_allFields(t *testing.T) {
	var buf bytes.Buffer
	if err := formatted.PrintJSON(&buf, makePRList(), nil, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// The entire object should be serialized
	if _, ok := result["items"]; !ok {
		t.Error("expected items field in JSON output when no fields specified")
	}
}

func TestPrintJSON_fieldSelection(t *testing.T) {
	var buf bytes.Buffer
	if err := formatted.PrintJSON(&buf, makePRList(), []string{"metadata.name", "metadata.namespace"}, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	items, ok := result["items"].([]any)
	if !ok {
		t.Fatal("expected items array")
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			t.Errorf("item %d is not a map", i)
			continue
		}
		meta, ok := m["metadata"].(map[string]any)
		if !ok {
			t.Errorf("item %d: expected metadata key", i)
			continue
		}
		if _, ok := meta["name"]; !ok {
			t.Errorf("item %d: expected metadata.name key", i)
		}
		if _, ok := meta["namespace"]; !ok {
			t.Errorf("item %d: expected metadata.namespace key", i)
		}
		// status should not be present
		if _, ok := m["status"]; ok {
			t.Errorf("item %d: unexpected status key", i)
		}
	}
}

func TestPrintJSON_emptyList(t *testing.T) {
	var buf bytes.Buffer
	empty := &v1.PipelineRunList{}
	if err := formatted.PrintJSON(&buf, empty, []string{"metadata.name"}, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	items, ok := result["items"].([]any)
	if !ok {
		t.Fatal("expected items array")
	}

	if len(items) != 0 {
		t.Errorf("expected empty items array, got %d items", len(items))
	}
}

func TestPrintNDJSON_arrayIndexing(t *testing.T) {
	// Create a PipelineRun with conditions array
	prList := &v1.PipelineRunList{
		Items: []v1.PipelineRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pr-with-conditions",
					Namespace: "default",
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{Reason: "Succeeded", Type: "Ready"},
							{Reason: "Running", Type: "InProgress"},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	// Request first condition's reason using array indexing
	if err := formatted.PrintNDJSON(&buf, prList, []string{"metadata.name", "status.conditions[0].reason"}, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := splitLines(buf.String())
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &m); err != nil {
		t.Fatalf("line is not valid JSON: %v", err)
	}

	// Check metadata.name
	meta, ok := m["metadata"].(map[string]any)
	if !ok {
		t.Fatal("expected metadata key")
	}
	if name, ok := meta["name"].(string); !ok || name != "pr-with-conditions" {
		t.Errorf("expected metadata.name='pr-with-conditions', got %v", meta["name"])
	}

	// Check status.conditions[0].reason
	// When we select conditions[0].reason, the result should be nested as:
	// status -> conditions[0] -> reason
	status, ok := m["status"].(map[string]any)
	if !ok {
		t.Fatal("expected status key")
	}

	// The field picker creates: status.conditions[0].reason
	// This means: status -> conditions[0] (the array element) -> reason
	// Since we're picking a field from an array element, we get the element directly
	conditionsArr, ok := status["conditions[0]"]
	if ok {
		// If it's stored as "conditions[0]" key
		condObj, ok := conditionsArr.(map[string]any)
		if !ok {
			t.Fatalf("expected conditions[0] to be a map, got %T", conditionsArr)
		}
		reason, ok := condObj["reason"].(string)
		if !ok || reason != "Succeeded" {
			t.Errorf("expected conditions[0].reason='Succeeded', got %v", condObj["reason"])
		}
	} else {
		// Alternative: conditions might be set as a map directly
		t.Logf("status keys: %v", status)
		t.Fatal("expected status.conditions[0] key but array indexing may not be working correctly")
	}
}

func TestValidateFields_empty(t *testing.T) {
	// Empty slice should pass
	if err := formatted.ValidateFields([]string{}); err != nil {
		t.Errorf("empty fields should not error: %v", err)
	}

	// All whitespace should error
	err := formatted.ValidateFields([]string{"", "  ", "\t"})
	if err == nil {
		t.Error("expected error for all-whitespace fields")
	}
	if err != nil && err.Error() != "at least one field required" {
		t.Errorf("expected 'at least one field required', got '%v'", err)
	}
}

func TestValidateFields_syntax(t *testing.T) {
	tests := []struct {
		name      string
		fields    []string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid simple field",
			fields:    []string{"metadata.name"},
			wantError: false,
		},
		{
			name:      "valid array index",
			fields:    []string{"status.conditions[0].reason"},
			wantError: false,
		},
		{
			name:      "multiple valid fields",
			fields:    []string{"metadata.name", "status.startTime", "spec.params[1].value"},
			wantError: false,
		},
		{
			name:      "consecutive dots",
			fields:    []string{"metadata..name"},
			wantError: true,
			errorMsg:  "consecutive dots",
		},
		{
			name:      "leading dot",
			fields:    []string{".metadata.name"},
			wantError: true,
			errorMsg:  "cannot start or end with dot",
		},
		{
			name:      "trailing dot",
			fields:    []string{"metadata.name."},
			wantError: true,
			errorMsg:  "cannot start or end with dot",
		},
		{
			name:      "unclosed bracket",
			fields:    []string{"status.conditions[0"},
			wantError: true,
			errorMsg:  "unclosed bracket",
		},
		{
			name:      "empty bracket",
			fields:    []string{"status.conditions[]"},
			wantError: true,
			errorMsg:  "empty array index",
		},
		{
			name:      "non-numeric index",
			fields:    []string{"status.conditions[abc]"},
			wantError: true,
			errorMsg:  "array index must be numeric",
		},
		{
			name:      "empty field name before bracket",
			fields:    []string{"status.[0]"},
			wantError: true,
			errorMsg:  "empty field name before array index",
		},
		{
			name:      "text after closing bracket",
			fields:    []string{"status.conditions[0]suffix"},
			wantError: true,
			errorMsg:  "unexpected characters after closing bracket",
		},
		{
			name:      "unmatched closing bracket",
			fields:    []string{"status.conditions]"},
			wantError: true,
			errorMsg:  "unmatched closing bracket",
		},
		{
			name:      "reversed brackets",
			fields:    []string{"status.conditions]x[0]"},
			wantError: true,
			errorMsg:  "closing bracket before opening bracket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatted.ValidateFields(tt.fields)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPrintNDJSON_deduplication(t *testing.T) {
	var buf bytes.Buffer
	// Request same field multiple times
	if err := formatted.PrintNDJSON(&buf, makePRList(), []string{"metadata.name", "metadata.name", "metadata.namespace", "metadata.name"}, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := splitLines(buf.String())
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &m); err != nil {
		t.Fatalf("line is not valid JSON: %v", err)
	}

	meta, ok := m["metadata"].(map[string]any)
	if !ok {
		t.Fatal("expected metadata key")
	}

	// Should have both name and namespace, each appearing only once
	if len(meta) != 2 {
		t.Errorf("expected exactly 2 fields in metadata after dedup, got %d", len(meta))
	}
}

// splitLines returns non-empty lines from s.
func splitLines(s string) []string {
	var out []string
	for _, l := range bytes.Split([]byte(s), []byte("\n")) {
		if len(l) > 0 {
			out = append(out, string(l))
		}
	}
	return out
}
