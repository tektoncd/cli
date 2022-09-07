// Copyright © 2020 The Tekton Authors.
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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	"sigs.k8s.io/yaml"
)

func ParsePodTemplate(httpClient http.Client, podTemplateLocation string, validate file.TypeValidator, errorMsg error) (pod.PodTemplate, error) {
	podTemplate := pod.PodTemplate{}
	b, err := file.LoadFileContent(httpClient, podTemplateLocation, validate, errorMsg)
	if err != nil {
		return podTemplate, err
	}

	if err := yaml.UnmarshalStrict(b, &podTemplate); err != nil {
		return podTemplate, err
	}

	return podTemplate, nil
}
