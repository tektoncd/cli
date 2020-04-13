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

package validate

import (
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNamespaceExists_Invalid_Namespace(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Kube: cs.Kube}
	p.SetNamespace("foo")

	err := NamespaceExists(p)
	test.AssertOutput(t, "namespaces \"foo\" not found", err.Error())
}

func TestNamespaceExists_Valid_Namespace(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Kube: cs.Kube}

	err := NamespaceExists(p)
	test.AssertOutput(t, nil, err)
}

func TestTaskRefExists_Present(t *testing.T) {
	spec := v1alpha1.TaskRunSpec{
		TaskRef: &v1alpha1.TaskRef{
			Name: "Task",
		},
	}

	output := TaskRefExists(spec)
	test.AssertOutput(t, "Task", output)
}

func TestTaskRefExists_Not_Present(t *testing.T) {
	spec := v1alpha1.TaskRunSpec{
		TaskRef: nil,
	}

	output := TaskRefExists(spec)
	test.AssertOutput(t, "", output)
}

func TestPipelineRefExists_Present(t *testing.T) {
	spec := v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{
			Name: "Pipeline",
		},
	}

	output := PipelineRefExists(spec)
	test.AssertOutput(t, "Pipeline", output)
}

func TestPipelineRefExists_Not_Present(t *testing.T) {
	spec := v1beta1.PipelineRunSpec{
		PipelineRef: nil,
	}

	output := PipelineRefExists(spec)
	test.AssertOutput(t, "", output)
}

func TestPipelineResourceRefExists_Present(t *testing.T) {
	res := v1beta1.PipelineResourceBinding{
		ResourceRef: &v1beta1.PipelineResourceRef{
			Name: "Resource",
		},
	}

	output := PipelineResourceRefExists(res)
	test.AssertOutput(t, "Resource", output)
}

func TestPipelineResourceRefExists_Not_Present(t *testing.T) {
	res := v1beta1.PipelineResourceBinding{
		ResourceRef: nil,
	}

	output := PipelineResourceRefExists(res)
	test.AssertOutput(t, "", output)
}

func TestTaskResourceRefExists_Present(t *testing.T) {
	res := v1alpha1.TaskResourceBinding{
		PipelineResourceBinding: v1alpha1.PipelineResourceBinding{
			ResourceRef: &v1alpha1.PipelineResourceRef{
				Name: "Resource",
			},
		},
	}

	output := TaskResourceRefExists(res)
	test.AssertOutput(t, "Resource", output)
}

func TestTaskResourceRefExists_Not_Present(t *testing.T) {
	res := v1alpha1.TaskResourceBinding{
		PipelineResourceBinding: v1alpha1.PipelineResourceBinding{
			ResourceRef: nil,
		},
	}

	output := TaskResourceRefExists(res)
	test.AssertOutput(t, "", output)
}

func TestStepReasonExists_Terminated_Not_Present(t *testing.T) {
	state := v1alpha1.StepState{}

	output := StepReasonExists(state)
	test.AssertOutput(t, "---", output)
}

func TestStepReasonExists_Terminated_Present(t *testing.T) {
	state := v1alpha1.StepState{
		ContainerState: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				Reason: "Completed",
			},
		},
	}

	output := StepReasonExists(state)
	test.AssertOutput(t, "Completed", output)
}

func TestStepReasonExists_Running_Present(t *testing.T) {
	state := v1alpha1.StepState{
		ContainerState: corev1.ContainerState{
			Running: &corev1.ContainerStateRunning{
				StartedAt: metav1.Time{
					Time: time.Now(),
				},
			},
		},
	}

	output := StepReasonExists(state)
	test.AssertOutput(t, "Running", output)
}

func TestStepReasonExists_Waiting_Present(t *testing.T) {
	state := v1alpha1.StepState{
		ContainerState: corev1.ContainerState{
			Waiting: &corev1.ContainerStateWaiting{
				Reason: "PodInitializing",
			},
		},
	}

	output := StepReasonExists(state)
	test.AssertOutput(t, "PodInitializing", output)
}

func TestSidecarReasonExists_Terminated_Not_Present(t *testing.T) {
	state := v1alpha1.SidecarState{}

	output := SidecarReasonExists(state)
	test.AssertOutput(t, "---", output)
}

func TestSidecarReasonExists_Terminated_Present(t *testing.T) {
	state := v1alpha1.SidecarState{
		ContainerState: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				Reason: "Completed",
			},
		},
	}

	output := SidecarReasonExists(state)
	test.AssertOutput(t, "Completed", output)
}

func TestSidecarReasonExists_Running_Present(t *testing.T) {
	state := v1alpha1.SidecarState{
		ContainerState: corev1.ContainerState{
			Running: &corev1.ContainerStateRunning{
				StartedAt: metav1.Time{
					Time: time.Now(),
				},
			},
		},
	}

	output := SidecarReasonExists(state)
	test.AssertOutput(t, "Running", output)
}

func TestSidecarReasonExists_Waiting_Present(t *testing.T) {
	state := v1alpha1.SidecarState{
		ContainerState: corev1.ContainerState{
			Waiting: &corev1.ContainerStateWaiting{
				Reason: "PodInitializing",
			},
		},
	}

	output := SidecarReasonExists(state)
	test.AssertOutput(t, "PodInitializing", output)
}
