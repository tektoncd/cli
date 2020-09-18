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

package pipelinerun

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	tb "github.com/tektoncd/cli/internal/builder/v1alpha1"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestPipelineRunDescribe_invalid_namespace(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	pipelinerun := Command(p)
	out, err := test.ExecuteCommand(pipelinerun, "desc", "bar", "-n", "invalid")
	if err == nil {
		t.Errorf("Expected error for invalid namespace")
	}
	expected := "Error: failed to find pipelinerun \"bar\"\n"
	test.AssertOutput(t, expected, out)
}

func TestPipelineRunDescribe_not_found(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	_, err = test.ExecuteCommand(pipelinerun, "desc", "bar", "-n", "ns")
	if err == nil {
		t.Errorf("Expected error, did not get any")
	}
	expected := "failed to find pipelinerun \"bar\""
	test.AssertOutput(t, expected, err.Error())
}

func TestPipelineRunDescribe_only_taskrun(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(2*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(5*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
			),
		),
	}

	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunTaskRunsStatus("tr-1", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "t-1",
					Status:           &trs[0].Status,
				}),
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.PipelineRunReasonSuccessful.String(),
				}),
				tb.PipelineRunStartTime(clock.Now()),
				cb.PipelineRunCompletionTime(clock.Now().Add(5*time.Minute)),
			),
		),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_multiple_taskrun_ordering(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(2*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(5*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
			),
		),
		tb.TaskRun("tr-2",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(5*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(9*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
			),
		),
	}

	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunTaskRunsStatus("tr-1", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "t-1",
					Status:           &trs[0].Status,
				}),
				tb.PipelineRunTaskRunsStatus("tr-2", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "t-2",
					Status:           &trs[1].Status,
				}),
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.PipelineRunReasonSuccessful.String(),
				}),
				tb.PipelineRunStartTime(clock.Now()),
				cb.PipelineRunCompletionTime(clock.Now().Add(15*time.Minute)),
			),
		),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
		cb.UnstructuredTR(trs[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))

}

func TestPipelineRunDescribe_failed(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(2*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(5*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Status:  corev1.ConditionFalse,
					Reason:  v1beta1.PipelineRunReasonFailed.String(),
					Message: "Testing tr failed",
				}),
			),
		),
	}

	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline",
				tb.PipelineRunServiceAccountName("test-sa"),
			),
			tb.PipelineRunStatus(
				tb.PipelineRunTaskRunsStatus("tr-1", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "t-1",
					Status:           &trs[0].Status,
				}),
				tb.PipelineRunStatusCondition(apis.Condition{
					Status:  corev1.ConditionFalse,
					Reason:  "Resource not found",
					Message: "Resource test-resource not found in the pipelinerun",
				}),
				tb.PipelineRunStartTime(clock.Now()),
				cb.PipelineRunCompletionTime(clock.Now().Add(5*time.Minute)),
			),
		),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_last_no_PipelineRun_present(t *testing.T) {
	pipelineRuns := []*v1beta1.PipelineRun{}
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns})

	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	out, err := test.ExecuteCommand(pipelinerun, "desc", "--last", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := "No PipelineRuns present in namespace ns\n"
	test.AssertOutput(t, expected, out)
}

