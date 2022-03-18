// Copyright © 2019 The Tekton Authors.
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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVersionGood(t *testing.T) {
	v := clientVersion
	defer func() { clientVersion = v }()

	scenarios := []struct {
		name          string
		clientVersion string
		serverVersion string
		expected      string
	}{
		{
			name:          "test-available-new-version",
			clientVersion: "v0.0.1",
			serverVersion: "v0.0.2",
			expected:      "A newer version (v0.0.2) of Tekton CLI is available, please check https://github.com/tektoncd/cli/releases/tag/v0.0.2\n",
		},
		{
			name:          "test-same-version",
			clientVersion: "v0.0.10",
			serverVersion: "v0.0.10",
			expected:      "You are running the latest version (v0.0.10) of Tekton CLI\n",
		},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			clientVersion = s.clientVersion
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				okResponse, _ := json.Marshal(GHVersion{
					TagName: s.serverVersion,
					HTMLURL: "https://github.com/tektoncd/cli/releases/tag/" + s.serverVersion,
				})
				_, _ = w.Write([]byte(okResponse))
			})
			httpClient, teardown := testingHTTPClient(h)
			defer teardown()

			cli := NewClient(time.Duration(0))
			cli.httpClient = httpClient
			output, err := checkRelease(cli)
			assert.NilError(t, err)
			golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
		})
	}

	clientVersion = "v1.2.3"

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{})

	cs := pipelinetest.Clients{Kube: seedData.Kube}
	p := &test.Params{Kube: cs.Kube}
	version := Command(p)
	got, err := test.ExecuteCommand(version, "version", "")
	assert.NilError(t, err)
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestGetComponentVersions(t *testing.T) {
	pipelineDeploymentLabels := map[string]string{
		"app.kubernetes.io/part-of":   "tekton-pipelines",
		"app.kubernetes.io/component": "controller",
		"app.kubernetes.io/name":      "controller",
	}
	triggersDeploymentLabels := map[string]string{
		"app.kubernetes.io/part-of":   "tekton-triggers",
		"app.kubernetes.io/component": "controller",
		"app.kubernetes.io/name":      "controller",
	}
	dashboardDeploymentLabels := map[string]string{
		"app.kubernetes.io/part-of":   "tekton-dashboard",
		"app.kubernetes.io/component": "dashboard",
		"app.kubernetes.io/name":      "dashboard",
	}
	pipelineDeployment := getDeploymentData("pipeline-dep", "", pipelineDeploymentLabels, map[string]string{"app.kubernetes.io/version": "v0.10.0"}, nil)
	triggersDeployment := getDeploymentData("triggers-dep", "", triggersDeploymentLabels, map[string]string{"app.kubernetes.io/version": "v0.5.0"}, nil)
	dashboardDeployment := getDeploymentData("dashboard-dep", "", dashboardDeploymentLabels, map[string]string{"app.kubernetes.io/version": "v0.7.0"}, nil)

	testParams := []struct {
		name                  string
		namespace             string
		userProvidedNamespace string
		deployment            []*v1.Deployment
		versions              []string
	}{{
		name:                  "deployment only with pipeline installed",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            []*v1.Deployment{pipelineDeployment},
		versions:              []string{"dev", "v0.10.0", "unknown", "unknown"},
	}, {
		name:                  "deployment with pipeline and triggers installed",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            []*v1.Deployment{pipelineDeployment, triggersDeployment},
		versions:              []string{"dev", "v0.10.0", "unknown", "v0.5.0"},
	}, {
		name:                  "deployment with pipeline and dashboard installed",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            []*v1.Deployment{pipelineDeployment, dashboardDeployment},
		versions:              []string{"dev", "v0.10.0", "v0.7.0", "unknown"},
	}, {
		name:                  "deployment with pipeline, triggers and dashboard installed",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            []*v1.Deployment{pipelineDeployment, triggersDeployment, dashboardDeployment},
		versions:              []string{"dev", "v0.10.0", "v0.7.0", "v0.5.0"},
	},
	}
	components := []string{"client", "pipeline", "dashboard", "triggers"}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			for i, c := range components {
				seedData, _ := test.SeedTestData(t, pipelinetest.Data{})
				cs := pipelinetest.Clients{Kube: seedData.Kube}
				p := &test.Params{Kube: cs.Kube}
				version := Command(p)
				cls, err := p.Clients()
				if err != nil {
					t.Errorf("failed to get client: %v", err)
				}
				// To add multiple deployments in a particular namespace
				for _, v := range tp.deployment {
					if _, err := cls.Kube.AppsV1().Deployments(tp.namespace).Create(context.Background(), v, metav1.CreateOptions{}); err != nil {
						t.Errorf("failed to create deployment: %v", err)
					}
				}
				got, _ := test.ExecuteCommand(version, "version", "-n", "test", "--component", c)
				fmt.Println(t.Name())
				assert.Equal(t, strings.TrimSpace(got), tp.versions[i])
			}
		})
	}

}

func TestVersionBad(t *testing.T) {
	v := clientVersion
	defer func() { clientVersion = v }()

	scenarios := []struct {
		name          string
		clientVersion string
		serverVersion string
		expectederr   string
	}{
		{
			name:          "bad-server-version",
			clientVersion: "v0.0.1",
			serverVersion: "BAD",
			expectederr:   "failed to parse version: No Major.Minor.Patch elements found",
		},
		{
			name:          "bad-client-version",
			clientVersion: "BAD",
			serverVersion: "v0.0.1",
			expectederr:   "failed to parse version: No Major.Minor.Patch elements found",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			clientVersion = s.clientVersion
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				okResponse, _ := json.Marshal(GHVersion{
					TagName: s.serverVersion,
					HTMLURL: "https://github.com/tektoncd/cli/releases/tag/" + s.serverVersion,
				})
				_, _ = w.Write([]byte(okResponse))
			})
			httpClient, teardown := testingHTTPClient(h)
			defer teardown()

			cli := NewClient(time.Duration(0))
			cli.httpClient = httpClient
			output, err := checkRelease(cli)
			assert.Error(t, err, s.expectederr)
			assert.Assert(t, output == "")
		})
	}
}

