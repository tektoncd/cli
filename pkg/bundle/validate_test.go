// Copyright © 2026 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package bundle

import (
	"context"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	validTask = `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: valid
spec:
  steps:
    - name: hello
      image: alpine
      script: echo hello
`
	taskWithoutSteps = `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: nosteps
spec:
  params:
    - name: someparam
`
	pipelineWithBadRunAfter = `apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: badorder
spec:
  tasks:
    - name: second
      runAfter: ["missing"]
      taskRef:
        name: valid
`
	taskWithoutName = `apiVersion: tekton.dev/v1
kind: Task
metadata:
  labels:
    app: test
spec:
  steps:
    - image: alpine
`
)

func TestValidateTektonObjects(t *testing.T) {
	tests := []struct {
		name     string
		contents []string
		wantErrs []string
	}{
		{
			name:     "valid task",
			contents: []string{validTask},
		},
		{
			name:     "task without steps",
			contents: []string{taskWithoutSteps},
			wantErrs: []string{"Task nosteps is invalid"},
		},
		{
			name:     "all problems are reported",
			contents: []string{validTask, taskWithoutSteps, pipelineWithBadRunAfter},
			wantErrs: []string{"Task nosteps is invalid", "Pipeline badorder is invalid"},
		},
		{
			name:     "objects in one document stream",
			contents: []string{validTask + "---\n" + taskWithoutSteps},
			wantErrs: []string{"Task nosteps is invalid"},
		},
		{
			name:     "object without a name",
			contents: []string{taskWithoutName},
			wantErrs: []string{"Task: kubernetes resources should have a name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTektonObjects(context.Background(), tt.contents)
			if len(tt.wantErrs) == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected errors containing %q", tt.wantErrs)
			}
			for _, want := range tt.wantErrs {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
			if strings.Contains(err.Error(), "Task valid") {
				t.Errorf("valid task was reported: %v", err)
			}
		})
	}
}

func TestDecodeErrorNamesDocument(t *testing.T) {
	err := decodeObjects(validTask+"---\napiVersion: v1\nkind: Pod\nmetadata:\n  name: p\n", func(*schema.GroupVersionKind, runtime.Object, []byte) error {
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "(document 2)") {
		t.Errorf("expected the error to name document 2, got %v", err)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short"); got != "short" {
		t.Errorf("truncate(short) = %q", got)
	}
	long := strings.Repeat("x", maxErrorSnippet+50)
	if got := truncate(long); len(got) != maxErrorSnippet+3 || !strings.HasSuffix(got, "...") {
		t.Errorf("truncate(long) = %q", got)
	}
}

func TestDecodeObjectsMissingKindOrVersionIsShort(t *testing.T) {
	filler := strings.Repeat("x", 5000)
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "missing kind",
			content: strings.Join([]string{"apiVersion: tekton.dev/v1", "metadata:", "  name: " + filler, ""}, "\n"),
			want:    "object 'kind' is missing",
		},
		{
			name:    "missing apiVersion",
			content: strings.Join([]string{"kind: Task", "metadata:", "  name: " + filler, ""}, "\n"),
			want:    "object 'apiVersion' is missing",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := decodeObjects(tt.content, func(*schema.GroupVersionKind, runtime.Object, []byte) error { return nil })
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not contain %q", err, tt.want)
			}
			if len(err.Error()) > 2*maxErrorSnippet+200 {
				t.Errorf("error is %d bytes, the manifest leaked into it", len(err.Error()))
			}
		})
	}
}

func TestValidateTektonObjectsStrictAndGenerateName(t *testing.T) {
	misspelled := `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: typo
spec:
  steps:
    - name: hello
      image: alpine
      workngDir: /tmp
`
	duplicate := `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: dup
spec:
  steps:
    - name: hello
      image: alpine
      image: busybox
`
	generated := `apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  generateName: release-
spec:
  pipelineRef:
    name: release
`
	generatedBad := `apiVersion: tekton.dev/v1
kind: Task
metadata:
  generateName: bad-
spec:
  params:
    - name: someparam
`
	unsupported := `apiVersion: v1
kind: Pod
metadata:
  name: p
`
	jsonTask := `{"apiVersion":"tekton.dev/v1","kind":"Task","metadata":{"name":"j"},"spec":{"steps":[{"name":"a","image":"alpine","workngDir":"/"}]}}`

	tests := []struct {
		name     string
		content  string
		wantErrs []string
		wantNot  []string
	}{
		{name: "misspelled optional field", content: misspelled, wantErrs: []string{"workngDir"}},
		{name: "duplicate key", content: duplicate, wantErrs: []string{"image"}},
		{name: "unknown field in json", content: jsonTask, wantErrs: []string{"workngDir"}},
		{name: "generateName only is accepted", content: generated},
		{name: "generateName only still gets its spec checked", content: generatedBad, wantErrs: []string{"Task bad-* is invalid"}, wantNot: []string{"should have a name"}},
		{
			name:     "later documents are still checked after an unsupported kind",
			content:  unsupported + "---\n" + taskWithoutSteps,
			wantErrs: []string{"(document 1)", "Task nosteps is invalid"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTektonObjects(context.Background(), []string{tt.content})
			if len(tt.wantErrs) == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected errors containing %q", tt.wantErrs)
			}
			for _, want := range tt.wantErrs {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
			for _, not := range tt.wantNot {
				if strings.Contains(err.Error(), not) {
					t.Errorf("error %q should not contain %q", err, not)
				}
			}
		})
	}
}

func TestBuildStillRequiresName(t *testing.T) {
	generated := `apiVersion: tekton.dev/v1
kind: Task
metadata:
  generateName: gen-
spec:
  steps:
    - name: hello
      image: alpine
`
	if err := decodeObjects(generated, func(_ *schema.GroupVersionKind, obj runtime.Object, _ []byte) error {
		_, err := getObjectName(obj)
		return err
	}); err == nil || !strings.Contains(err.Error(), "should have a name") {
		t.Errorf("expected the explicit name requirement, got %v", err)
	}
}
