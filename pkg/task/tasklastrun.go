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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//Todo
// changing this LastRun to use dynamic client leads to multiple other change in start commands
// keeping this as it is, will change this while working on start commands
//LastRun returns the last taskrun for a given task
func LastRun(tekton versioned.Interface, task string, ns string, kind string) (*v1alpha1.TaskRun, error) {
	options := metav1.ListOptions{}
	if task != "" {
		options = metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tekton.dev/task=%s", task),
		}
	}

	runs, err := tekton.TektonV1alpha1().TaskRuns(ns).List(options)
	if err != nil {
		return nil, err
	}

	if len(runs.Items) == 0 {
		return nil, fmt.Errorf("no taskruns related to %s %s found in namespace %s", kind, task, ns)
	}

	latest := runs.Items[0]
	for _, run := range runs.Items {
		if run.CreationTimestamp.Time.After(latest.CreationTimestamp.Time) {
			latest = run
		}
	}

	return &latest, nil
}

//DynamicLastRun returns the last taskrun for a given task
func DynamicLastRun(cs *cli.Clients, task string, ns string, kind string) (*v1beta1.TaskRun, error) {
	options := metav1.ListOptions{}
	if task != "" {
		options = metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tekton.dev/task=%s", task),
		}
	}

	runs, err := trlist.TaskRuns(cs, options, ns)
	if err != nil {
		return nil, err
	}

	if len(runs.Items) == 0 {
		return nil, fmt.Errorf("no taskruns related to %s %s found in namespace %s", kind, task, ns)
	}

	latest := runs.Items[0]
	for _, run := range runs.Items {
		if run.CreationTimestamp.Time.After(latest.CreationTimestamp.Time) {
			latest = run
		}
	}

	return &latest, nil
}
