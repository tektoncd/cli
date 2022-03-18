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

package version

import (
	"context"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetPipelineVersion(t *testing.T) {
	oldDeploymentLabels := map[string]string{
		"app.kubernetes.io/component": "controller",
		"app.kubernetes.io/name":      "tekton-pipelines",
	}

	newDeploymentLabels := map[string]string{
		"app.kubernetes.io/part-of":   "tekton-pipelines",
		"app.kubernetes.io/component": "controller",
		"app.kubernetes.io/name":      "controller",
	}

	testParams := []struct {
		name                  string
		namespace             string
		userProvidedNamespace string
		deployment            *v1.Deployment
		want                  string
	}{{
		name:       "empty deployment items",
		namespace:  "tekton-pipelines",
		deployment: &v1.Deployment{},
		want:       "",
	}, {
		name:                  "controller in different namespace (old labels)",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            getDeploymentData("dep", "", oldDeploymentLabels, nil, map[string]string{"tekton.dev/release": "v0.10.0"}),
		want:                  "v0.10.0",
	}, {
		name:       "deployment spec does not have labels and annotations specific to version (old labels)",
		namespace:  "tekton-pipelines",
		deployment: getDeploymentData("dep1", "pipeline/cmd/controller:v0.9.0@sha256:5d23", oldDeploymentLabels, nil, nil),
		want:       "v0.9.0",
	}, {
		name:       "deployment spec have annotation specific to version (old labels)",
		namespace:  "openshift-pipelines",
		deployment: getDeploymentData("dep2", "", oldDeploymentLabels, nil, map[string]string{"tekton.dev/release": "v0.10.0"}),
		want:       "v0.10.0",
	}, {
		name:       "deployment spec have labels specific to version (old labels)",
		namespace:  "tekton-pipelines",
		deployment: getDeploymentData("dep3", "", oldDeploymentLabels, map[string]string{"pipeline.tekton.dev/release": "v0.11.0"}, nil),
		want:       "v0.11.0",
	}, {
		name:                  "controller in different namespace (new labels)",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            getDeploymentData("dep4", "", newDeploymentLabels, map[string]string{"app.kubernetes.io/version": "master-test"}, nil),
		want:                  "master-test",
	}, {
		name:       "deployment spec have labels specific to master version (new labels)",
		namespace:  "tekton-pipelines",
		deployment: getDeploymentData("dep5", "", newDeploymentLabels, map[string]string{"app.kubernetes.io/version": "master-tekton-pipelines"}, nil),
		want:       "master-tekton-pipelines",
	}}
	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			cs, _ := test.SeedTestData(t, pipelinetest.Data{})
			p := &test.Params{Kube: cs.Kube}
			cls, err := p.Clients()
			if err != nil {
				t.Errorf("failed to get client: %v", err)
			}
			if _, err := cls.Kube.AppsV1().Deployments(tp.namespace).Create(context.Background(), tp.deployment, metav1.CreateOptions{}); err != nil {
				t.Errorf("failed to create deployment: %v", err)
			}
			version, _ := GetPipelineVersion(cls, tp.userProvidedNamespace)
			test.AssertOutput(t, tp.want, version)
		})
	}
}

func TestGetPipelinesVersionViaConfigMap(t *testing.T) {

	testParams := []struct {
		name                  string
		namespace             string
		userProvidedNamespace string
		configMap             *corev1.ConfigMap
		want                  string
	}{
		{
			name:      "get pipelines version from configmap in tekton-pipelines namespace",
			namespace: "tekton-pipelines",
			configMap: getConfigMapData("pipelines-info", "main", map[string]string{"app.kubernetes.io/part-of": "tekton-pipelines"}),
			want:      "main",
		},
		{
			name:                  "get pipelines version from configmap present in different namespace other than default namespaces",
			namespace:             "test",
			userProvidedNamespace: "test",
			configMap:             getConfigMapData("pipelines-info", "test", map[string]string{"app.kubernetes.io/part-of": "tekton-pipelines"}),
			want:                  "test",
		},
	}
	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			cs, _ := test.SeedTestData(t, pipelinetest.Data{})
			p := &test.Params{Kube: cs.Kube}
			cls, err := p.Clients()
			if err != nil {
				t.Errorf("failed to get client: %v", err)
			}
			if _, err := cls.Kube.CoreV1().ConfigMaps(tp.namespace).Create(context.Background(), tp.configMap, metav1.CreateOptions{}); err != nil {
				t.Errorf("failed to create configmap: %v", err)
			}
			version, _ := GetPipelineVersion(cls, tp.userProvidedNamespace)
			test.AssertOutput(t, tp.want, version)
		})
	}
}

