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
	"os"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var taskGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "tasks"}

func GetAllTaskNames(gr schema.GroupVersionResource, c *cli.Clients, ns string) ([]string, error) {
	var tasks *v1.TaskList
	if err := actions.ListV1(gr, c, metav1.ListOptions{}, ns, &tasks); err != nil {
		return nil, fmt.Errorf("failed to list Tasks from namespace %s: %v", ns, err)
	}

	ret := []string{}
	for _, item := range tasks.Items {
		ret = append(ret, item.ObjectMeta.Name)
	}
	return ret, nil
}

// TODO: remove as all the function uses are moved to new func
// Get will fetch the task resource based on the task name
func Get(c *cli.Clients, taskname string, opts metav1.GetOptions, ns string) (*v1beta1.Task, error) {
	unstructuredT, err := actions.Get(taskGroupResource, c.Dynamic, c.Tekton.Discovery(), taskname, ns, opts)
	if err != nil {
		return nil, err
	}

	var task *v1beta1.Task
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredT.UnstructuredContent(), &task); err != nil {
		fmt.Fprintf(os.Stderr, "failed to get task from %s namespace \n", ns)
		return nil, err
	}
	return task, nil
}

func Create(c *cli.Clients, t *v1beta1.Task, opts metav1.CreateOptions, ns string) (*v1beta1.Task, error) {
	object, err := runtime.DefaultUnstructuredConverter.ToUnstructured(t)
	if err != nil {
		return nil, err
	}
	unstructuredT := &unstructured.Unstructured{
		Object: object,
	}
	newUnstructuredT, err := actions.Create(taskGroupResource, c, unstructuredT, ns, opts)
	if err != nil {
		return nil, err
	}
	var task *v1beta1.Task
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(newUnstructuredT.UnstructuredContent(), &task); err != nil {
		return nil, err
	}
	return task, nil
}
