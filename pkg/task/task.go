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
	"context"
	"fmt"
	"os"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var taskGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "tasks"}

func GetAllTaskNames(p cli.Params) ([]string, error) {
	cs, err := p.Clients()
	if err != nil {
		return nil, err
	}

	ps, err := List(cs, metav1.ListOptions{}, p.Namespace())
	if err != nil {
		return nil, err
	}

	ret := []string{}
	for _, item := range ps.Items {
		ret = append(ret, item.ObjectMeta.Name)
	}
	return ret, nil
}

func List(c *cli.Clients, opts metav1.ListOptions, ns string) (*v1beta1.TaskList, error) {
	unstructuredT, err := actions.List(taskGroupResource, c, ns, opts)
	if err != nil {
		return nil, err
	}

	var tasks *v1beta1.TaskList
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredT.UnstructuredContent(), &tasks); err != nil {
		return nil, err
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list tasks from %s namespace \n", ns)
		return nil, err
	}

	return tasks, nil
}

// It will fetch the resource based on the api available and return v1beta1 form
func Get(c *cli.Clients, taskname string, opts metav1.GetOptions, ns string) (*v1beta1.Task, error) {
	gvr, err := actions.GetGroupVersionResource(taskGroupResource, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if gvr.Version == "v1alpha1" {
		task, err := getV1alpha1(c, taskname, opts, ns)
		if err != nil {
			return nil, err
		}
		var taskConverted v1beta1.Task
		err = task.ConvertUp(context.Background(), &taskConverted)
		if err != nil {
			return nil, err
		}
		return &taskConverted, nil
	}
	return GetV1beta1(c, taskname, opts, ns)
}

// It will fetch the resource in v1beta1 struct format
func GetV1beta1(c *cli.Clients, taskname string, opts metav1.GetOptions, ns string) (*v1beta1.Task, error) {
	unstructuredT, err := actions.Get(taskGroupResource, c, taskname, ns, opts)
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

// It will fetch the resource in v1alpha1 struct format
func getV1alpha1(c *cli.Clients, taskname string, opts metav1.GetOptions, ns string) (*v1alpha1.Task, error) {
	unstructuredT, err := actions.Get(taskGroupResource, c, taskname, ns, opts)
	if err != nil {
		return nil, err
	}

	var task *v1alpha1.Task
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredT.UnstructuredContent(), &task); err != nil {
		fmt.Fprintf(os.Stderr, "failed to get task from %s namespace \n", ns)
		return nil, err
	}
	return task, nil
}
