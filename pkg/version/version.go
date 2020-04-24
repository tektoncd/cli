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
	"fmt"
	"strings"

	"github.com/tektoncd/cli/pkg/cli"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetPipelineVersion Get pipeline version, functions imported from Dashboard
func GetPipelineVersion(c *cli.Clients) (string, error) {
	version := ""
	namespaces := []string{"tekton-pipelines", "openshift-pipelines"}

	listOptions := metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/component=controller,app.kubernetes.io/name=tekton-pipelines",
	}

	var deployments []v1.Deployment
	for _, namespace := range namespaces {
		d, _ := c.Kube.AppsV1().Deployments(namespace).List(listOptions)
		deployments = append(deployments, d.Items...)
	}

	if len(deployments) > 0 {
		version = findVersion(deployments)
	}

	if version == "" {
		d, _ := c.Kube.AppsV1().Deployments("").List(listOptions)
		deployments = append(deployments, d.Items...)
		version = findVersion(deployments)
	}

	if version == "" {
		return "", fmt.Errorf("Error getting the tekton pipelines deployment version. Version is unknown")
	}

	return version, nil
}

func findVersion(deployments []v1.Deployment) string {
	version := ""
	for _, deployment := range deployments {
		deploymentLabels := deployment.Spec.Template.GetLabels()
		deploymentAnnotations := deployment.Spec.Template.GetAnnotations()

		// For master of Tekton Pipelines
		version = deploymentLabels["pipeline.tekton.dev/release"]

		// For Tekton Pipelines 0.10.0 + 0.10.1 tekton.dev/release has been set as annotation
		if version == "" {
			version = deploymentAnnotations["tekton.dev/release"]
		}

		// For Tekton Pipelines 0.9.0 - 0.9.2
		if version == "" {
			deploymentImage := deployment.Spec.Template.Spec.Containers[0].Image
			if strings.Contains(deploymentImage, "pipeline/cmd/controller") && strings.Contains(deploymentImage, ":") && strings.Contains(deploymentImage, "@") {
				s := strings.SplitAfter(deploymentImage, ":")
				if strings.Contains(s[1], "@") {
					t := strings.Split(s[1], "@")
					version = t[0]
				}
			}
		}
	}
	return version
}
