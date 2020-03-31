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

package description

import (
	"testing"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_SortStepStatesByStartTime_Waiting_Not_Nil(t *testing.T) {
	stepStates := []v1alpha1.StepState{
		{
			Name: "step1",
			ContainerState: corev1.ContainerState{
				Waiting: &corev1.ContainerStateWaiting{
					Reason: "PodInitializing",
				},
			},
		},
		{
			Name: "step2",
			ContainerState: corev1.ContainerState{
				Waiting: &corev1.ContainerStateWaiting{
					Reason: "PodInitializing",
				},
			},
		},
	}

	sortedSteps := sortStepStatesByStartTime(stepStates)

	element0 := sortedSteps[0].Name
	if element0 != "step1" {
		t.Errorf("sortStepStatesByStartTime should be step1 but returned: %s", element0)
	}

	element1 := sortedSteps[1].Name
	if element1 != "step2" {
		t.Errorf("sortStepStatesByStartTime should be step2 but returned: %s", element1)
	}
}

func Test_SortStepStatesByStartTime_Step1_Running(t *testing.T) {
	stepStates := []v1alpha1.StepState{
		{
			Name: "step1",
			ContainerState: corev1.ContainerState{
				Running: &corev1.ContainerStateRunning{
					StartedAt: metav1.Time{Time: time.Now()},
				},
			},
		},
		{
			Name: "step2",
			ContainerState: corev1.ContainerState{
				Waiting: &corev1.ContainerStateWaiting{
					Reason: "PodInitializing",
				},
			},
		},
	}

	sortedSteps := sortStepStatesByStartTime(stepStates)

	element0 := sortedSteps[0].Name
	if element0 != "step1" {
		t.Errorf("sortStepStatesByStartTime should be step1 but returned: %s", element0)
	}

	element1 := sortedSteps[1].Name
	if element1 != "step2" {
		t.Errorf("sortStepStatesByStartTime should be step2 but returned: %s", element1)
	}
}

func Test_SortStepStatesByStartTime_Step2_Running(t *testing.T) {
	stepStates := []v1alpha1.StepState{
		{
			Name: "step1",
			ContainerState: corev1.ContainerState{
				Waiting: &corev1.ContainerStateWaiting{
					Reason: "PodInitializing",
				},
			},
		},
		{
			Name: "step2",
			ContainerState: corev1.ContainerState{
				Running: &corev1.ContainerStateRunning{
					StartedAt: metav1.Time{Time: time.Now()},
				},
			},
		},
	}

	sortedSteps := sortStepStatesByStartTime(stepStates)

	element0 := sortedSteps[0].Name
	if element0 != "step2" {
		t.Errorf("sortStepStatesByStartTime should be step2 but returned: %s", element0)
	}

	element1 := sortedSteps[1].Name
	if element1 != "step1" {
		t.Errorf("sortStepStatesByStartTime should be step1 but returned: %s", element1)
	}
}

func Test_SortStepStatesByStartTime_Both_Steps_Running(t *testing.T) {
	stepStates := []v1alpha1.StepState{
		{
			Name: "step1",
			ContainerState: corev1.ContainerState{
				Running: &corev1.ContainerStateRunning{
					StartedAt: metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				},
			},
		},
		{
			Name: "step2",
			ContainerState: corev1.ContainerState{
				Running: &corev1.ContainerStateRunning{
					StartedAt: metav1.Time{Time: time.Now().Add(-2 * time.Minute)},
				},
			},
		},
	}

	sortedSteps := sortStepStatesByStartTime(stepStates)

	element0 := sortedSteps[0].Name
	if element0 != "step2" {
		t.Errorf("sortStepStatesByStartTime should be step2 but returned: %s", element0)
	}

	element1 := sortedSteps[1].Name
	if element1 != "step1" {
		t.Errorf("sortStepStatesByStartTime should be step1 but returned: %s", element1)
	}
}

func Test_SortStepStatesByStartTime_Steps_Terminated_And_Running(t *testing.T) {
	stepStates := []v1alpha1.StepState{
		{
			Name: "step1",
			ContainerState: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{
					StartedAt: metav1.Time{Time: time.Now().Add(-4 * time.Minute)},
				},
			},
		},
		{
			Name: "step2",
			ContainerState: corev1.ContainerState{
				Running: &corev1.ContainerStateRunning{
					StartedAt: metav1.Time{Time: time.Now().Add(-2 * time.Minute)},
				},
			},
		},
		{
			Name: "step3",
			ContainerState: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{
					StartedAt: metav1.Time{Time: time.Now().Add(-3 * time.Minute)},
				},
			},
		},
		{
			Name: "step4",
			ContainerState: corev1.ContainerState{
				Running: &corev1.ContainerStateRunning{
					StartedAt: metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				},
			},
		},
	}

	sortedSteps := sortStepStatesByStartTime(stepStates)

	element0 := sortedSteps[0].Name
	if element0 != "step1" {
		t.Errorf("sortStepStatesByStartTime should be step1 but returned: %s", element0)
	}

	element1 := sortedSteps[1].Name
	if element1 != "step3" {
		t.Errorf("sortStepStatesByStartTime should be step3 but returned: %s", element1)
	}

	element2 := sortedSteps[2].Name
	if element2 != "step2" {
		t.Errorf("sortStepStatesByStartTime should be step2 but returned: %s", element2)
	}

	element3 := sortedSteps[3].Name
	if element3 != "step4" {
		t.Errorf("sortStepStatesByStartTime should be step3 but returned: %s", element3)
	}
}
