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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stest "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

var (
	success = apis.Condition{Type: apis.ConditionSucceeded, Status: corev1.ConditionTrue}
	failure = apis.Condition{Type: apis.ConditionSucceeded, Status: corev1.ConditionFalse}
	cancel  = apis.Condition{Status: corev1.ConditionFalse, Reason: "PipelineRunCancelled"}
)

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

	expected := "Error: namespaces \"invalid\" not found\n"
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

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace("ns"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipelineName"),
			tb.PipelineRunSpec("pipelineName",
				tb.PipelineRunServiceAccountName("test-sa"),
				tb.PipelineRunResourceBinding("git-repo", tb.PipelineResourceBindingRef("some-repo")),
				tb.PipelineRunResourceBinding("build-image", tb.PipelineResourceBindingRef("some-image")),
				tb.PipelineRunParam("pipeline-param-1", "somethingmorefun"),
				tb.PipelineRunParam("rev-param", "revision1"),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prs[0], versionA1),
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

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace("ns"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipelineName"),
			tb.PipelineRunSpec("pipelineName",
				tb.PipelineRunServiceAccountName("test-sa"),
				tb.PipelineRunResourceBinding("git-repo", tb.PipelineResourceBindingRef("some-repo")),
				tb.PipelineRunResourceBinding("build-image", tb.PipelineResourceBindingRef("some-image")),
				tb.PipelineRunParam("pipeline-param-1", "somethingmorefun"),
				tb.PipelineRunParam("rev-param", "revision1"),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{
			{
				Verb:     "patch",
				Resource: "pipelineruns",
				Action: func(action k8stest.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New(errStr)
				}}}}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prs[0], versionA1),
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

func Test_finished_pipelinerun_success(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	prName := "test-pipeline-run-123"

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace("ns"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipelineName"),
			tb.PipelineRunSpec("pipelineName",
				tb.PipelineRunServiceAccountName("test-sa"),
				tb.PipelineRunResourceBinding("git-repo", tb.PipelineResourceBindingRef("some-repo")),
				tb.PipelineRunResourceBinding("build-image", tb.PipelineResourceBindingRef("some-image")),
				tb.PipelineRunParam("pipeline-param-1", "somethingmorefun"),
				tb.PipelineRunParam("rev-param", "revision1"),
			),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(success),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prs[0], versionA1),
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

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace("ns"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipelineName"),
			tb.PipelineRunSpec("pipelineName",
				tb.PipelineRunServiceAccountName("test-sa"),
				tb.PipelineRunResourceBinding("git-repo", tb.PipelineResourceBindingRef("some-repo")),
				tb.PipelineRunResourceBinding("build-image", tb.PipelineResourceBindingRef("some-image")),
				tb.PipelineRunParam("pipeline-param-1", "somethingmorefun"),
				tb.PipelineRunParam("rev-param", "revision1"),
			),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(failure),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prs[0], versionA1),
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

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName,
			tb.PipelineRunNamespace("ns"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipelineName"),
			tb.PipelineRunSpec("pipelineName",
				tb.PipelineRunServiceAccountName("test-sa"),
				tb.PipelineRunResourceBinding("git-repo", tb.PipelineResourceBindingRef("some-repo")),
				tb.PipelineRunResourceBinding("build-image", tb.PipelineResourceBindingRef("some-image")),
				tb.PipelineRunParam("pipeline-param-1", "somethingmorefun"),
				tb.PipelineRunParam("rev-param", "revision1"),
			),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(cancel),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionA1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(prs[0], versionA1),
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
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "git-repo",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-repo",
						},
					},
					{
						Name: "build-image",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-image",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "somethingmorefun",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prs[0], versionB1),
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
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "git-repo",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-repo",
						},
					},
					{
						Name: "build-image",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-image",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{
			{
				Verb:     "patch",
				Resource: "pipelineruns",
				Action: func(action k8stest.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New(errStr)
				}}}}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prs[0], versionB1),
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
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "git-repo",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-repo",
						},
					},
					{
						Name: "build-image",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-image",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prs[0], versionB1),
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
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "git-repo",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-repo",
						},
					},
					{
						Name: "build-image",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-image",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prs[0], versionB1),
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
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "git-repo",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-repo",
						},
					},
					{
						Name: "build-image",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-image",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: "PipelineRunCancelled",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prs[0], versionB1),
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
