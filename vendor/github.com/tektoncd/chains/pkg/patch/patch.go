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

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TektonObject interface to get GVK information needed for server-side apply
type TektonObject interface {
	GetName() string
	GetNamespace() string
	GetGVK() string // Returns GroupVersionKind as "group/version/kind" string
}

// GetAnnotationsPatch returns patch bytes that can be used with kubectl patch
func GetAnnotationsPatch(newAnnotations map[string]string, obj TektonObject) ([]byte, error) {
	// Get GVK using the TektonObject interface method (more reliable than runtime.Object)
	gvkStr := obj.GetGVK()
	if gvkStr == "" {
		return nil, fmt.Errorf("unable to determine GroupVersionKind for object %s/%s", obj.GetNamespace(), obj.GetName())
	}

	// Parse the string format "group/version/kind"
	parts := strings.Split(gvkStr, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid GVK format: %s", gvkStr)
	}
	apiVersion := parts[0] + "/" + parts[1]
	kind := parts[2]

	// For server-side apply, we need to create a structured patch with metadata
	p := serverSideApplyPatch{
		APIVersion: apiVersion,
		Kind:       kind,
		Metadata: serverSideApplyMetadata{
			Name:        obj.GetName(),
			Namespace:   obj.GetNamespace(),
			Annotations: newAnnotations,
		},
	}
	return json.Marshal(p)
}

// These are used to get proper json formatting for server-side apply
type serverSideApplyPatch struct {
	APIVersion string                  `json:"apiVersion"`
	Kind       string                  `json:"kind"`
	Metadata   serverSideApplyMetadata `json:"metadata"`
}

type serverSideApplyMetadata struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
