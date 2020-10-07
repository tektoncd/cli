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
	"fmt"
	"strings"

	"github.com/tektoncd/cli/pkg/cli"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	pipelinesControllerSelector    string = "app.kubernetes.io/part-of=tekton-pipelines,app.kubernetes.io/component=controller,app.kubernetes.io/name=controller"
	oldPipelinesControllerSelector string = "app.kubernetes.io/component=controller,app.kubernetes.io/name=tekton-pipelines"
	triggersControllerSelector     string = "app.kubernetes.io/part-of=tekton-triggers,app.kubernetes.io/component=controller,app.kubernetes.io/name=controller"
	oldTriggersControllerSelector  string = "app.kubernetes.io/component=controller,app.kubernetes.io/name=tekton-triggers"
	dashboardControllerSelector    string = "app.kubernetes.io/part-of=tekton-dashboard,app.kubernetes.io/component=dashboard,app.kubernetes.io/name=dashboard"
	oldDashboardControllerSelector string = "app=tekton-dashboard"
)

// GetPipelineVersion Get pipeline version, functions imported from Dashboard
func GetPipelineVersion(c *cli.Clients, ns string) (string, error) {
	deploymentsList, err := getDeployments(c, pipelinesControllerSelector, oldPipelinesControllerSelector, ns)

	if err != nil {
		return "", err
	}

	version := findPipelineVersion(deploymentsList.Items)

	if version == "" {
		return "", fmt.Errorf("error getting the tekton pipelines deployment version. Version is unknown")
	}

	return version, nil
}

// Get deployments for either Tekton Triggers, Tekton Dashboard or Tekton Pipelines
func getDeployments(c *cli.Clients, newLabel, oldLabel, ns string) (*v1.DeploymentList, error) {
	var (
		err               error
		deployments       *v1.DeploymentList
		defaultNamespaces = []string{"tekton-pipelines", "openshift-pipelines"}
	)
	if ns != "" {
		deployments, err = getDeploy(c, newLabel, oldLabel, ns)
		return deployments, err
	}
	// If ldflag and flag doesn't specify the namespace fallback to default.
	for _, n := range defaultNamespaces {
		deployments, err = getDeploy(c, newLabel, oldLabel, n)
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

func getDeploy(c *cli.Clients, newLabel, oldLabel, ns string) (*v1.DeploymentList, error) {
	deployments, err := c.Kube.AppsV1().Deployments(ns).List(context.Background(), metav1.ListOptions{LabelSelector: newLabel})
	if err != nil {
		return nil, err
	}

	// NOTE: If the new labels selector returned nothing, try with old labels selector
	// The old labels selectors are deprecated and should be removed at some point
	if deployments == nil || len(deployments.Items) == 0 {
		deployments, err = c.Kube.AppsV1().Deployments(ns).List(context.Background(), metav1.ListOptions{LabelSelector: oldLabel})
		if err != nil {
			return nil, err
		}
	}
	return deployments, err
}

func findPipelineVersion(deployments []v1.Deployment) string {
	version := ""
	for _, deployment := range deployments {
		deploymentLabels := deployment.Spec.Template.GetLabels()
		deploymentAnnotations := deployment.Spec.Template.GetAnnotations()

		// For master of Tekton Pipelines
		version = deploymentLabels["app.kubernetes.io/version"]

		// For Tekton Pipelines 0.11.*
		if version == "" {
			version = deploymentLabels["pipeline.tekton.dev/release"]
		}

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

// GetTriggerVersion Get triggers version.
func GetTriggerVersion(c *cli.Clients, ns string) (string, error) {
	deploymentsList, err := getDeployments(c, triggersControllerSelector, oldTriggersControllerSelector, ns)

	if err != nil {
		return "", err
	}

	version := findTriggersVersion(deploymentsList.Items)

	if version == "" {
		return "", fmt.Errorf("error getting the tekton triggers deployment version. Version is unknown")
	}

	return version, nil
}

func findTriggersVersion(deployments []v1.Deployment) string {
	version := ""
	for _, deployment := range deployments {
		deploymentLabels := deployment.Spec.Template.GetLabels()

		// For master of Tekton Triggers
		if version = deploymentLabels["app.kubernetes.io/version"]; version == "" {
			// For Tekton Triggers 0.3.*
			version = deploymentLabels["triggers.tekton.dev/release"]
		}
	}
	return version
}

// GetDashboardVersion Get dashboard version.
func GetDashboardVersion(c *cli.Clients, ns string) (string, error) {
	deploymentsList, err := getDeployments(c, dashboardControllerSelector, oldDashboardControllerSelector, ns)

	if err != nil {
		return "", err
	}

	version := findDashboardVersion(deploymentsList.Items)
	if version == "" {
		return "", fmt.Errorf("error getting the tekton dashboard deployment version. Version is unknown")
	}

	return version, nil
}

func findDashboardVersion(deployments []v1.Deployment) string {
	version := ""
	for _, deployment := range deployments {
		deploymentLabels := deployment.Spec.Template.GetLabels()

		// For master of Tekton Dashboard
		if version = deploymentLabels["app.kubernetes.io/version"]; version == "" {
			// For Tekton Dashboard 0.6.*
			deploymentMetaLabels := deployment.GetObjectMeta().GetLabels()
			version = deploymentMetaLabels["version"]
		}
	}
	return version
}
