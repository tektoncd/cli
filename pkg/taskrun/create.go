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
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/task"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func Create(c *cli.Clients, tr *v1beta1.TaskRun, opts metav1.CreateOptions, ns string) (*v1beta1.TaskRun, error) {
	trGVR, err := actions.GetGroupVersionResource(trGroupResource, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if trGVR.Version == "v1alpha1" {
		v1alpha1TaskRun := ConvertFrom(tr)
		if err != nil {
			return nil, err
		}
		return createUnstructured(v1alpha1TaskRun, c, opts, ns)
	}

	return createUnstructured(tr, c, opts, ns)
}

func createUnstructured(obj runtime.Object, c *cli.Clients, opts metav1.CreateOptions, ns string) (*v1beta1.TaskRun, error) {
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	unstructuredTR := &unstructured.Unstructured{
		Object: object,
	}
	newUnstructuredTR, err := actions.Create(trGroupResource, c, unstructuredTR, ns, opts)
	if err != nil {
		return nil, err
	}
	var taskrun *v1beta1.TaskRun
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(newUnstructuredTR.UnstructuredContent(), &taskrun); err != nil {
		return nil, err
	}
	return taskrun, nil
}

// convert v1beta1 Taskrun to v1alpha1 Taskrun to support backward compatibility
func ConvertFrom(tr *v1beta1.TaskRun) *v1alpha1.TaskRun {
	downtr := &v1alpha1.TaskRun{}
	downtr.Kind = "TaskRun"
	downtr.APIVersion = "tekton.dev/v1alpha1"
	downtr.ObjectMeta = tr.ObjectMeta
	downtr.Status = tr.Status
	if tr.Spec.Resources != nil {
		if len(tr.Spec.Resources.Inputs) > 0 {
			if downtr.Spec.Inputs == nil {
				downtr.Spec.Inputs = &v1alpha1.TaskRunInputs{}
			}
			downtr.Spec.Inputs.Resources = make([]v1alpha1.TaskResourceBinding, len(tr.Spec.Resources.Inputs))
			for i, resource := range tr.Spec.Resources.Inputs {
				downtr.Spec.Inputs.Resources[i] = v1alpha1.TaskResourceBinding{
					PipelineResourceBinding: v1alpha1.PipelineResourceBinding{
						Name:         resource.Name,
						ResourceRef:  resource.ResourceRef,
						ResourceSpec: resource.ResourceSpec,
					},
					Paths: resource.Paths,
				}
			}
		}

		if len(tr.Spec.Resources.Outputs) > 0 {
			if downtr.Spec.Outputs == nil {
				downtr.Spec.Outputs = &v1alpha1.TaskRunOutputs{}
			}
			downtr.Spec.Outputs.Resources = make([]v1alpha1.TaskResourceBinding, len(tr.Spec.Resources.Outputs))
			for i, resource := range tr.Spec.Resources.Outputs {
				downtr.Spec.Outputs.Resources[i] = v1alpha1.TaskResourceBinding{
					PipelineResourceBinding: v1alpha1.PipelineResourceBinding{
						Name:         resource.Name,
						ResourceRef:  resource.ResourceRef,
						ResourceSpec: resource.ResourceSpec,
					},
					Paths: resource.Paths,
				}
			}
		}
	}

	if tr.Spec.Params != nil && len(tr.Spec.Params) > 0 {
		if downtr.Spec.Inputs == nil {
			downtr.Spec.Inputs = &v1alpha1.TaskRunInputs{}
		}
		downtr.Spec.Inputs.Params = make([]v1alpha1.Param, len(tr.Spec.Params))
		for i, param := range tr.Spec.Params {
			downtr.Spec.Inputs.Params[i] = *param.DeepCopy()
		}
	}

	if tr.Spec.TaskSpec != nil {
		downtr.Spec.TaskSpec = &v1alpha1.TaskSpec{}
		downtr.Spec.TaskSpec = task.SpecConvertFrom(tr.Spec.TaskSpec)
	}

	downtr.Spec.TaskRef = tr.Spec.TaskRef
	downtr.Spec.ServiceAccountName = tr.Spec.ServiceAccountName
	downtr.Spec.Status = tr.Spec.Status
	downtr.Spec.PodTemplate = tr.Spec.PodTemplate
	downtr.Spec.Workspaces = tr.Spec.Workspaces
	downtr.Spec.Timeout = tr.Spec.Timeout

	return downtr
}
