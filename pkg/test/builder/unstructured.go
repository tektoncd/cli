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

package builder

import (
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func UnstructuredP(pipeline *v1.Pipeline, version string) *unstructured.Unstructured {
	pipeline.APIVersion = "tekton.dev/" + version
	pipeline.Kind = "pipeline"
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pipeline)
	return &unstructured.Unstructured{
		Object: object,
	}
}

func UnstructuredPR(pipelinerun *v1.PipelineRun, version string) *unstructured.Unstructured {
	pipelinerun.APIVersion = "tekton.dev/" + version
	pipelinerun.Kind = "pipelinerun"
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pipelinerun)
	return &unstructured.Unstructured{
		Object: object,
	}
}

func UnstructuredTR(taskrun *v1.TaskRun, version string) *unstructured.Unstructured {
	taskrun.APIVersion = "tekton.dev/" + version
	taskrun.Kind = "taskrun"
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(taskrun)
	return &unstructured.Unstructured{
		Object: object,
	}
}

func UnstructuredT(task *v1.Task, version string) *unstructured.Unstructured {
	task.APIVersion = "tekton.dev/" + version
	task.Kind = "task"
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(task)
	return &unstructured.Unstructured{
		Object: object,
	}
}

func UnstructuredV1beta1P(pipeline *v1beta1.Pipeline, version string) *unstructured.Unstructured {
	pipeline.APIVersion = "tekton.dev/" + version
	pipeline.Kind = "pipeline"
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pipeline)
	return &unstructured.Unstructured{
		Object: object,
	}
}

func UnstructuredV1beta1PR(pipelinerun *v1beta1.PipelineRun, version string) *unstructured.Unstructured {
	pipelinerun.APIVersion = "tekton.dev/" + version
	pipelinerun.Kind = "pipelinerun"
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pipelinerun)
	return &unstructured.Unstructured{
		Object: object,
	}
}

func UnstructuredV1beta1TR(taskrun *v1beta1.TaskRun, version string) *unstructured.Unstructured {
	taskrun.APIVersion = "tekton.dev/" + version
	taskrun.Kind = "taskrun"
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(taskrun)
	return &unstructured.Unstructured{
		Object: object,
	}
}

func UnstructuredV1beta1CustomRun(customrun *v1beta1.CustomRun, version string) *unstructured.Unstructured {
	customrun.APIVersion = "tekton.dev/" + version
	customrun.Kind = "customrun"
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(customrun)
	return &unstructured.Unstructured{
		Object: object,
	}
}

func UnstructuredV1beta1T(task *v1beta1.Task, version string) *unstructured.Unstructured {
	task.APIVersion = "tekton.dev/" + version
	task.Kind = "task"
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(task)
	return &unstructured.Unstructured{
		Object: object,
	}
}

func UnstructuredV1beta1CT(clustertask *v1beta1.ClusterTask, version string) *unstructured.Unstructured {
	clustertask.APIVersion = "tekton.dev/" + version
	clustertask.Kind = "clustertask"
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(clustertask)
	return &unstructured.Unstructured{
		Object: object,
	}
}

func UnstructuredV1beta1TT(triggertemplate *triggersv1beta1.TriggerTemplate, version string) *unstructured.Unstructured {
	triggertemplate.APIVersion = "triggers.tekton.dev/" + version
	triggertemplate.Kind = "TriggerTemplate"
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(triggertemplate)
	return &unstructured.Unstructured{
		Object: object,
	}
}

func UnstructuredV1beta1TB(triggerbinding *triggersv1beta1.TriggerBinding, version string) *unstructured.Unstructured {
	triggerbinding.APIVersion = "triggers.tekton.dev/" + version
	triggerbinding.Kind = "TriggerBinding"
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(triggerbinding)
	return &unstructured.Unstructured{
		Object: object,
	}
}

func UnstructuredV1beta1CTB(clustertriggerbinding *triggersv1beta1.ClusterTriggerBinding, version string) *unstructured.Unstructured {
	clustertriggerbinding.APIVersion = "triggers.tekton.dev/" + version
	clustertriggerbinding.Kind = "ClusterTriggerBinding"
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(clustertriggerbinding)
	return &unstructured.Unstructured{
		Object: object,
	}
}

func UnstructuredV1beta1EL(eventlistener *triggersv1beta1.EventListener, version string) *unstructured.Unstructured {
	eventlistener.APIVersion = "triggers.tekton.dev/" + version
	eventlistener.Kind = "EventListener"
	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(eventlistener)
	return &unstructured.Unstructured{
		Object: object,
	}
}
