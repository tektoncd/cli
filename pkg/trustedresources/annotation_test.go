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
	}, {
		name: "existing signature",
		doc: `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: example
  annotations:
    tekton.dev/displayName: example
    tekton.dev/signature: MEUCIQDvbGRzaWduYXR1cmU=
spec:
  steps: []
`,
		want: `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: example
  annotations:
    tekton.dev/displayName: example
    tekton.dev/signature: ` + signatureValue + `
spec:
  steps: []
`,
	}}

	for _, tc := range tcs {
		for eolName, eol := range map[string]string{"LF": "\n", "CRLF": "\r\n"} {
			t.Run(tc.name+" "+eolName, func(t *testing.T) {
				doc := strings.ReplaceAll(tc.doc, "\n", eol)
				want := strings.ReplaceAll(tc.want, "\n", eol)
				got, err := insertAnnotation([]byte(doc), SignatureAnnotation, signatureValue)
				if err != nil {
					t.Fatalf("insertAnnotation() got err %v", err)
				}
				if string(got) != want {
					t.Errorf("insertAnnotation() mismatch\ngot:\n%q\nwant:\n%q", got, want)
				}
			})
		}
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

	// Removing the signature line must give back the input exactly, line order included.
	line := "    " + SignatureAnnotation + ": " + signatureValue + "\n"
	if !strings.Contains(string(got), line) {
		t.Fatalf("expected the signature line %q, but got:\n%s", line, got)
	}
	if rest := strings.Replace(string(got), line, "", 1); rest != doc {
		t.Errorf("insertAnnotation() changed the document beyond the signature line:\n%s", rest)
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
		name: "existing signature as a block scalar",
		doc:  "apiVersion: tekton.dev/v1\nkind: Task\nmetadata:\n  annotations:\n    tekton.dev/signature: >-\n      MEUCIQ\n",
	}, {
		name: "existing signature continued on the next line",
		doc:  "apiVersion: tekton.dev/v1\nkind: Task\nmetadata:\n  annotations:\n    tekton.dev/signature: MEUC\n      IQ\n",
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
