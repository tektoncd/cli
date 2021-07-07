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
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	pipelinesControllerSelector string = "app.kubernetes.io/part-of=tekton-pipelines,app.kubernetes.io/component=controller,app.kubernetes.io/name=controller"
	pipelinesConfigMapName      string = "pipelines-info"
)

var defaultNamespaces = []string{"tekton-pipelines", "openshift-pipelines"}

func GetPipelineVersion(dynamic dynamic.Interface) (string, error) {

	var version string

	configMap, err := getConfigMap(dynamic, pipelinesConfigMapName)
	if err == nil {
		dataObj, _, _ := unstructured.NestedStringMap(configMap.Object, "data")
		version = dataObj["version"]
		if version != "" {
			return version, nil
		}
	}

	deploymentsList, err := getDeployments(dynamic, pipelinesControllerSelector)

	if err != nil {
		return "", err
	}

	version = findPipelineVersion(deploymentsList.Items)

	if version == "" {
		return "", fmt.Errorf("error getting the tekton pipelines deployment version. Version is unknown")
	}

	return version, nil
}

func getDeployments(dynamic dynamic.Interface, newLabel string) (*unstructured.UnstructuredList, error) {
	var (
		err         error
		deployments *unstructured.UnstructuredList
	)

	for _, n := range defaultNamespaces {
		deployments, err = getDeploy(dynamic, newLabel, n)
		if err != nil {
			if strings.Contains(err.Error(), fmt.Sprintf(`cannot list resource "deployments" in API group "apps" in the namespace "%s"`, n)) {
				continue
			} else {
				return nil, err
			}
		}
		if len(deployments.Items) != 0 {
			break
		}
	}
	return deployments, err
}

func getDeploy(dynamic dynamic.Interface, newLabel, ns string) (*unstructured.UnstructuredList, error) {
	gvrObj := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	deployments, err := dynamic.Resource(gvrObj).Namespace(ns).List(context.Background(), metav1.ListOptions{LabelSelector: newLabel})
	if err != nil {
		return nil, err
	}
	return deployments, nil
}

func getConfigMap(dynamic dynamic.Interface, name string) (*unstructured.Unstructured, error) {

	var (
		err       error
		configMap *unstructured.Unstructured
	)

	gvrObj := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}

	for _, ns := range defaultNamespaces {
		configMap, err = dynamic.Resource(gvrObj).Namespace(ns).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if configMap != nil {
			break
		}
	}

	return configMap, nil
}

func findPipelineVersion(deployments []unstructured.Unstructured) string {
	version := ""
	for _, deployment := range deployments {
		// Fetching the labels from pod's template
		deploymentLabels, _, _ := unstructured.NestedStringMap(deployment.Object, "spec", "template", "metadata", "labels")
		version = deploymentLabels["app.kubernetes.io/version"]
	}
	return version
}
