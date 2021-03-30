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

package task

import (
	"fmt"

	"github.com/tektoncd/cli/pkg/cli"
	trlist "github.com/tektoncd/cli/pkg/taskrun/list"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LastRun returns the last taskrun for a given task/clustertask
func LastRun(cs *cli.Clients, resourceName, ns, kind string) (*v1beta1.TaskRun, error) {
	options := metav1.ListOptions{}

	// change the label value to clusterTask if the resource is ClusterTask
	label := "task"
	if kind == "ClusterTask" {
		label = "clusterTask"
	}

	if resourceName != "" {
		options = metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tekton.dev/%s=%s", label, resourceName),
		}
	}

	runs, err := trlist.TaskRuns(cs, options, ns)
	if err != nil {
		return nil, err
	}

	if len(runs.Items) == 0 {
		return nil, fmt.Errorf("no TaskRuns related to %s %s found in namespace %s", kind, resourceName, ns)
	}

	if kind == "Task" {
		runs.Items = FilterByRef(runs.Items, kind)
	}

	latest := runs.Items[0]
	for _, run := range runs.Items {
		if run.CreationTimestamp.Time.After(latest.CreationTimestamp.Time) {
			latest = run
		}
	}

	return &latest, nil
}

// this will filter the taskrun which have reference to Task or ClusterTask
func FilterByRef(taskruns []v1beta1.TaskRun, kind string) []v1beta1.TaskRun {
	var filtered []v1beta1.TaskRun
	for _, taskrun := range taskruns {
		if string(taskrun.Spec.TaskRef.Kind) == kind {
			filtered = append(filtered, taskrun)
		}
	}
	return filtered
}
