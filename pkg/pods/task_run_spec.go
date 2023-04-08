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
	"net/http"

	"github.com/tektoncd/cli/pkg/file"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/yaml"
)

type TaskRunSpec = []v1beta1.PipelineTaskRunSpec

func ParseTaskRunSpec(httpClient http.Client, taskRunSpecLocation string, validate file.TypeValidator, errorMsg error) (TaskRunSpec, error) {
	taskRunSpec := TaskRunSpec{}
	b, err := file.LoadFileContent(httpClient, taskRunSpecLocation, validate, errorMsg)
	if err != nil {
		return taskRunSpec, err
	}

	if err := yaml.UnmarshalStrict(b, &taskRunSpec); err != nil {
		return taskRunSpec, err
	}

	return taskRunSpec, nil
}
