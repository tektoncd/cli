// Copyright © 2022 The Tekton Authors.
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

package export

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func RemoveFieldForExport(obj *unstructured.Unstructured) error {
	content := obj.UnstructuredContent()

	// remove the status from pipelinerun and taskrun
	unstructured.RemoveNestedField(content, "status")

	// remove some metadata information of previous resource
	metadataFields := []string{
		"managedFields",
		"resourceVersion",
		"uid",
		"finalizers",
		"generation",
		"namespace",
		"creationTimestamp",
		"ownerReferences",
	}
	for _, field := range metadataFields {
		unstructured.RemoveNestedField(content, "metadata", field)
	}
	unstructured.RemoveNestedField(content, "metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration")

	// check if generateName exists and remove name if it does
	if _, exist, err := unstructured.NestedString(content, "metadata", "generateName"); err != nil {
		return err
	} else if exist {
		unstructured.RemoveNestedField(content, "metadata", "name")
	}

	// remove the status from spec which are related to status
	specFields := []string{"status", "statusMessage"}
	for _, field := range specFields {
		unstructured.RemoveNestedField(content, "spec", field)
	}

	return nil
}
