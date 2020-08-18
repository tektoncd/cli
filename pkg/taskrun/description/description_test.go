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

package description

import (
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_SortStepStatesByStartTime_Waiting_Not_Nil(t *testing.T) {
	stepStates := []v1beta1.StepState{
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
	stepStates := []v1beta1.StepState{
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
	stepStates := []v1beta1.StepState{
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
	stepStates := []v1beta1.StepState{
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
	stepStates := []v1beta1.StepState{
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

func TestTaskRefExists_Present(t *testing.T) {
	spec := v1beta1.TaskRunSpec{
		TaskRef: &v1beta1.TaskRef{
			Name: "Task",
		},
	}

	output := taskRefExists(spec)
	test.AssertOutput(t, "Task", output)
}

func TestTaskRefExists_Not_Present(t *testing.T) {
	spec := v1beta1.TaskRunSpec{
		TaskRef: nil,
	}

	output := taskRefExists(spec)
	test.AssertOutput(t, "", output)
}

func TestTaskResourceRefExists_Present(t *testing.T) {
	res := v1beta1.TaskResourceBinding{
		PipelineResourceBinding: v1beta1.PipelineResourceBinding{
			ResourceRef: &v1beta1.PipelineResourceRef{
				Name: "Resource",
			},
		},
	}

	output := taskResourceRefExists(res)
	test.AssertOutput(t, "Resource", output)
}

func TestTaskResourceRefExists_Not_Present(t *testing.T) {
	res := v1beta1.TaskResourceBinding{
		PipelineResourceBinding: v1beta1.PipelineResourceBinding{
			ResourceRef: nil,
		},
	}

	output := taskResourceRefExists(res)
	test.AssertOutput(t, "", output)
}

func TestStepReasonExists_Terminated_Not_Present(t *testing.T) {
	state := v1beta1.StepState{}

	output := stepReasonExists(state)
	test.AssertOutput(t, "---", output)
}

func TestStepReasonExists_Terminated_Present(t *testing.T) {
	state := v1beta1.StepState{
		ContainerState: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				Reason: "Completed",
			},
		},
	}

	output := stepReasonExists(state)
	test.AssertOutput(t, "Completed", output)
}

func TestStepReasonExists_Running_Present(t *testing.T) {
	state := v1beta1.StepState{
		ContainerState: corev1.ContainerState{
			Running: &corev1.ContainerStateRunning{
				StartedAt: metav1.Time{
					Time: time.Now(),
				},
			},
		},
	}

	output := stepReasonExists(state)
	test.AssertOutput(t, "Running", output)
}

func TestStepReasonExists_Waiting_Present(t *testing.T) {
	state := v1beta1.StepState{
		ContainerState: corev1.ContainerState{
			Waiting: &corev1.ContainerStateWaiting{
				Reason: "PodInitializing",
			},
		},
	}

	output := stepReasonExists(state)
	test.AssertOutput(t, "PodInitializing", output)
}

func TestSidecarReasonExists_Terminated_Not_Present(t *testing.T) {
	state := v1beta1.SidecarState{}

	output := sidecarReasonExists(state)
	test.AssertOutput(t, "---", output)
}

func TestSidecarReasonExists_Terminated_Present(t *testing.T) {
	state := v1beta1.SidecarState{
		ContainerState: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				Reason: "Completed",
			},
		},
	}

	output := sidecarReasonExists(state)
	test.AssertOutput(t, "Completed", output)
}

func TestSidecarReasonExists_Running_Present(t *testing.T) {
	state := v1beta1.SidecarState{
		ContainerState: corev1.ContainerState{
			Running: &corev1.ContainerStateRunning{
				StartedAt: metav1.Time{
					Time: time.Now(),
				},
			},
		},
	}

	output := sidecarReasonExists(state)
	test.AssertOutput(t, "Running", output)
}

func TestSidecarReasonExists_Waiting_Present(t *testing.T) {
	state := v1beta1.SidecarState{
		ContainerState: corev1.ContainerState{
			Waiting: &corev1.ContainerStateWaiting{
				Reason: "PodInitializing",
			},
		},
	}

	output := sidecarReasonExists(state)
	test.AssertOutput(t, "PodInitializing", output)
}

func TestTaskRunDefaultSetting(t *testing.T) {
	testData := []struct {
		input  *v1beta1.TaskRun
		expect []v1beta1.Param
	}{
		{
			input: &v1beta1.TaskRun{
				Spec: v1beta1.TaskRunSpec{
					Params: []v1beta1.Param{
						{
							Name: "param1",
							Value: v1beta1.ArrayOrString{
								Type:      v1beta1.ParamTypeString,
								StringVal: "real1",
							},
						},
					},
				},
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						TaskSpec: &v1beta1.TaskSpec{
							Params: []v1beta1.ParamSpec{
								{
									Name: "param1",
									Type: v1beta1.ParamTypeString,
									Default: &v1beta1.ArrayOrString{
										Type:      v1beta1.ParamTypeString,
										StringVal: "test1",
									},
								},
								{
									Name: "param2",
									Type: v1beta1.ParamTypeString,
									Default: &v1beta1.ArrayOrString{
										Type:      v1beta1.ParamTypeString,
										StringVal: "test2",
									},
								},
							},
						},
					},
				},
			},
			expect: []v1beta1.Param{
				{
					Name: "param1",
					Value: v1beta1.ArrayOrString{
						Type:      v1beta1.ParamTypeString,
						StringVal: "real1",
					},
				},
				{
					Name: "param2",
					Value: v1beta1.ArrayOrString{
						Type:      v1beta1.ParamTypeString,
						StringVal: "test2",
					},
				},
			},
		},
		{
			input: &v1beta1.TaskRun{
				Spec: v1beta1.TaskRunSpec{
					Params: []v1beta1.Param{
						{
							Name: "param1",
							Value: v1beta1.ArrayOrString{
								Type:     v1beta1.ParamTypeArray,
								ArrayVal: []string{"real1"},
							},
						},
					},
				},
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						TaskSpec: &v1beta1.TaskSpec{
							Params: []v1beta1.ParamSpec{
								{
									Name: "param1",
									Type: v1beta1.ParamTypeArray,
									Default: &v1beta1.ArrayOrString{
										Type:     v1beta1.ParamTypeArray,
										ArrayVal: []string{"test1", "test11", "test111"},
									},
								},
								{
									Name: "param2",
									Type: v1beta1.ParamTypeArray,
									Default: &v1beta1.ArrayOrString{
										Type:     v1beta1.ParamTypeArray,
										ArrayVal: []string{"test2", "test22", "test222"},
									},
								},
							},
						},
					},
				},
			},
			expect: []v1beta1.Param{
				{
					Name: "param1",
					Value: v1beta1.ArrayOrString{
						Type:     v1beta1.ParamTypeArray,
						ArrayVal: []string{"real1"},
					},
				},
				{
					Name: "param2",
					Value: v1beta1.ArrayOrString{
						Type:     v1beta1.ParamTypeArray,
						ArrayVal: []string{"test2", "test22", "test222"},
					},
				},
			},
		},
	}

	for _, td := range testData {
		err := SetDefault(&td.input.Spec.Params, td.input.Status.TaskSpec.Params)
		if err != nil {
			t.Errorf(err.Error())
		} else if deep.Equal(td.input.Spec.Params, td.expect) != nil {
			t.Errorf("setDefault should be %v but returned: %v", td.expect, td.input.Spec.Params)
		}
	}
}
