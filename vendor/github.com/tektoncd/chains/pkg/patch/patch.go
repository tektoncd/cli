// Copyright 2021 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package patch

import "encoding/json"

// GetAnnotationsPatch returns patch bytes that can be used with kubectl patch
func GetAnnotationsPatch(newAnnotations map[string]string) ([]byte, error) {
	p := patch{
		Metadata: metadata{
			Annotations: newAnnotations,
		},
	}
	return json.Marshal(p)
}

// These are used to get proper json formatting
type patch struct {
	Metadata metadata `json:"metadata,omitempty"`
}
type metadata struct {
	Annotations map[string]string `json:"annotations,omitempty"`
}
