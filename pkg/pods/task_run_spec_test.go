// Copyright © 2023 The Tekton Authors.
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

package pods

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/tektoncd/cli/pkg/file"
	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func getTestTaskRunSpec() TaskRunSpec {
	runAsNonRoot := true
	runAsUser := int64(1001)

	return TaskRunSpec{
		v1beta1.PipelineTaskRunSpec{
			PipelineTaskName: "first-create-file",
			TaskPodTemplate: &pod.PodTemplate{
				ImagePullSecrets: nil,
				HostNetwork:      false,
				SchedulerName:    "SchedulerName",
				SecurityContext: &corev1.PodSecurityContext{
					RunAsNonRoot: &runAsNonRoot,
					RunAsUser:    &runAsUser,
				},
			},
		},
	}
}

func TestTaskRunSpec_Local_File(t *testing.T) {
	httpClient := *http.DefaultClient
	podTemplateLocation := "./testdata/taskrunspec.yaml"

	podTemplate, err := ParseTaskRunSpec(httpClient, podTemplateLocation, file.IsYamlFile(), fmt.Errorf("invalid file format for %s: .yaml or .yml file extension and format required", podTemplateLocation))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, getTestTaskRunSpec(), podTemplate)
}

func TestTaskRunSpec_Local_File_Typo(t *testing.T) {
	httpClient := *http.DefaultClient
	podTemplateLocation := "./testdata/taskrunspec-typo.yaml"

	_, err := ParseTaskRunSpec(httpClient, podTemplateLocation, file.IsYamlFile(), fmt.Errorf("invalid file format for %s: .yaml or .yml file extension and format required", podTemplateLocation))
	if err == nil {
		t.Fatalf("Expected error for local file typo, but error was nil")
	}

	expected := `error unmarshaling JSON: while decoding JSON: json: unknown field "ecurityContext"`
	test.AssertOutput(t, expected, err.Error())
}

func TestTaskRunSpec_Local_File_Not_YAML(t *testing.T) {
	httpClient := *http.DefaultClient
	podTemplateLocation := "./testdata/taskrunspec-not-yaml"

	_, err := ParseTaskRunSpec(httpClient, podTemplateLocation, file.IsYamlFile(), fmt.Errorf("invalid file format for %s: .yaml or .yml file extension and format required", podTemplateLocation))
	if err == nil {
		t.Fatalf("Expected error for local file typo, but error was nil")
	}

	expected := "invalid file format for ./testdata/taskrunspec-not-yaml: .yaml or .yml file extension and format required"
	test.AssertOutput(t, expected, err.Error())
}

func TestTaskRunSpec_Local_File_Not_Found(t *testing.T) {
	httpClient := *http.DefaultClient
	podTemplateLocation := "./testdata/not-exist.yaml"

	_, err := ParseTaskRunSpec(httpClient, podTemplateLocation, file.IsYamlFile(), fmt.Errorf("invalid file format for %s: .yaml or .yml file extension and format required", podTemplateLocation))
	if err == nil {
		t.Fatalf("Expected error for local file typo, but error was nil")
	}

	expected := "open ./testdata/not-exist.yaml: no such file or directory"
	test.AssertOutput(t, expected, err.Error())
}
