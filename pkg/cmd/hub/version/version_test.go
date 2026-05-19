// Copyright © 2021 The Tekton Authors.
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

package version

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/cli/pkg/cmd/hub/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGetPipelineVersion(t *testing.T) {

	testParams := []struct {
		name                  string
		namespace             string
		userProvidedNamespace string
		deployment            *unstructured.Unstructured
		want                  string
	}{{
		name:       "empty deployment items",
		namespace:  "tekton-pipelines",
		deployment: &unstructured.Unstructured{},
		want:       "",
	}, {
		name:       "deployment spec have labels specific to master version (new labels)",
		namespace:  "tekton-pipelines",
		deployment: test.GetDeploymentData("dep5", "master-tekton-pipelines"),
		want:       "master-tekton-pipelines",
	}}
	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			dynamic := test.DynamicClient()
			deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
			_, err := dynamic.Resource(deploymentRes).Namespace(tp.namespace).Create(context.TODO(), tp.deployment, metav1.CreateOptions{})
			if err != nil {
				t.Errorf("failed to create deployment: %v", err)
			}
			version, _ := GetPipelineVersion(dynamic)
			assert.Equal(t, tp.want, version)
		})
	}
}

func TestGetPipelineVersionViaConfigMap(t *testing.T) {

	testParams := []struct {
		name                  string
		namespace             string
		userProvidedNamespace string
		configMap             *unstructured.Unstructured
		want                  string
	}{{
		name:      "empty deployment items",
		namespace: "tekton-pipelines",
		configMap: &unstructured.Unstructured{},
		want:      "",
	}, {
		name:      "deployment spec have labels specific to master version (new labels)",
		namespace: "tekton-pipelines",
		configMap: test.GetConfigMapData("pipelines-info", "master-tekton-pipelines"),
		want:      "master-tekton-pipelines",
	}}
	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			dynamic := test.DynamicClient()
			deploymentRes := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
			_, err := dynamic.Resource(deploymentRes).Namespace(tp.namespace).Create(context.TODO(), tp.configMap, metav1.CreateOptions{})
			if err != nil {
				t.Errorf("failed to create deployment: %v", err)
			}
			version, _ := GetPipelineVersion(dynamic)
			assert.Equal(t, tp.want, version)
		})
	}
}