func getConfigMapData(name, version string, labels map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Data: map[string]string{
			"version": version,
		},
	}
}

func getDeploymentData(name, image string, deploymentLabels, podTemplateLabels, annotations map[string]string) *v1.Deployment {
	return &v1.Deployment{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: deploymentLabels,
		},
		Spec: v1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podTemplateLabels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: image,
					}},
				},
			},
		},
	}
}

func TestGetChainsVersionViaConfigMap(t *testing.T) {

	testParams := []struct {
		name                  string
		namespace             string
		userProvidedNamespace string
		configMap             *corev1.ConfigMap
		want                  string
	}{
		{
			name:                  "get chains version from configmap present in different namespace other than default namespaces",
			namespace:             "test",
			userProvidedNamespace: "test",
			configMap:             getConfigMapData("chains-info", "test", map[string]string{"app.kubernetes.io/part-of": "tekton-chains"}),
			want:                  "test",
		}, {
			name:      "get chains version from configmap in tekton-chains namespace",
			namespace: "tekton-chains",
			configMap: getConfigMapData("chains-info", "main", map[string]string{"app.kubernetes.io/part-of": "tekton-chains"}),
			want:      "main",
		},
	}
	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			cs, _ := test.SeedTestData(t, pipelinetest.Data{})
			p := &test.Params{Kube: cs.Kube}
			cls, err := p.Clients()
			if err != nil {
				t.Errorf("failed to get client: %v", err)
			}
			if _, err := cls.Kube.CoreV1().ConfigMaps(tp.namespace).Create(context.Background(), tp.configMap, metav1.CreateOptions{}); err != nil {
				t.Errorf("failed to create configmap: %v", err)
			}
			version, _ := GetChainsVersion(cls, tp.userProvidedNamespace)
			test.AssertOutput(t, tp.want, version)
		})
	}
}

func TestGetTriggerVersion(t *testing.T) {
	oldDeploymentLabels := map[string]string{
		"app.kubernetes.io/component": "controller",
		"app.kubernetes.io/name":      "tekton-triggers",
	}

	newDeploymentLabels := map[string]string{
		"app.kubernetes.io/part-of":   "tekton-triggers",
		"app.kubernetes.io/component": "controller",
		"app.kubernetes.io/name":      "controller",
	}

	testParams := []struct {
		name                  string
		namespace             string
		userProvidedNamespace string
		deployment            *v1.Deployment
		want                  string
	}{{
		name:       "empty deployment items",
		namespace:  "tekton-pipelines",
		deployment: &v1.Deployment{},
		want:       "",
	}, {
		name:                  "controller in different namespace (old labels)",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            getDeploymentData("dep", "", oldDeploymentLabels, nil, nil),
		want:                  "",
	}, {
		name:       "deployment spec have labels specific to version (old labels)",
		namespace:  "tekton-pipelines",
		deployment: getDeploymentData("dep1", "", oldDeploymentLabels, map[string]string{"triggers.tekton.dev/release": "v0.3.1"}, nil),
		want:       "v0.3.1",
	}, {
		name:                  "controller in different namespace (new labels)",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            getDeploymentData("dep2", "", newDeploymentLabels, map[string]string{"app.kubernetes.io/version": "v0.5.0"}, nil),
		want:                  "v0.5.0",
	}, {
		name:       "deployment spec have labels specific to master version (new labels)",
		namespace:  "tekton-pipelines",
		deployment: getDeploymentData("dep3", "", newDeploymentLabels, map[string]string{"app.kubernetes.io/version": "master-tekton-triggers"}, nil),
		want:       "master-tekton-triggers",
	}}
	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			cs, _ := test.SeedTestData(t, pipelinetest.Data{})
			p := &test.Params{Kube: cs.Kube}
			cls, err := p.Clients()
			if err != nil {
				t.Errorf("failed to get client: %v", err)
			}
			if _, err := cls.Kube.AppsV1().Deployments(tp.namespace).Create(context.Background(), tp.deployment, metav1.CreateOptions{}); err != nil {
				t.Errorf("failed to create deployment: %v", err)
			}
			version, _ := GetTriggerVersion(cls, tp.userProvidedNamespace)
			test.AssertOutput(t, tp.want, version)
		})
	}
}

