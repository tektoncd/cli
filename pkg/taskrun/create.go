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
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func Create(c *cli.Clients, tr *v1beta1.TaskRun, opts metav1.CreateOptions, ns string) (*v1beta1.TaskRun, error) {
	gvr, err := actions.GetGroupVersionResource(taskrunGroupResource, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if gvr.Version == "v1" {
		trv1 := v1.TaskRun{}
		err = tr.ConvertTo(context.Background(), &trv1)
		if err != nil {
			return nil, err
		}
		trv1.Kind = "TaskRun"
		trv1.APIVersion = "tekton.dev/v1"

		object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&trv1)
		unstructuredTR := &unstructured.Unstructured{
			Object: object,
		}
		newUnstructuredTR, err := actions.Create(taskrunGroupResource, c, unstructuredTR, ns, opts)
		if err != nil {
			return nil, err
		}
		var taskrun v1.TaskRun
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(newUnstructuredTR.UnstructuredContent(), &taskrun); err != nil {
			return nil, err
		}

		taskrunv1beta1 := v1beta1.TaskRun{}
		err = taskrunv1beta1.ConvertFrom(context.Background(), &taskrun)
		if err != nil {
			return nil, err
		}
		taskrunv1beta1.Kind = "TaskRun"
		taskrunv1beta1.APIVersion = "tekton.dev/v1beta1"
		return &taskrunv1beta1, nil
	}

	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(tr)
	unstructuredTR := &unstructured.Unstructured{
		Object: object,
	}
	newUnstructuredTR, err := actions.Create(taskrunGroupResource, c, unstructuredTR, ns, opts)
	if err != nil {
		return nil, err
	}
	var taskrun *v1beta1.TaskRun
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(newUnstructuredTR.UnstructuredContent(), &taskrun); err != nil {
		return nil, err
	}
	return taskrun, nil
}