func TestPipelineRunDescribe_failed_withoutTRCondition(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(2*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(5*time.Minute)),
			),
		),
	}

	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline",
				tb.PipelineRunServiceAccountName("test-sa"),
			),
			tb.PipelineRunStatus(
				tb.PipelineRunTaskRunsStatus("tr-1", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "t-1",
					Status:           &trs[0].Status,
				}),
				tb.PipelineRunStatusCondition(apis.Condition{
					Status:  corev1.ConditionFalse,
					Reason:  "Resource not found",
					Message: "Resource test-resource not found in the pipelinerun",
				}),
				tb.PipelineRunStartTime(clock.Now()),
				cb.PipelineRunCompletionTime(clock.Now().Add(5*time.Minute)),
			),
		),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_failed_withoutPRCondition(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(2*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(5*time.Minute)),
			),
		),
	}

	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline",
				tb.PipelineRunServiceAccountName("test-sa"),
			),
			tb.PipelineRunStatus(
				tb.PipelineRunTaskRunsStatus("tr-1", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "t-1",
					Status:           &trs[0].Status,
				}),
				tb.PipelineRunStartTime(clock.Now()),
				cb.PipelineRunCompletionTime(clock.Now().Add(5*time.Minute)),
			),
		),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_with_resources_taskrun(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(2*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(5*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
			),
		),
	}

	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline",
				tb.PipelineRunServiceAccountName("test-sa"),
				tb.PipelineRunParam("test-param", "param-value"),
				tb.PipelineRunResourceBinding("test-resource",
					tb.PipelineResourceBindingRef("test-resource-ref"),
				),
			),
			tb.PipelineRunStatus(
				tb.PipelineRunTaskRunsStatus("tr-1", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "t-1",
					Status:           &trs[0].Status,
				}),
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.PipelineRunReasonSuccessful.String(),
				}),
				tb.PipelineRunStartTime(clock.Now()),
				cb.PipelineRunCompletionTime(clock.Now().Add(5*time.Minute)),
			),
		),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_without_start_time(t *testing.T) {
	clock := clockwork.NewFakeClock()

	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(),
		),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_without_pipelineref(t *testing.T) {
	clock := clockwork.NewFakeClock()

	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunStatus(),
		),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_no_resourceref(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(2*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(5*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
			),
		),
	}

	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline",
				tb.PipelineRunServiceAccountName("test-sa"),
				tb.PipelineRunParam("test-param", "param-value"),
				tb.PipelineRunResourceBinding("test-resource"),
			),
			tb.PipelineRunStatus(
				tb.PipelineRunTaskRunsStatus("tr-1", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "t-1",
					Status:           &trs[0].Status,
				}),
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.PipelineRunReasonSuccessful.String(),
				}),
				tb.PipelineRunStartTime(clock.Now()),
				cb.PipelineRunCompletionTime(clock.Now().Add(5*time.Minute)),
			),
		),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_cancelled_pipelinerun(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(2*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(5*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
			),
		),
	}

	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunTaskRunsStatus("tr-1", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "t-1",
					Status:           &trs[0].Status,
				}),
				tb.PipelineRunStatusCondition(apis.Condition{
					Status:  corev1.ConditionFalse,
					Reason:  "PipelineRunCancelled",
					Message: "PipelineRun \"pipeline-run\" was cancelled",
				}),
				tb.PipelineRunStartTime(clock.Now()),
				cb.PipelineRunCompletionTime(clock.Now().Add(5*time.Minute)),
			),
		),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_without_tr_start_time(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionReady,
					Status: corev1.ConditionUnknown,
				}),
			),
		),
	}

	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionUnknown,
					Reason: v1beta1.PipelineRunReasonRunning.String(),
				}),
				tb.PipelineRunTaskRunsStatus("tr-1", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "t-1",
					Status:           &trs[0].Status,
				}),
				tb.PipelineRunStartTime(clock.Now()),
			),
		),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_custom_timeout(t *testing.T) {
	prun := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pr-custom-timeout",
			tb.PipelineRunNamespace("ns"),
			tb.PipelineRunSpec("pr-custom-timeout", tb.PipelineRunTimeout(time.Minute)),
		),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(prun[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: prun})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pr-custom-timeout", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunsDescribe_custom_output(t *testing.T) {
	pipelinerunname := "pipeline-run"
	expected := "pipelinerun.tekton.dev/" + pipelinerunname

	prun := []*v1alpha1.PipelineRun{
		tb.PipelineRun(pipelinerunname,
			tb.PipelineRunNamespace("ns"),
		),
	}
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(prun[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: prun})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-o", "name", "-n", "ns", pipelinerunname)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	got = strings.TrimSpace(got)
	if got != expected {
		t.Errorf("Result should be '%s' != '%s'", got, expected)
	}
}

func TestPipelineRunDescribeV1beta1_custom_output(t *testing.T) {
	pipelinerunname := "pipeline-run"
	expected := "pipelinerun.tekton.dev/" + pipelinerunname
	prun := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname,
				Namespace: "ns",
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
		cb.UnstructuredV1beta1PR(prun[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: namespaces, PipelineRuns: prun})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-o", "name", "-n", "ns", pipelinerunname)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	got = strings.TrimSpace(got)
	if got != expected {
		t.Errorf("Result should be '%s' != '%s'", got, expected)
	}
}

