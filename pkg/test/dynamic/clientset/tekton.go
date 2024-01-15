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

package clientset

import (
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/triggers/pkg/apis/triggers"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var allowedTektonTypes = map[string][]string{
	"v1beta1": {"pipelineruns", "taskruns", "pipelines", "clustertasks", "tasks", "conditions", "customruns"},
	"v1":      {"pipelineruns", "taskruns", "pipelines", "tasks"},
}

var allowedTriggerTektonTypes = map[string][]string{
	"v1alpha1": {"triggertemplates", "triggerbindings", "clustertriggerbindings", "eventlisteners"},
	"v1beta1":  {"triggertemplates", "triggerbindings", "clustertriggerbindings", "eventlisteners"},
}

// WithClient adds Tekton related clients to the Dynamic client.
func WithClient(client dynamic.Interface) Option {
	return func(cs *Clientset) {
		for version, resources := range allowedTektonTypes {
			for _, resource := range resources {
				r := schema.GroupVersionResource{
					Group:    pipeline.GroupName,
					Version:  version,
					Resource: resource,
				}
				cs.Add(r, client)
			}
		}

		for version, resources := range allowedTriggerTektonTypes {
			for _, resource := range resources {
				r := schema.GroupVersionResource{
					Group:    triggers.GroupName,
					Version:  version,
					Resource: resource,
				}
				cs.Add(r, client)
			}
		}
	}
}
