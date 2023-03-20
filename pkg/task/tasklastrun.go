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

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var taskrunGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}

// LastRun returns the name of last taskrun for a given task/clustertask
func LastRunName(cs *cli.Clients, resourceName, ns, kind string) (string, error) {
	latest, err := LastRun(cs, resourceName, ns, kind)
	if err != nil {
		return "", err
	}
	return latest.Name, nil
}

// LastRun returns the last taskrun for a given task/clustertask
func LastRun(cs *cli.Clients, resourceName, ns, kind string) (*v1.TaskRun, error) {
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

	var trs *v1.TaskRunList
	if err := actions.ListV1(taskrunGroupResource, cs, options, ns, &trs); err != nil {
		return nil, err
	}

	if len(trs.Items) == 0 {
		return nil, fmt.Errorf("no TaskRuns related to %s %s found in namespace %s", kind, resourceName, ns)
	}

	if kind == "Task" {
		trs.Items = FilterByRef(trs.Items, kind)
	}

	latest := trs.Items[0]
	for _, tr := range trs.Items {
		if tr.CreationTimestamp.Time.After(latest.CreationTimestamp.Time) {
			latest = tr
		}
	}

	return &latest, nil
}

// this will filter the taskrun which have reference to Task or ClusterTask
func FilterByRef(taskruns []v1.TaskRun, kind string) []v1.TaskRun {
	var filtered []v1.TaskRun
	for _, taskrun := range taskruns {
		if string(taskrun.Spec.TaskRef.Kind) == kind {
			filtered = append(filtered, taskrun)
		}
	}
	return filtered
}
