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

package pipeline

import (
	"io"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	pipelinetest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_Pipeline_Apply(t *testing.T) {

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
			command:     []string{"apply", "--from", "./testdata/pipeline.yaml", "-n", "invalid"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "namespaces \"invalid\" not found",
		},
		{
			name:        "Create pipeline successfully",
			command:     []string{"apply", "--from", "./testdata/pipeline.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "Pipeline created: test-pipeline\n",
		},
		{
			name:        "Update pipeline successfully",
			command:     []string{"apply", "-f", "./testdata/pipeline.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "Pipeline updated: test-pipeline\n",
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
			command:     []string{"apply", "-f", "./testdata/pipeline.txt", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "inavlid file format for ./testdata/pipeline.txt: .yaml or .yml file extension and format required",
		},
		{
			name:        "Mismatched resource file",
			command:     []string{"apply", "-f", "./testdata/pipelinerun.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "provided kind PipelineRun instead of kind Pipeline",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube}
			pipeline := Command(p)

			if tp.inputStream != nil {
				pipeline.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(pipeline, tp.command...)
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

func Test_Pipeline_Apply_Update(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}
	pipeline := Command(p)

	//create Pipeline
	_, err := test.ExecuteCommand(pipeline, "apply", "-f", "./testdata/pipeline.yaml", "-n", "ns")
	if err != nil {
		t.Errorf("Error from creating Pipeline: %s", err)
	}

	//describe newly created Pipeline
	output, err := test.ExecuteCommand(pipeline, "desc", "test-pipeline")
	if err != nil {
		t.Errorf("Error from describing Pipeline: %s", err)
	}
	expected := `Name:   test-pipeline

Resources
NAME          TYPE
source-repo   git
web-image     image

Params
No params

Tasks
NAME                 TASKREF                              RUNAFTER
build-skaffold-web   build-docker-image-from-git-source   []
deploy-web           deploy-using-kubectl                 []

Pipelineruns
No pipelineruns
`

	test.AssertOutput(t, expected, output)

	//update Pipeline
	_, err = test.ExecuteCommand(pipeline, "apply", "-f", "./testdata/pipeline-updated.yaml", "-n", "ns")
	if err != nil {
		t.Errorf("Error from updating Pipeline: %s", err)
	}

	//describe updated Pipeline
	output, err = test.ExecuteCommand(pipeline, "desc", "test-pipeline")
	if err != nil {
		t.Errorf("Error from describing Pipeline: %s", err)
	}
	expected = `Name:   test-pipeline

Resources
NAME                 TYPE
source-repo-rename   git
web-image-update     image

Params
No params

Tasks
NAME                 TASKREF                              RUNAFTER
build-skaffold-web   build-docker-image-from-git-source   []
deploy-web           deploy-using-kubectl                 []

Pipelineruns
No pipelineruns
`

	test.AssertOutput(t, expected, output)
}
