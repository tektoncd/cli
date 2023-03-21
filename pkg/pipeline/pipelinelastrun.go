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

package pipeline

import (
	"fmt"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LastRun returns the name of last pipelinerun for a given pipeline
func LastRunName(cs *cli.Clients, resourceName, ns string) (string, error) {
	latest, err := LastRun(cs, resourceName, ns)
	if err != nil {
		return "", err
	}
	return latest.Name, nil
}

// DynamicLastRun returns the last run for a given pipeline
func LastRun(cs *cli.Clients, pipeline string, ns string) (*v1.PipelineRun, error) {
	options := metav1.ListOptions{}
	if pipeline != "" {
		options = metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tekton.dev/pipeline=%s", pipeline),
		}
	}

	var runs *v1.PipelineRunList
	err := actions.ListV1(pipelineRunGroupResource, cs, options, ns, &runs)
	if err != nil {
		return nil, err
	}

	if len(runs.Items) == 0 {
		return nil, fmt.Errorf("no pipelineruns related to pipeline %s found in namespace %s", pipeline, ns)
	}

	latest := runs.Items[0]
	for _, run := range runs.Items {
		if run.CreationTimestamp.Time.After(latest.CreationTimestamp.Time) {
			latest = run
		}
	}

	return &latest, nil
}
