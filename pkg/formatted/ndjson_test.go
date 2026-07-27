// Copyright © 2024 The Tekton Authors.
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
	if err := formatted.PrintNDJSON(&buf, makePRList(), nil); err != nil {
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
	if err := formatted.PrintNDJSON(&buf, makePRList(), []string{"metadata.name", "metadata.namespace"}); err != nil {
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
	if err := formatted.PrintNDJSON(&buf, makePRList(), []string{"metadata"}); err != nil {
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
	if err := formatted.PrintNDJSON(&buf, makePRList(), []string{"metadata.name", "does.not.exist"}); err != nil {
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
	if err := formatted.PrintNDJSON(&buf, empty, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for empty list, got %q", buf.String())
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
