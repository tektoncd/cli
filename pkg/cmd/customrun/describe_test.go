// Copyright © 2026 The Tekton Authors.
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

package customrun

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestCustomRunDescribe_not_found(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"customrun"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	customrun := Command(p)
	out, err := test.ExecuteCommand(customrun, "desc", "bar", "-n", "ns")
	if err == nil {
		t.Errorf("Expected error but did not get one")
	}
	expected := "Error: failed to get CustomRun bar: customruns.tekton.dev \"bar\" not found\n"
	test.AssertOutput(t, expected, out)
}

func TestCustomRunDescribe_empty_customrun(t *testing.T) {
	clock := test.FakeClock()

	crs := []*v1beta1.CustomRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cr-1",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/customTask": "mytask"},
			},
			Spec: v1beta1.CustomRunSpec{
				CustomRef: &v1beta1.TaskRef{
					APIVersion: "example.dev/v0",
					Kind:       "MyCustomTask",
					Name:       "mytask",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
			},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CustomRun(crs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"customrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	customrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(customrun, "desc", "cr-1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestCustomRunDescribe_with_params(t *testing.T) {
	clock := test.FakeClock()

	crs := []*v1beta1.CustomRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cr-1",
				Namespace: "ns",
			},
			Spec: v1beta1.CustomRunSpec{
				CustomRef: &v1beta1.TaskRef{
					APIVersion: "example.dev/v0",
					Kind:       "MyCustomTask",
					Name:       "mytask",
				},
				Params: v1beta1.Params{
					{
						Name:  "param1",
						Value: v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "value1"},
					},
					{
						Name:  "param2",
						Value: v1beta1.ParamValue{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"a", "b", "c"}},
					},
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
			},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
				CustomRunStatusFields: v1beta1.CustomRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CustomRun(crs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"customrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	customrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(customrun, "desc", "cr-1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestCustomRunDescribe_failed(t *testing.T) {
	clock := test.FakeClock()

	crs := []*v1beta1.CustomRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cr-1",
				Namespace: "ns",
			},
			Spec: v1beta1.CustomRunSpec{
				CustomRef: &v1beta1.TaskRef{
					APIVersion: "example.dev/v0",
					Kind:       "MyCustomTask",
					Name:       "mytask",
				},
				Timeout: &metav1.Duration{Duration: 1 * time.Hour},
			},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  v1beta1.CustomRunReasonFailed.String(),
							Message: "CustomRun failed because of some error",
						},
					},
				},
				CustomRunStatusFields: v1beta1.CustomRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(2 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CustomRun(crs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"customrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	customrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(customrun, "desc", "cr-1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestCustomRunDescribe_last_no_customrun_present(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"customrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	customrun := Command(p)
	out, err := test.ExecuteCommand(customrun, "desc", "--last", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := "No CustomRuns present in namespace ns\n"
	test.AssertOutput(t, expected, out)
}

func TestCustomRunDescribe_custom_output(t *testing.T) {
	name := "custom-run"
	expected := "customrun.tekton.dev/" + name

	clock := test.FakeClock()

	crs := []*v1beta1.CustomRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "ns",
			},
			Spec: v1beta1.CustomRunSpec{
				CustomRef: &v1beta1.TaskRef{
					APIVersion: "example.dev/v0",
					Kind:       "MyCustomTask",
					Name:       "mytask",
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CustomRun(crs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"customrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	customrun := Command(p)
	got, err := test.ExecuteCommand(customrun, "desc", "-o", "name", "-n", "ns", name)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	got = strings.TrimSpace(got)
	if got != expected {
		t.Errorf("Result should be '%s' != '%s'", got, expected)
	}
}

func TestCustomRunDescribe_with_labels_and_annotations(t *testing.T) {
	clock := test.FakeClock()

	crs := []*v1beta1.CustomRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cr-with-labels",
				Namespace: "ns",
				Labels: map[string]string{
					"tekton.dev/customTask": "mytask",
					"app":                   "myapp",
				},
				Annotations: map[string]string{
					corev1.LastAppliedConfigAnnotation: "LastAppliedConfig",
					"tekton.dev/tags":                  "testing",
				},
			},
			Spec: v1beta1.CustomRunSpec{
				CustomRef: &v1beta1.TaskRef{
					APIVersion: "example.dev/v0",
					Kind:       "MyCustomTask",
					Name:       "mytask",
				},
			},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CustomRun(crs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"customrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	customrun := Command(p)
	actual, err := test.ExecuteCommand(customrun, "desc", "cr-with-labels", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestCustomRunDescribe_with_results(t *testing.T) {
	clock := test.FakeClock()

	crs := []*v1beta1.CustomRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cr-with-results",
				Namespace: "ns",
			},
			Spec: v1beta1.CustomRunSpec{
				CustomRef: &v1beta1.TaskRef{
					APIVersion: "example.dev/v0",
					Kind:       "MyCustomTask",
					Name:       "mytask",
				},
			},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
				CustomRunStatusFields: v1beta1.CustomRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
					Results: []v1beta1.CustomRunResult{
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

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CustomRun(crs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"customrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	customrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(customrun, "desc", "cr-with-results", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestCustomRunDescribe_cancelled(t *testing.T) {
	clock := test.FakeClock()

	crs := []*v1beta1.CustomRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cr-cancelled",
				Namespace: "ns",
			},
			Spec: v1beta1.CustomRunSpec{
				CustomRef: &v1beta1.TaskRef{
					APIVersion: "example.dev/v0",
					Kind:       "MyCustomTask",
					Name:       "mytask",
				},
				Status: v1beta1.CustomRunSpecStatusCancelled,
			},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Reason:  v1beta1.CustomRunReasonCancelled.String(),
							Message: "CustomRun \"cr-cancelled\" was cancelled",
						},
					},
				},
				CustomRunStatusFields: v1beta1.CustomRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(2 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CustomRun(crs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"customrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}

	customrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(customrun, "desc", "cr-cancelled", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}

func TestCustomRunDescribe_only_one_customrun_present(t *testing.T) {
	clock := test.FakeClock()

	crs := []*v1beta1.CustomRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cr-1",
				Namespace: "ns",
			},
			Spec: v1beta1.CustomRunSpec{
				CustomRef: &v1beta1.TaskRef{
					APIVersion: "example.dev/v0",
					Kind:       "MyCustomTask",
					Name:       "mytask",
				},
			},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
				CustomRunStatusFields: v1beta1.CustomRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CustomRun(crs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"customrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	p.SetNamespace("ns")

	customrun := Command(p)
	clock.Advance(10 * time.Minute)
	actual, err := test.ExecuteCommand(customrun, "desc")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, actual, fmt.Sprintf("%s.golden", t.Name()))
}
