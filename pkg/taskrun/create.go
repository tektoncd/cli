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

package taskrun

import (
	"context"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Create(c *cli.Clients, tr *v1beta1.TaskRun, opts metav1.CreateOptions, ns string) (*v1beta1.TaskRun, error) {
	trGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}
	trGVR, err := actions.GetGroupVersionResource(trGroupResource, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	var (
		taskrun           *v1beta1.TaskRun
		newUnstructuredTR *unstructured.Unstructured
	)

	if trGVR.Version == "v1alpha1" {
		v1alpha1TaskRun := v1alpha1.TaskRun{}
		v1alpha1TaskRun.Kind = "TaskRun"
		v1alpha1TaskRun.APIVersion = "tekton.dev/v1alpha1"

		err := v1alpha1TaskRun.ConvertDown(context.Background(), tr)
		if err != nil {
			return nil, err
		}
		unstructuredTR, err := convertToUnstructuredTaskRun(&v1alpha1TaskRun)
		if err != nil {
			return nil, err
		}
		newUnstructuredTR, err = actions.Create(*trGVR, c, unstructuredTR, ns, opts)
		if err != nil {
			return nil, err
		}

		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(newUnstructuredTR.UnstructuredContent(), &taskrun); err != nil {
			return nil, err
		}

		return taskrun, nil
	}

	unstructuredTR, err := convertToUnstructuredTaskRun(tr)
	if err != nil {
		return nil, err
	}
	newUnstructuredTR, err = actions.Create(*trGVR, c, unstructuredTR, ns, opts)
	if err != nil {
		return nil, err
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(newUnstructuredTR.UnstructuredContent(), &taskrun); err != nil {
		return nil, err
	}

	return taskrun, nil
}

func convertToUnstructuredTaskRun(obj runtime.Object) (*unstructured.Unstructured, error) {
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	return &unstructured.Unstructured{
		Object: object,
	}, nil
}
