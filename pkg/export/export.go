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
	// remove the status from pipelinerun and taskrun
	unstructured.RemoveNestedField(obj.UnstructuredContent(), "status")

	// remove some metadata information of previous resource
	unstructured.RemoveNestedField(obj.UnstructuredContent(), "metadata", "managedFields")
	unstructured.RemoveNestedField(obj.UnstructuredContent(), "metadata", "resourceVersion")
	unstructured.RemoveNestedField(obj.UnstructuredContent(), "metadata", "uid")
	unstructured.RemoveNestedField(obj.UnstructuredContent(), "metadata", "generation")
	unstructured.RemoveNestedField(obj.UnstructuredContent(), "metadata", "namespace")
	unstructured.RemoveNestedField(obj.UnstructuredContent(), "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(obj.UnstructuredContent(), "metadata", "ownerReferences")
	unstructured.RemoveNestedField(obj.UnstructuredContent(), "metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration")
	_, exist, err := unstructured.NestedString(obj.UnstructuredContent(), "metadata", "generateName")
	if err != nil {
		return err
	}
	if exist {
		unstructured.RemoveNestedField(obj.UnstructuredContent(), "metadata", "name")
	}

	// remove the status from spec which are related to status
	unstructured.RemoveNestedField(obj.UnstructuredContent(), "spec", "status")
	unstructured.RemoveNestedField(obj.UnstructuredContent(), "spec", "statusMessage")

	return nil
}