func TestPipelineRunDescribeV1beta1(t *testing.T) {
	clock := clockwork.NewFakeClock()
	pipelinerunname := "pipeline-run"
	taskRuns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-1",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.PipelineRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-2",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(17 * time.Minute)},
				},
			},
		},
	}

	prun := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipeline",
				},
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "res-1",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "test-res",
						},
					},
					{
						Name: "res-2",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "test-res2",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "p-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "p-2",
						Value: v1beta1.ArrayOrString{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Reason:  v1beta1.PipelineRunReasonSuccessful.String(),
							Message: "Completed",
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(20 * time.Minute)},
					TaskRuns: map[string]*v1beta1.PipelineRunTaskRunStatus{
						"tr-1": {
							PipelineTaskName: "t-1",
							Status:           &taskRuns[0].Status,
						},
						"tr-2": {
							PipelineTaskName: "t-2",
							Status:           &taskRuns[1].Status,
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
		cb.UnstructuredV1beta1PR(prun[0], version),
		cb.UnstructuredV1beta1TR(taskRuns[0], version),
		cb.UnstructuredV1beta1TR(taskRuns[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: namespaces, PipelineRuns: prun, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-n", "ns", pipelinerunname)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribeV1beta1_taskrun_with_no_status(t *testing.T) {
	clock := clockwork.NewFakeClock()
	pipelinerunname := "pipeline-run"
	taskRuns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-1",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.PipelineRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-2",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
				},
			},
		},
	}

	prun := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipeline",
				},
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "res-1",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "test-res",
						},
					},
					{
						Name: "res-2",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "test-res2",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "p-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "p-2",
						Value: v1beta1.ArrayOrString{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Reason:  v1beta1.PipelineRunReasonSuccessful.String(),
							Message: "Completed",
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(20 * time.Minute)},
					TaskRuns: map[string]*v1beta1.PipelineRunTaskRunStatus{
						"tr-1": {
							PipelineTaskName: "t-1",
							Status:           &taskRuns[0].Status,
						},
						"tr-2": {
							PipelineTaskName: "t-2",
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
		cb.UnstructuredV1beta1PR(prun[0], version),
		cb.UnstructuredV1beta1TR(taskRuns[0], version),
		cb.UnstructuredV1beta1TR(taskRuns[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: namespaces, PipelineRuns: prun, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-n", "ns", pipelinerunname)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_lastV1beta1(t *testing.T) {
	clock := clockwork.NewFakeClock()
	pipelinerunname1 := "pipeline-run"
	pipelinerunname2 := "pipeline-run2"
	taskRuns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-1",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.PipelineRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-2",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(-3 * time.Minute)},
				},
			},
		},
	}

	prun := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname1,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipeline",
				},
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "res-1",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "test-res",
						},
					},
					{
						Name: "res-2",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "test-res2",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "p-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "p-2",
						Value: v1beta1.ArrayOrString{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Reason:  v1beta1.PipelineRunReasonSuccessful.String(),
							Message: "Completed",
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					TaskRuns: map[string]*v1beta1.PipelineRunTaskRunStatus{
						"tr-1": {
							PipelineTaskName: "t-1",
							Status:           &taskRuns[0].Status,
						},
						"tr-2": {
							PipelineTaskName: "t-2",
							Status:           &taskRuns[1].Status,
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname2,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipeline2",
				},
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "res-1x",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "test-res",
						},
					},
					{
						Name: "res-2x",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "test-res2",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "p-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "p-2",
						Value: v1beta1.ArrayOrString{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Reason:  v1beta1.PipelineRunReasonSuccessful.String(),
							Message: "Completed",
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(20 * time.Minute)},
					TaskRuns: map[string]*v1beta1.PipelineRunTaskRunStatus{
						"tr-1": {
							PipelineTaskName: "t-1",
							Status:           &taskRuns[0].Status,
						},
						"tr-2": {
							PipelineTaskName: "t-2",
							Status:           &taskRuns[1].Status,
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
		cb.UnstructuredV1beta1PR(prun[0], version),
		cb.UnstructuredV1beta1PR(prun[1], version),
		cb.UnstructuredV1beta1TR(taskRuns[0], version),
		cb.UnstructuredV1beta1TR(taskRuns[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: namespaces, PipelineRuns: prun, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-n", "ns", "--last")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_last(t *testing.T) {
	clock := clockwork.NewFakeClock()

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr-1",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(2*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(5*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
			),
		),
		tb.TaskRun("tr-2",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(5*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(9*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
			),
		),
		tb.TaskRun("tr-3",
			tb.TaskRunNamespace("ns"),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(clock.Now().Add(5*time.Minute)),
				cb.TaskRunCompletionTime(clock.Now().Add(9*time.Minute)),
				tb.StatusCondition(apis.Condition{
					Type:   apis.ConditionSucceeded,
					Status: corev1.ConditionTrue,
				}),
			),
		),
	}

	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunTaskRunsStatus("tr-1", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "t-1",
					Status:           &trs[0].Status,
				}),
				tb.PipelineRunTaskRunsStatus("tr-2", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "t-2",
					Status:           &trs[1].Status,
				}),
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.PipelineRunReasonSuccessful.String(),
				}),
				tb.PipelineRunStartTime(clock.Now().Add(5*time.Minute)),
				cb.PipelineRunCompletionTime(clock.Now().Add(20*time.Minute)),
			),
		),
		tb.PipelineRun("pipeline-run2",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(clock.Now().Add(5*time.Minute)),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunTaskRunsStatus("tr-1", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "t-1",
					Status:           &trs[0].Status,
				}),
				tb.PipelineRunTaskRunsStatus("tr-3", &v1alpha1.PipelineRunTaskRunStatus{
					PipelineTaskName: "t-3",
					Status:           &trs[1].Status,
				}),
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: v1beta1.PipelineRunReasonSuccessful.String(),
				}),
				tb.PipelineRunStartTime(clock.Now()),
				cb.PipelineRunCompletionTime(clock.Now().Add(15*time.Minute)),
			),
		),
	}

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredPR(pipelineRuns[1], version),
		cb.UnstructuredTR(trs[0], version),
		cb.UnstructuredTR(trs[1], version),
		cb.UnstructuredTR(trs[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: pipelineRuns,
		TaskRuns: trs,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	pipelinerun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "--last", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))

}

