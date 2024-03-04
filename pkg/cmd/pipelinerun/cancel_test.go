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
	"errors"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	tu "github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stest "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func Test_cancel_invalid_namespace_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns})
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "invalid")

	expected := "Error: failed to find PipelineRun: test-pipeline-run-123\n"
	tu.AssertOutput(t, expected, got)
}

func Test_cancel_invalid_namespace(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "invalid")

	expected := "Error: failed to find PipelineRun: test-pipeline-run-123\n"
	tu.AssertOutput(t, expected, got)
}

func Test_cancel_pipelinerun_not_found_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns})
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: failed to find PipelineRun: " + prName + "\n"
	tu.AssertOutput(t, expected, got)
}

func Test_cancel_pipelinerun_not_found(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: failed to find PipelineRun: " + prName + "\n"
	tu.AssertOutput(t, expected, got)
}

func Test_cancel_pipelinerun_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipelineName",
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "somethingmorefun",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Type:   apis.ConditionReady,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "PipelineRun cancelled: " + prName + "\n"
	tu.AssertOutput(t, expected, got)
}

func Test_cancel_pipelinerun_client_err_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"
	errStr := "test generated error"

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipelineName",
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{
			{
				Verb:     "patch",
				Resource: "pipelineruns",
				Action: func(_ k8stest.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New(errStr)
				}}}}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: failed to cancel PipelineRun: " + prName + ": " + errStr + "\n"
	tu.AssertOutput(t, expected, got)
}

func Test_cancel_pipelinerun_client_err(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}
	prName := "test-pipeline-run-123"
	errStr := "test generated error"

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipelineName",
				},
				Params: []v1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{
			{
				Verb:     "patch",
				Resource: "pipelineruns",
				Action: func(_ k8stest.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New(errStr)
				}}}}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: failed to cancel PipelineRun: " + prName + ": " + errStr + "\n"
	tu.AssertOutput(t, expected, got)
}

func Test_finished_pipelinerun_success_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipelineName",
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: failed to cancel PipelineRun " + prName + ": PipelineRun has already finished execution\n"
	tu.AssertOutput(t, expected, got)
}

func Test_finished_pipelinerun_success(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipelineName",
				},
				Params: []v1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: failed to cancel PipelineRun " + prName + ": PipelineRun has already finished execution\n"
	tu.AssertOutput(t, expected, got)
}

func Test_finished_pipelinerun_failure_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipelineName",
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: failed to cancel PipelineRun " + prName + ": PipelineRun has already finished execution\n"
	tu.AssertOutput(t, expected, got)
}

func Test_finished_pipelinerun_failure(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}
	prName := "test-pipeline-run-123"

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipelineName",
				},
				Params: []v1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: failed to cancel PipelineRun " + prName + ": PipelineRun has already finished execution\n"
	tu.AssertOutput(t, expected, got)
}

func Test_finished_pipelinerun_cancel_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipelineName",
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: "PipelineRunCancelled",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: failed to cancel PipelineRun " + prName + ": PipelineRun has already finished execution\n"
	tu.AssertOutput(t, expected, got)
}

func Test_finished_pipelinerun_cancel(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipelineName",
				},
				Params: []v1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: "PipelineRunCancelled",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: failed to cancel PipelineRun " + prName + ": PipelineRun has already finished execution\n"
	tu.AssertOutput(t, expected, got)
}

func Test_finished_pipelinerun_timeout_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipelineName",
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.PipelineRunReasonTimedOut.String(),
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: failed to cancel PipelineRun " + prName + ": PipelineRun has already finished execution\n"
	tu.AssertOutput(t, expected, got)
}

func Test_finished_pipelinerun_timeout(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipelineName",
				},
				Params: []v1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.PipelineRunReasonTimedOut.String(),
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: failed to cancel PipelineRun " + prName + ": PipelineRun has already finished execution\n"
	tu.AssertOutput(t, expected, got)
}

func Test_finished_pipelinerun_cancelled_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipelineName",
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: "Cancelled",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: failed to cancel PipelineRun " + prName + ": PipelineRun has already finished execution\n"
	tu.AssertOutput(t, expected, got)
}

func Test_finished_pipelinerun_cancelled(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}
	prName := "test-pipeline-run-123"

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipelineName",
				},
				Params: []v1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: "Cancelled",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: failed to cancel PipelineRun " + prName + ": PipelineRun has already finished execution\n"
	tu.AssertOutput(t, expected, got)
}

func Test_cancel_pipelinerun_with_grace_stopped_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipelineName",
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "somethingmorefun",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Type:   apis.ConditionReady,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns", "--grace", "StoppedRunFinally")

	expected := "PipelineRun cancelled: " + prName + "\n"
	tu.AssertOutput(t, expected, got)
}

func Test_cancel_pipelinerun_with_grace_stopped1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}
	prName := "test-pipeline-run-123"

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipelineName",
				},
				Params: []v1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "somethingmorefun",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Type:   apis.ConditionReady,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns", "--grace", "StoppedRunFinally")

	expected := "PipelineRun cancelled: " + prName + "\n"
	tu.AssertOutput(t, expected, got)
}

func Test_cancel_pipelinerun_with_grace_cancelled_v1bea1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipelineName",
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "somethingmorefun",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Type:   apis.ConditionReady,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns", "--grace", "CancelledRunFinally")

	expected := "PipelineRun cancelled: " + prName + "\n"
	tu.AssertOutput(t, expected, got)
}

func Test_cancel_pipelinerun_with_grace_cancelled(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipelineName",
				},
				Params: []v1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "somethingmorefun",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Type:   apis.ConditionReady,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns", "--grace", "CancelledRunFinally")

	expected := "PipelineRun cancelled: " + prName + "\n"
	tu.AssertOutput(t, expected, got)
}

func Test_cancel_pipelinerun(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipelineName"},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipelineName",
				},
				Params: []v1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "somethingmorefun",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Type:   apis.ConditionReady,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "PipelineRun cancelled: " + prName + "\n"
	tu.AssertOutput(t, expected, got)
}
