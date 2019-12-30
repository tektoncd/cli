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
	pipelinetest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_Task_Apply(t *testing.T) {

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seeds := make([]pipelinetest.Clients, 0)
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	//seeds[0]
	seeds = append(seeds, cs)

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
			command:     []string{"apply", "--from", "./testdata/task.yaml", "-n", "invalid"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "namespaces \"invalid\" not found",
		},
		{
			name:        "Create task successfully",
			command:     []string{"apply", "--from", "./testdata/task.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "Task created: test-task\n",
		},
		{
			name:        "Update task successfully",
			command:     []string{"apply", "-f", "./testdata/task.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "Task updated: test-task\n",
		},
		{
			name:        "Filename does not exist",
			command:     []string{"apply", "-f", "./testdata/notexist.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "open ./testdata/notexist.yaml: no such file or directory",
		},
		{
			name:        "Unsupported file type",
			command:     []string{"apply", "-f", "./testdata/task.txt", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "inavlid file format for ./testdata/task.txt: .yaml or .yml file extension and format required",
		},
		{
			name:        "Mismatched resource file",
			command:     []string{"apply", "-f", "./testdata/taskrun.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "provided kind TaskRun instead of kind Task",
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

func TestTaskApply_Update(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}
	task := Command(p)

	//create Task
	_, err := test.ExecuteCommand(task, "apply", "-f", "./testdata/task.yaml", "-n", "ns")
	if err != nil {
		t.Errorf("Error from creating Task: %s", err)
	}

	//describe newly created Task
	output, err := test.ExecuteCommand(task, "desc", "test-task")
	if err != nil {
		t.Errorf("Error from describing Task: %s", err)
	}
	expected := `Name:        test-task
Namespace:   ns

Input Resources
NAME            TYPE
docker-source   git

Output Resources
NAME         TYPE
builtImage   image

Params
NAME               TYPE     DEFAULT VALUE
pathToDockerFile   string   /workspace/docker-source/Dockerfile
pathToContext      string   /workspace/docker-source

Steps
NAME
build-and-push

Taskruns
No taskruns
`

	test.AssertOutput(t, expected, output)

	//update Task
	_, err = test.ExecuteCommand(task, "apply", "-f", "./testdata/task-updated.yaml", "-n", "ns")
	if err != nil {
		t.Errorf("Error from updating Task: %s", err)
	}

	//describe updated Task
	output, err = test.ExecuteCommand(task, "desc", "test-task")
	if err != nil {
		t.Errorf("Error from describing Task: %s", err)
	}
	expected = `Name:        test-task
Namespace:   ns

Input Resources
NAME            TYPE
docker-source   git

Output Resources
NAME         TYPE
builtImage   image

Params
NAME               TYPE     DEFAULT VALUE
pathToDockerFile   string   /workspace/docker-source/Dockerfile
pathToContext      string   /workspace/docker-source

Steps
NAME
build-and-push-v0.13.0

Taskruns
No taskruns
`

	test.AssertOutput(t, expected, output)
}
