// Copyright © 2023 The Tekton Authors.
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
	"fmt"
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestTaskRunExport_v1beta1(t *testing.T) {
	clock := test.FakeClock()
	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "task-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/task": "task"},
				Annotations: map[string]string{
					"kubectl.kubernetes.io/last-applied-configuration": "test",
					"pipeline.dev": "cli",
				},
				GenerateName: "generate-name",
				Generation:   5,
				UID:          "f54b8b67-ce52-4509-8a4a-f245b093b62e",
			},
			Spec: v1beta1.TaskRunSpec{
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
				},
				Status:        v1beta1.PipelineRunSpecStatusPending,
				StatusMessage: "Pending",
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1TR(taskruns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: namespaces, TaskRuns: taskruns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}

	got, err := test.ExecuteCommand(Command(p), "export", "-n", "ns", "task-run")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTaskRunExport(t *testing.T) {
	clock := test.FakeClock()
	taskruns := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "task-run",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/task": "task"},
				Annotations: map[string]string{
					"kubectl.kubernetes.io/last-applied-configuration": "test",
					"pipeline.dev": "cli",
				},
				GenerateName: "generate-name",
				Generation:   5,
				UID:          "f54b8b67-ce52-4509-8a4a-f245b093b62e",
			},
			Spec: v1.TaskRunSpec{
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
				TaskRef: &v1.TaskRef{
					Name: "task",
				},
				Status:        v1.PipelineRunSpecStatusPending,
				StatusMessage: "Pending",
			},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredTR(taskruns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, TaskRuns: taskruns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}

	got, err := test.ExecuteCommand(Command(p), "export", "-n", "ns", "task-run")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}