func TestGetTriggersVersionViaConfigMap(t *testing.T) {

	testParams := []struct {
		name                  string
		namespace             string
		userProvidedNamespace string
		configMap             *corev1.ConfigMap
		want                  string
	}{
		{
			name:                  "get triggers version from configmap present in different namespace other than default namespaces",
			namespace:             "test",
			userProvidedNamespace: "test",
			configMap:             getConfigMapData("triggers-info", "test", map[string]string{"app.kubernetes.io/part-of": "tekton-triggers"}),
			want:                  "test",
		}, {
			name:      "get triggers version from configmap in tekton-pipelines namespace",
			namespace: "tekton-pipelines",
			configMap: getConfigMapData("triggers-info", "main", map[string]string{"app.kubernetes.io/part-of": "tekton-triggers"}),
			want:      "main",
		},
	}
	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			cs, _ := test.SeedTestData(t, pipelinetest.Data{})
			p := &test.Params{Kube: cs.Kube}
			cls, err := p.Clients()
			if err != nil {
				t.Errorf("failed to get client: %v", err)
			}
			if _, err := cls.Kube.CoreV1().ConfigMaps(tp.namespace).Create(context.Background(), tp.configMap, metav1.CreateOptions{}); err != nil {
				t.Errorf("failed to create configmap: %v", err)
			}
			version, _ := GetTriggerVersion(cls, tp.userProvidedNamespace)
			test.AssertOutput(t, tp.want, version)
		})
	}
}

func TestGetDashboardVersion(t *testing.T) {
	oldDeploymentLabels := map[string]string{
		"app": "tekton-dashboard",
	}

	newDeploymentLabels := map[string]string{
		"app.kubernetes.io/part-of":   "tekton-dashboard",
		"app.kubernetes.io/component": "dashboard",
		"app.kubernetes.io/name":      "dashboard",
	}

	testParams := []struct {
		name                  string
		namespace             string
		userProvidedNamespace string
		deployment            *v1.Deployment
		want                  string
	}{{
		name:       "empty deployment items",
		namespace:  "tekton-pipelines",
		deployment: &v1.Deployment{},
		want:       "",
	}, {
		name:                  "dashboard in different namespace (old labels)",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            getDeploymentData("dep", "", oldDeploymentLabels, nil, nil),
		want:                  "",
	}, {
		name:       "deployment spec have labels specific to version (old labels)",
		namespace:  "tekton-pipelines",
		deployment: getDeploymentData("dep1", "", map[string]string{"app": "tekton-dashboard", "version": "v0.6.0"}, oldDeploymentLabels, nil),
		want:       "v0.6.0",
	}, {
		name:                  "dashboard in different namespace (new labels)",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            getDeploymentData("dep2", "", newDeploymentLabels, map[string]string{"app.kubernetes.io/version": "v0.7.0"}, nil),
		want:                  "v0.7.0",
	}, {
		name:       "deployment spec have labels specific to master version (new labels)",
		namespace:  "tekton-pipelines",
		deployment: getDeploymentData("dep3", "", newDeploymentLabels, map[string]string{"app.kubernetes.io/version": "master-tekton-dashboard"}, nil),
		want:       "master-tekton-dashboard",
	}}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			cs, _ := test.SeedTestData(t, pipelinetest.Data{})
			p := &test.Params{Kube: cs.Kube}
			cls, err := p.Clients()
			if err != nil {
				t.Errorf("failed to get client: %v", err)
			}
			if _, err := cls.Kube.AppsV1().Deployments(tp.namespace).Create(context.Background(), tp.deployment, metav1.CreateOptions{}); err != nil {
				t.Errorf("failed to create deployment: %v", err)
			}
			version, _ := GetDashboardVersion(cls, tp.userProvidedNamespace)
			test.AssertOutput(t, tp.want, version)
		})
	}
}

