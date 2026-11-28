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

package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tektoncd/cli/pkg/exitcode"
	"github.com/tektoncd/cli/pkg/test"
)

const (
	validPipeline = `apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: build
spec:
  tasks:
    - name: hello
      taskRef:
        name: hello
`
	validTask = `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: hello
spec:
  steps:
    - name: hello
      image: alpine
      script: echo hello
`
	invalidTask = `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: nosteps
spec:
  params:
    - name: someparam
`
)

func writeFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestValidate(t *testing.T) {
	dir := t.TempDir()
	valid := writeFile(t, dir, "valid.yaml", validTask+"---\n"+validPipeline)
	invalid := writeFile(t, dir, "invalid.yaml", invalidTask)

	tests := []struct {
		name     string
		args     []string
		stdin    string
		wantOut  string
		wantErrs []string
		wantCode int
	}{
		{
			name:    "valid file",
			args:    []string{"-f", valid},
			wantOut: "All Tekton resources in 1 file(s) are valid",
		},
		{
			name:     "invalid file",
			args:     []string{"-f", invalid},
			wantErrs: []string{"invalid.yaml: ", "Task nosteps is invalid"},
			wantCode: exitcode.InvalidInput,
		},
		{
			name:     "every file is checked",
			args:     []string{"-f", invalid, "-f", valid, "-f", filepath.Join(dir, "missing.yaml")},
			wantErrs: []string{"Task nosteps is invalid", "failed to read", "missing.yaml"},
			wantCode: exitcode.GeneralError,
		},
		{
			name:    "stdin",
			args:    []string{"-f", "-"},
			stdin:   validTask,
			wantOut: "All Tekton resources in 1 file(s) are valid",
		},
		{
			name:     "invalid stdin",
			args:     []string{"-f", "-"},
			stdin:    invalidTask,
			wantErrs: []string{"stdin: ", "Task nosteps is invalid"},
			wantCode: exitcode.InvalidInput,
		},
		{
			name:     "no files",
			wantErrs: []string{"at least one file is required"},
			wantCode: exitcode.InvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Command(&test.Params{})
			c.SetIn(strings.NewReader(tt.stdin))
			out, err := test.ExecuteCommand(c, tt.args...)
			if len(tt.wantErrs) > 0 {
				if err == nil {
					t.Fatalf("expected an error, got output %q", out)
				}
				for _, want := range tt.wantErrs {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("error %q does not contain %q", err, want)
					}
				}
				if got := exitcode.CodeFrom(err); got != tt.wantCode {
					t.Errorf("exit code = %d, want %d", got, tt.wantCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(out, tt.wantOut) {
				t.Errorf("output %q does not contain %q", out, tt.wantOut)
			}
		})
	}
}
