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

package taskrun

import (
	"errors"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	k8stest "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestTaskRunCancel(t *testing.T) {
	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("taskrun-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("task")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonRunning,
				}),
			),
		),
		tb.TaskRun("taskrun-2",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "failure-task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("failure-task")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
	}

	trs2 := []*v1alpha1.TaskRun{
		tb.TaskRun("failure-taskrun-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "failure-task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("failure-task")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonFailed,
				}),
			),
		),
	}

	trs3 := []*v1alpha1.TaskRun{
		tb.TaskRun("cancel-taskrun-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunLabel("tekton.dev/task", "cancel-task"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("cancel-task")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionFalse,
					Reason: "TaskRunCancelled",
				}),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}
	type clients struct {
		pipelineClient pipelinetest.Clients
		dynamicClient  dynamic.Interface
	}

	seeds := make([]clients, 0)
	failures := make([]clients, 0)
	cancels := make([]clients, 0)

	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trs[0], versionA1),
		cb.UnstructuredTR(trs[1], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs2, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs2, Namespaces: ns})
	cs2.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"task", "taskrun"})
	tdc2 := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{
			{Verb: "patch",
				Resource: "taskruns",
				Action: func(action k8stest.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("test error")
				}}}}
	dc2, err := tdc2.Client(
		cb.UnstructuredTR(trs2[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs3, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs3, Namespaces: ns})
	cs3.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"task", "taskrun"})
	tdc3 := testDynamic.Options{}
	dc3, err := tdc3.Client(
		cb.UnstructuredTR(trs3[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	seeds = append(seeds, clients{pipelineClient: cs, dynamicClient: dc})
	failures = append(failures, clients{pipelineClient: cs2, dynamicClient: dc2})
	cancels = append(cancels, clients{pipelineClient: cs3, dynamicClient: dc3})

	testParams := []struct {
		name      string
		command   []string
		dynamic   dynamic.Interface
		input     pipelinetest.Clients
		wantError bool
		want      string
	}{
		{
			name:      "Invalid namespace",
			command:   []string{"cancel", "taskrun-1", "-n", "invalid"},
			dynamic:   seeds[0].dynamicClient,
			input:     seeds[0].pipelineClient,
			wantError: true,
			want:      "namespaces \"invalid\" not found",
		},
		{
			name:      "Canceling taskrun successfully",
			command:   []string{"cancel", "taskrun-1", "-n", "ns"},
			dynamic:   seeds[0].dynamicClient,
			input:     seeds[0].pipelineClient,
			wantError: false,
			want:      "TaskRun cancelled: taskrun-1\n",
		},
		{
			name:      "Not found taskrun",
			command:   []string{"cancel", "nonexistent", "-n", "ns"},
			dynamic:   seeds[0].dynamicClient,
			input:     seeds[0].pipelineClient,
			wantError: true,
			want:      "failed to find TaskRun: nonexistent",
		},
		{
			name:      "Failed canceling taskrun",
			command:   []string{"cancel", "failure-taskrun-1", "-n", "ns"},
			dynamic:   failures[0].dynamicClient,
			input:     failures[0].pipelineClient,
			wantError: true,
			want:      "failed to cancel TaskRun failure-taskrun-1: test error",
		},
		{
			name:      "Failed canceling taskrun that succeeded",
			command:   []string{"cancel", "taskrun-2", "-n", "ns"},
			dynamic:   seeds[0].dynamicClient,
			input:     seeds[0].pipelineClient,
			wantError: true,
			want:      "failed to cancel TaskRun taskrun-2: TaskRun has already finished execution",
		},
		{
			name:      "Failed canceling taskrun that was cancelled",
			command:   []string{"cancel", "cancel-taskrun-1", "-n", "ns"},
			dynamic:   cancels[0].dynamicClient,
			input:     cancels[0].pipelineClient,
			wantError: true,
			want:      "failed to cancel TaskRun cancel-taskrun-1: TaskRun has already finished execution",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube, Dynamic: tp.dynamic}
			taskrun := Command(p)

			out, err := test.ExecuteCommand(taskrun, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}

func TestTaskRunCancel_v1beta1(t *testing.T) {
	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "taskrun-1",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonRunning,
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "taskrun-2",
				Labels:    map[string]string{"tekton.dev/task": "failure-task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "failure-task",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
			},
		},
	}

	trs2 := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "failure-taskrun-1",
				Labels:    map[string]string{"tekton.dev/task": "failure-task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "failure-task",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonFailed,
						},
					},
				},
			},
		},
	}

	trs3 := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "cancel-taskrun-1",
				Labels:    map[string]string{"tekton.dev/task": "cancel-task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "cancel-task",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: "TaskRunCancelled",
						},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}
	type clients struct {
		pipelineClient pipelinev1beta1test.Clients
		dynamicClient  dynamic.Interface
	}

	seeds := make([]clients, 0)
	failures := make([]clients, 0)
	cancels := make([]clients, 0)

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionB1),
		cb.UnstructuredV1beta1TR(trs[1], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs2, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs2, Namespaces: ns})
	cs2.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun"})
	tdc2 := testDynamic.Options{PrependReactors: []testDynamic.PrependOpt{
		{Verb: "patch",
			Resource: "taskruns",
			Action: func(action k8stest.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("test error")
			}}}}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1TR(trs2[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs3, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs3, Namespaces: ns})
	cs3.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"task", "taskrun"})
	tdc3 := testDynamic.Options{}
	dc3, err := tdc3.Client(
		cb.UnstructuredV1beta1TR(trs3[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	seeds = append(seeds, clients{pipelineClient: cs, dynamicClient: dc})
	failures = append(failures, clients{pipelineClient: cs2, dynamicClient: dc2})
	cancels = append(cancels, clients{pipelineClient: cs3, dynamicClient: dc3})

	testParams := []struct {
		name      string
		command   []string
		dynamic   dynamic.Interface
		input     pipelinev1beta1test.Clients
		wantError bool
		want      string
	}{
		{
			name:      "Invalid namespace",
			command:   []string{"cancel", "taskrun-1", "-n", "invalid"},
			dynamic:   seeds[0].dynamicClient,
			input:     seeds[0].pipelineClient,
			wantError: true,
			want:      "namespaces \"invalid\" not found",
		},
		{
			name:      "Canceling taskrun successfully",
			command:   []string{"cancel", "taskrun-1", "-n", "ns"},
			dynamic:   seeds[0].dynamicClient,
			input:     seeds[0].pipelineClient,
			wantError: false,
			want:      "TaskRun cancelled: taskrun-1\n",
		},
		{
			name:      "Not found taskrun",
			command:   []string{"cancel", "nonexistent", "-n", "ns"},
			dynamic:   seeds[0].dynamicClient,
			input:     seeds[0].pipelineClient,
			wantError: true,
			want:      "failed to find TaskRun: nonexistent",
		},
		{
			name:      "Failed canceling taskrun",
			command:   []string{"cancel", "failure-taskrun-1", "-n", "ns"},
			dynamic:   failures[0].dynamicClient,
			input:     failures[0].pipelineClient,
			wantError: true,
			want:      "failed to cancel TaskRun failure-taskrun-1: test error",
		},
		{
			name:      "Failed canceling taskrun that succeeded",
			command:   []string{"cancel", "taskrun-2", "-n", "ns"},
			dynamic:   seeds[0].dynamicClient,
			input:     seeds[0].pipelineClient,
			wantError: true,
			want:      "failed to cancel TaskRun taskrun-2: TaskRun has already finished execution",
		},
		{
			name:      "Failed canceling taskrun that was cancelled",
			command:   []string{"cancel", "cancel-taskrun-1", "-n", "ns"},
			dynamic:   cancels[0].dynamicClient,
			input:     cancels[0].pipelineClient,
			wantError: true,
			want:      "failed to cancel TaskRun cancel-taskrun-1: TaskRun has already finished execution",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube, Dynamic: tp.dynamic}
			taskrun := Command(p)

			out, err := test.ExecuteCommand(taskrun, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}
