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

package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	pipelinetest "github.com/tektoncd/pipeline/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type Data struct {
	Pipelines []*v1beta1.Pipeline
	Tasks     []*v1beta1.Task
}

func SeedTestData(t *testing.T, d pipelinetest.Data) (pipelinetest.Clients, pipelinetest.Informers) {
	ctx, _ := ttesting.SetupFakeContext(t)
	return pipelinetest.SeedTestData(t, ctx, d)
}

func SeedV1beta1TestData(t *testing.T, d Data) (Clients, Informers) {
	ctx, _ := ttesting.SetupFakeContext(t)
	return seedTestData(t, ctx, d)
}

func GetDeploymentData(name, version string) *unstructured.Unstructured {
	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": name,
				"labels": map[string]interface{}{
					"app.kubernetes.io/part-of":   "tekton-pipelines",
					"app.kubernetes.io/component": "controller",
					"app.kubernetes.io/name":      "controller",
				},
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app.kubernetes.io/version": version,
						},
						"annotations": map[string]interface{}{},
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "web",
								"image": "nginx:1.12",
							},
						},
					},
				},
			},
		},
	}
	return deployment
}

func GetConfigMapData(name string, version string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name": name,
				"labels": map[string]interface{}{
					"app.kubernetes.io/part-of":  "tekton-pipelines",
					"app.kubernetes.io/instance": "default",
				},
			},
			"data": map[string]interface{}{
				"version": version,
			},
		},
	}
}

func CreateTektonPipelineController(dynamic dynamic.Interface, version string) error {

	deployment := GetDeploymentData("test", version)

	deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	_, err := dynamic.Resource(deploymentRes).Namespace("tekton-pipelines").Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create deployment: %v", err)
	}
	return nil
}
