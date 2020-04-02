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
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetPipelineVersion(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{})
	p := &test.Params{Kube: cs.Kube}
	cls, err := p.Clients()
	if err != nil {
		t.Errorf("failed to get client: %v", err)
	}

	testParams := []struct {
		name       string
		deployment *v1.Deployment
		want       string
	}{{
		name:       "emppty deployment items",
		deployment: &v1.Deployment{},
		want:       "",
	}, {
		name:       "deployment spec does not have labels and annotations specific to version",
		deployment: getDeploymentData("dep1", "pipeline/cmd/controller:v0.9.0@sha256:5d23", nil, nil),
		want:       "v0.9.0",
	}, {
		name:       "deployment spec have annotation specific to version",
		deployment: getDeploymentData("dep2", "", nil, map[string]string{"tekton.dev/release": "v0.10.0"}),
		want:       "v0.10.0",
	}, {
		name:       "deployment spec have labels specific to master version",
		deployment: getDeploymentData("dep4", "", map[string]string{"pipeline.tekton.dev/release": "master"}, nil),
		want:       "master",
	}}
	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			if _, err := cls.Kube.AppsV1().Deployments(pipelineNamespace).Create(tp.deployment); err != nil {
				t.Errorf("failed to create deployment: %v", err)
			}
			version, _ := GetPipelineVersion(cls)
			test.AssertOutput(t, tp.want, version)
		})
	}
}

func getDeploymentData(name, image string, labels, annotations map[string]string) *v1.Deployment {
	return &v1.Deployment{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app.kubernetes.io/component": "controller",
				"app.kubernetes.io/name":      "tekton-pipelines",
			},
		},
		Spec: v1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
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
