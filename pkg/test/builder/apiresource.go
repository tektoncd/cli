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

package builder

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	pipelinegroup string = "tekton.dev"
	triggersGroup string = "triggers.tekton.dev"
)

func APIResourceList(version string, kinds []string) []*metav1.APIResourceList {
	return []*metav1.APIResourceList{
		{
			GroupVersion: pipelinegroup + "/" + version,
			APIResources: apiresources(pipelinegroup, version, kinds),
		},
	}
}

func TriggersAPIResourceList(version string, kinds []string) []*metav1.APIResourceList {
	return []*metav1.APIResourceList{
		{
			GroupVersion: triggersGroup + "/" + version,
			APIResources: apiresources(triggersGroup, version, kinds),
		},
	}
}

func apiresources(group string, version string, kinds []string) []metav1.APIResource {
	apires := make([]metav1.APIResource, 0)
	for _, kind := range kinds {
		namespaced := true
		if strings.Contains(kind, "cluster") {
			namespaced = false
		}
		apires = append(apires, metav1.APIResource{
			Name:       kind + "s",
			Group:      group,
			Kind:       kind,
			Version:    version,
			Namespaced: namespaced,
		})
	}
	return apires
}