func testingHTTPClient(handler http.Handler) (*http.Client, func()) {
	s := httptest.NewTLSServer(handler)

	cli := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
			TLSClientConfig: &tls.Config{
				// nolint: gosec
				InsecureSkipVerify: true,
			},
		},
	}

	return cli, s.Close
}

func TestGetVersions(t *testing.T) {
	pipelineDeploymentLabels := map[string]string{
		"app.kubernetes.io/part-of":   "tekton-pipelines",
		"app.kubernetes.io/component": "controller",
		"app.kubernetes.io/name":      "controller",
	}
	triggersDeploymentLabels := map[string]string{
		"app.kubernetes.io/part-of":   "tekton-triggers",
		"app.kubernetes.io/component": "controller",
		"app.kubernetes.io/name":      "controller",
	}
	dashboardDeploymentLabels := map[string]string{
		"app.kubernetes.io/part-of":   "tekton-dashboard",
		"app.kubernetes.io/component": "dashboard",
		"app.kubernetes.io/name":      "dashboard",
	}
	pipelineDeployment := getDeploymentData("pipeline-dep", "", pipelineDeploymentLabels, map[string]string{"app.kubernetes.io/version": "v0.10.0"}, nil)
	triggersDeployment := getDeploymentData("triggers-dep", "", triggersDeploymentLabels, map[string]string{"app.kubernetes.io/version": "v0.5.0"}, nil)
	dashboardDeployment := getDeploymentData("dashboard-dep", "", dashboardDeploymentLabels, map[string]string{"app.kubernetes.io/version": "v0.7.0"}, nil)

	pipelineConfigMap := getConfigMapData("pipelines-info", "v0.10.0", map[string]string{"app.kubernetes.io/part-of": "tekton-pipelines"})
	chainsConfigMap := getConfigMapData("chains-info", "v0.8.0", map[string]string{"app.kubernetes.io/part-of": "tekton-chains"})
	triggersConfigMap := getConfigMapData("triggers-info", "v0.5.0", map[string]string{"app.kubernetes.io/part-of": "tekton-pipelines"})
	dashboardConfigMap := getConfigMapData("dashboard-info", "v0.7.0", map[string]string{"app.kubernetes.io/part-of": "tekton-pipelines"})
	operatorConfigMap := getConfigMapData("tekton-operator-info", "v0.54.0", map[string]string{"app.kubernetes.io/part-of": "tekton-pipelines"})

	testParams := []struct {
		name                  string
		namespace             string
		userProvidedNamespace string
		deployment            []*v1.Deployment
		configMap             []*corev1.ConfigMap
		goldenFile            bool
	}{{
		name:       "empty deployment items",
		namespace:  "tekton-pipelines",
		deployment: []*v1.Deployment{},
		goldenFile: true,
	}, {
		name:                  "deployment only with pipeline installed",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            []*v1.Deployment{pipelineDeployment},
		goldenFile:            true,
	}, {
		name:                  "deployment with pipeline and triggers installed",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            []*v1.Deployment{pipelineDeployment, triggersDeployment},
		goldenFile:            true,
	}, {
		name:                  "deployment with pipeline and dashboard installed",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            []*v1.Deployment{pipelineDeployment, dashboardDeployment},
		goldenFile:            true,
	}, {
		name:                  "deployment with pipeline, triggers and dashboard installed",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            []*v1.Deployment{pipelineDeployment, triggersDeployment, dashboardDeployment},
		goldenFile:            true,
	}, {
		name:                  "deployment with pipeline, triggers, dashboard and operator installed",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            []*v1.Deployment{},
		configMap:             []*corev1.ConfigMap{pipelineConfigMap, triggersConfigMap, dashboardConfigMap, operatorConfigMap},
		goldenFile:            true,
	}, {
		name:                  "deployment with pipeline, chains, triggers, dashboard and operator installed",
		namespace:             "test",
		userProvidedNamespace: "test",
		deployment:            []*v1.Deployment{},
		configMap:             []*corev1.ConfigMap{pipelineConfigMap, chainsConfigMap, triggersConfigMap, dashboardConfigMap, operatorConfigMap},
		goldenFile:            true,
	}}
	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			seedData, _ := test.SeedTestData(t, pipelinetest.Data{})
			cs := pipelinetest.Clients{Kube: seedData.Kube}
			p := &test.Params{Kube: cs.Kube}
			version := Command(p)
			cls, err := p.Clients()
			if err != nil {
				t.Errorf("failed to get client: %v", err)
			}
			// To add multiple deployments in a particular namespace
			for _, v := range tp.deployment {
				if _, err := cls.Kube.AppsV1().Deployments(tp.namespace).Create(context.Background(), v, metav1.CreateOptions{}); err != nil {
					t.Errorf("failed to create deployment: %v", err)
				}
			}
			for _, v := range tp.configMap {
				if _, err := cls.Kube.CoreV1().ConfigMaps(tp.namespace).Create(context.Background(), v, metav1.CreateOptions{}); err != nil {
					t.Errorf("failed to create configMap")
				}
			}
			got, _ := test.ExecuteCommand(version, "version", "-n", "test")
			golden.Assert(t, got, strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
		})
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