func TestGetDashboardVersionViaConfigMap(t *testing.T) {

	testParams := []struct {
		name                  string
		namespace             string
		userProvidedNamespace string
		configMap             *corev1.ConfigMap
		want                  string
	}{
		{
			name:      "get dashboard version from configmap in tekton-pipelines namespace",
			namespace: "tekton-pipelines",
			configMap: getConfigMapData("dashboard-info", "main", map[string]string{"app.kubernetes.io/part-of": "tekton-dashboard"}),
			want:      "main",
		},
		{
			name:                  "get dashboard version from configmap present in different namespace other than default namespaces",
			namespace:             "test",
			userProvidedNamespace: "test",
			configMap:             getConfigMapData("dashboard-info", "test", map[string]string{"app.kubernetes.io/part-of": "tekton-dashboard"}),
			want:                  "test",
		},
	}
	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			cs, _ := test.SeedTestData(t, pipelinetest.Data{})
			p := &test.Params{Kube: cs.Kube}
			cls, err := p.Clients()
			if err != nil {
				t.Errorf("failed to get client: %v", err)
			}
			if _, err := cls.Kube.CoreV1().ConfigMaps(tp.namespace).Create(context.Background(), tp.configMap, metav1.CreateOptions{}); err != nil {
				t.Errorf("failed to create configmap: %v", err)
			}
			version, _ := GetDashboardVersion(cls, tp.userProvidedNamespace)
			test.AssertOutput(t, tp.want, version)
		})
	}
}

func TestGetOperatorVersionViaConfigMap(t *testing.T) {

	testParams := []struct {
		name                  string
		namespace             string
		userProvidedNamespace string
		configMap             *corev1.ConfigMap
		want                  string
	}{
		{
			name:      "get operator version from configmap in tekton-pipelines namespace",
			namespace: "tekton-pipelines",
			configMap: getConfigMapData("tekton-operator-info", "main", map[string]string{"app.kubernetes.io/part-of": "tekton-pipelines"}),
			want:      "main",
		},
		{
			name:                  "get operator version from configmap present in different namespace other than default namespaces",
			namespace:             "test",
			userProvidedNamespace: "test",
			configMap:             getConfigMapData("tekton-operator-info", "test", map[string]string{"app.kubernetes.io/part-of": "tekton-pipelines"}),
			want:                  "test",
		},
	}
	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			cs, _ := test.SeedTestData(t, pipelinetest.Data{})
			p := &test.Params{Kube: cs.Kube}
			cls, err := p.Clients()
			if err != nil {
				t.Errorf("failed to get client: %v", err)
			}
			if _, err := cls.Kube.CoreV1().ConfigMaps(tp.namespace).Create(context.Background(), tp.configMap, metav1.CreateOptions{}); err != nil {
				t.Errorf("failed to create configmap: %v", err)
			}
			version, _ := GetOperatorVersion(cls, tp.userProvidedNamespace)
			test.AssertOutput(t, tp.want, version)
		})
	}
}
