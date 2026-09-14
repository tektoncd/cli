/*
Copyright 2026 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package trustedresources

import (
	"strings"
	"testing"
)

const signatureValue = "MEUCIQD0aXNpZ25hdHVyZQ=="

func TestInsertAnnotation(t *testing.T) {
	tcs := []struct {
		name string
		doc  string
		want string
	}{{
		name: "existing annotations",
		doc: `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: example
  annotations:
    tekton.dev/displayName: example
spec:
  steps: []
`,
		want: `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: example
  annotations:
    tekton.dev/signature: ` + signatureValue + `
    tekton.dev/displayName: example
spec:
  steps: []
`,
	}, {
		name: "no annotations",
		doc: `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: example
spec:
  steps: []
`,
		want: `apiVersion: tekton.dev/v1
kind: Task
metadata:
  annotations:
    tekton.dev/signature: ` + signatureValue + `
  name: example
spec:
  steps: []
`,
	}, {
		name: "empty annotations mapping",
		doc: `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: example
  annotations: {}
spec:
  steps: []
`,
		want: `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: example
  annotations:
    tekton.dev/signature: ` + signatureValue + `
spec:
  steps: []
`,
	}, {
		// The insertion point is a line number, so anything that shifts the
		// body down has to be accounted for.
		name: "leading document marker and comment",
		doc: `---
# a task
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: example
  annotations:
    tekton.dev/displayName: example
spec:
  steps: []
`,
		want: `---
# a task
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: example
  annotations:
    tekton.dev/signature: ` + signatureValue + `
    tekton.dev/displayName: example
spec:
  steps: []
`,
	}, {
		name: "annotations key with nothing under it",
		doc: `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: example
  annotations:
spec:
  steps: []
`,
		want: `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: example
  annotations:
    tekton.dev/signature: ` + signatureValue + `
spec:
  steps: []
`,
	}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := insertAnnotation([]byte(tc.doc), SignatureAnnotation, signatureValue)
			if err != nil {
				t.Fatalf("insertAnnotation() got err %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("insertAnnotation() mismatch\ngot:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

// TestInsertAnnotationPreservesDocument is the regression test for the reported
// behaviour: signing must not reformat anything it was not asked to change.
// Re-serializing a parsed document is what used to break this, and a folded
// scalar is the case that survives a naive round trip the least: its line
// breaks are collapsed even though the value is unchanged.
func TestInsertAnnotationPreservesDocument(t *testing.T) {
	doc := `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: example
  labels:
    app.kubernetes.io/version: "1.0.0"
  annotations:
    tekton.dev/displayName: example
spec:
  description: >-
    A folded
    description
  params:
    - name: flags
      default: "-v"
  steps:
    - name: run
      image: alpine
      script: |
        echo hi
`

	got, err := insertAnnotation([]byte(doc), SignatureAnnotation, signatureValue)
	if err != nil {
		t.Fatalf("insertAnnotation() got err %v", err)
	}

	added, removed := lineDiff(doc, string(got))
	if len(removed) != 0 {
		t.Errorf("expected no lines to be removed, but got %q", removed)
	}
	want := "    " + SignatureAnnotation + ": " + signatureValue
	if len(added) != 1 || added[0] != want {
		t.Errorf("expected exactly one added line %q, but got %q", want, added)
	}
}

func TestInsertAnnotationErrors(t *testing.T) {
	tcs := []struct {
		name string
		doc  string
	}{{
		name: "no metadata",
		doc:  "apiVersion: tekton.dev/v1\nkind: Task\nspec:\n  steps: []\n",
	}, {
		name: "flow style metadata",
		doc:  "apiVersion: tekton.dev/v1\nkind: Task\nmetadata: {name: example}\n",
	}, {
		name: "flow style annotations with entries",
		doc:  "apiVersion: tekton.dev/v1\nkind: Task\nmetadata:\n  annotations: {a: b}\n",
	}, {
		name: "empty document",
		doc:  "",
	}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := insertAnnotation([]byte(tc.doc), SignatureAnnotation, signatureValue); err == nil {
				t.Error("expected insertAnnotation to return an error, but got none")
			}
		})
	}
}

// lineDiff reports the lines present in only one of the two documents.
func lineDiff(before, after string) (added, removed []string) {
	count := map[string]int{}
	for _, l := range strings.Split(before, "\n") {
		count[l]++
	}
	for _, l := range strings.Split(after, "\n") {
		count[l]--
	}
	for _, l := range strings.Split(after, "\n") {
		if count[l] < 0 {
			added = append(added, l)
			count[l]++
		}
	}
	for _, l := range strings.Split(before, "\n") {
		if count[l] > 0 {
			removed = append(removed, l)
			count[l]--
		}
	}
	return added, removed
}
