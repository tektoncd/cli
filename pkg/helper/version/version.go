package version

import (
	"fmt"
	"strings"

	"github.com/tektoncd/cli/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const pipelineNamespace = "tekton-pipelines"

// GetPipelineVersion Get pipeline version, functions imported from Dashboard
func GetPipelineVersion(c *cli.Clients) (string, error) {
	version := ""

	listOptions := metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/component=controller,app.kubernetes.io/name=tekton-pipelines",
	}

	deployments, err := c.Kube.AppsV1().Deployments(pipelineNamespace).List(listOptions)
	if err != nil {
		return "", err
	}

	for _, deployment := range deployments.Items {
		deploymentAnnotations := deployment.Spec.Template.GetAnnotations()

		// For master of Tekton Pipelines
		version = deploymentAnnotations["pipeline.tekton.dev/release"]

		// For Tekton Pipelines 0.10.0 + 0.10.1
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

	if version == "" {
		return "", fmt.Errorf("Error getting the tekton pipelines deployment version. Version is unknown")
	}

	return version, nil
}