func TestPipelineRunDescribe_v1beta1_with_results(t *testing.T) {
	clock := clockwork.NewFakeClock()
	pipelinerunname := "pipeline-run"
	taskRuns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-1",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.PipelineRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
			},
		},
	}

	prun := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipeline",
				},
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "res-1",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "test-res",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "p-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Reason:  v1beta1.PipelineRunReasonSuccessful.String(),
							Message: "Completed",
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					TaskRuns: map[string]*v1beta1.PipelineRunTaskRunStatus{
						"tr-1": {
							PipelineTaskName: "t-1",
							Status:           &taskRuns[0].Status,
						},
					},
					PipelineResults: []v1beta1.PipelineRunResult{
						{
							Name:  "result-1",
							Value: "value-1",
						},
						{
							Name:  "result-2",
							Value: "value-2",
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
		cb.UnstructuredV1beta1PR(prun[0], version),
		cb.UnstructuredV1beta1TR(taskRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: namespaces, PipelineRuns: prun, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-n", "ns", "--last")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_zero_timeout(t *testing.T) {
	prun := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run-zero-timeout",
			tb.PipelineRunNamespace("ns"),
			tb.PipelineRunSpec("pipeline-zero-timeout", tb.PipelineRunTimeout(0)),
		),
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
		cb.UnstructuredPR(prun[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, PipelineRuns: prun})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	actual, err := test.ExecuteCommand(pipelinerun, "desc", "pipeline-run-zero-timeout", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineRunDescribe_v1beta1_with_workspaces(t *testing.T) {
	clock := clockwork.NewFakeClock()
	pipelinerunname := "pipeline-run"
	taskRuns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "tr-1",
				Labels:    map[string]string{"tekton.dev/task": "task-1"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task-1",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.PipelineRunReasonFailed.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
			},
		},
	}

	prun := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelinerunname,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipeline",
				},
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "res-1",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "test-res",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "p-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
				},
				Workspaces: []v1beta1.WorkspaceBinding{
					{
						Name:     "test",
						SubPath:  "test",
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
					{
						Name: "configmap",
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "bar"},
						},
					},
					{
						Name: "secret",
						Secret: &corev1.SecretVolumeSource{
							SecretName: "foobar",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Reason:  v1beta1.PipelineRunReasonSuccessful.String(),
							Message: "Completed",
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					TaskRuns: map[string]*v1beta1.PipelineRunTaskRunStatus{
						"tr-1": {
							PipelineTaskName: "t-1",
							Status:           &taskRuns[0].Status,
						},
					},
					PipelineResults: []v1beta1.PipelineRunResult{
						{
							Name:  "result-1",
							Value: "value-1",
						},
						{
							Name:  "result-2",
							Value: "value-2",
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
		cb.UnstructuredV1beta1PR(prun[0], version),
		cb.UnstructuredV1beta1TR(taskRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: namespaces, PipelineRuns: prun, TaskRuns: taskRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipelinerun := Command(p)
	got, err := test.ExecuteCommand(pipelinerun, "desc", "-n", "ns", "--last")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}
