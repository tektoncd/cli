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

package task

import (
	"io"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_Task_Create(t *testing.T) {

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seeds := make([]pipelinetest.Clients, 0)
	for i := 0; i < 1; i++ {
		cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
		seeds = append(seeds, cs)
	}

	testParams := []struct {
		name        string
		command     []string
		input       pipelinetest.Clients
		inputStream io.Reader
		wantError   bool
		want        string
	}{
		{
			name:        "Invalid namespace",
			command:     []string{"create", "--from", "./testdata/task.yaml", "-n", "invalid"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "namespaces \"invalid\" not found",
		},
		{
			name:        "Create task successfully",
			command:     []string{"create", "--from", "./testdata/task.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "Command \"create\" is deprecated, tkn task create will be removed in v1.0.0. Use kubectl create -f instead. Learn more here: https://github.com/tektoncd/cli/issues/816\nTask created: test-task\n",
		},
		{
			name:        "Filename does not exist",
			command:     []string{"create", "-f", "./testdata/notexist.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "open ./testdata/notexist.yaml: no such file or directory",
		},
		{
			name:        "Unsupported file type",
			command:     []string{"create", "-f", "./testdata/task.txt", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "invalid file format for ./testdata/task.txt: .yaml or .yml file extension and format required",
		},
		{
			name:        "Mismatched resource file",
			command:     []string{"create", "-f", "./testdata/taskrun.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "provided kind TaskRun instead of kind Task",
		},
		{
			name:        "Existing task",
			command:     []string{"create", "-f", "./testdata/task.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "failed to create task \"test-task\": tasks.tekton.dev \"test-task\" already exists",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube}
			task := Command(p)

			if tp.inputStream != nil {
				task.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(task, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("Error expected here")
				} else {
					test.AssertOutput(t, tp.want, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}
