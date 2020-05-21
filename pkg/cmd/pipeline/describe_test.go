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

package pipeline

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestPipelineDescribe_invalid_namespace(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	pipeline := Command(p)
	_, err := test.ExecuteCommand(pipeline, "desc", "bar", "-n", "invalid")
	if err == nil {
		t.Errorf("Error expected here for invalid namespace")
	}
	expected := "namespaces \"invalid\" not found"
	test.AssertOutput(t, expected, err.Error())
}

func TestPipelineDescribe_invalid_pipeline(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}

	pipeline := Command(p)
	_, err = test.ExecuteCommand(pipeline, "desc", "bar", "-n", "ns")
	if err == nil {
		t.Errorf("Error expected here")
	}
	expected := "pipelines.tekton.dev \"bar\" not found"
	test.AssertOutput(t, expected, err.Error())
}

func TestPipelineDescribe_empty(t *testing.T) {
	clock := clockwork.NewFakeClock()

	pipelines := []*v1alpha1.Pipeline{
		tb.Pipeline("pipeline", "ns",
			// created  5 minutes back
			cb.PipelineCreationTimestamp(clock.Now().Add(-5*time.Minute)),
		),
	}
	pipelineRuns := []*v1alpha1.PipelineRun{}
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
		cb.UnstructuredP(pipelines[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, Pipelines: pipelines, PipelineRuns: pipelineRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	pipeline := Command(p)

	got, err := test.ExecuteCommand(pipeline, "desc", "-n", "ns", "pipeline")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineDescribe_with_run(t *testing.T) {
	clock := clockwork.NewFakeClock()

	pipelines := []*v1alpha1.Pipeline{
		tb.Pipeline("pipeline", "ns",
			// created  5 minutes back
			cb.PipelineCreationTimestamp(clock.Now().Add(-5*time.Minute)),
		),
	}
	pipelineRuns := []*v1alpha1.PipelineRun{

		tb.PipelineRun("pipeline-run-1", "ns",
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				// pipeline run starts now
				tb.PipelineRunStartTime(clock.Now()),
				// takes 10 minutes to complete
				cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
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
		cb.UnstructuredP(pipelines[0], version),
		cb.UnstructuredPR(pipelineRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, Pipelines: pipelines, PipelineRuns: pipelineRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	pipeline := Command(p)

	// -5 : pipeline created
	//  0 : pipeline run - 1 started
	// 10 : pipeline run - 1 finished
	// 15 : <<< now run pipeline ls << - advance clock to this point

	clock.Advance(15 * time.Minute)
	got, err := test.ExecuteCommand(pipeline, "desc", "-n", "ns", "pipeline")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineDescribe_with_spec_run(t *testing.T) {
	clock := clockwork.NewFakeClock()

	pipelines := []*v1alpha1.Pipeline{
		tb.Pipeline("pipeline", "ns",
			// created  5 minutes back
			cb.PipelineCreationTimestamp(clock.Now().Add(-5*time.Minute)),
			tb.PipelineSpec(
				tb.PipelineDescription("a test description"),
				tb.PipelineTask("task", "taskref",
					tb.RunAfter("one", "two"),
					tb.PipelineTaskTimeout(5*time.Minute),
					tb.PipelineTaskParam("task-param", "value"),
				),
			),
		),
	}
	pipelineRuns := []*v1alpha1.PipelineRun{

		tb.PipelineRun("pipeline-run-1", "ns",
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				// pipeline run starts now
				tb.PipelineRunStartTime(clock.Now()),
				// takes 10 minutes to complete
				cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
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
		cb.UnstructuredP(pipelines[0], version),
		cb.UnstructuredPR(pipelineRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, Pipelines: pipelines, PipelineRuns: pipelineRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	pipeline := Command(p)

	// -5 : pipeline created
	//  0 : pipeline run - 1 started
	// 10 : pipeline run - 1 finished
	// 15 : <<< now run pipeline ls << - advance clock to this point

	clock.Advance(15 * time.Minute)
	got, err := test.ExecuteCommand(pipeline, "desc", "-n", "ns", "pipeline")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineDescribe_with_spec_resource_param_run(t *testing.T) {
	clock := clockwork.NewFakeClock()
	pipelines := []*v1alpha1.Pipeline{
		tb.Pipeline("pipeline", "ns",
			// created  5 minutes back
			cb.PipelineCreationTimestamp(clock.Now().Add(-5*time.Minute)),
			tb.PipelineSpec(
				tb.PipelineDescription("a test description"),
				tb.PipelineTask("task", "taskref",
					tb.RunAfter("one", "two"),
					tb.PipelineTaskTimeout(5*time.Minute),
					tb.PipelineTaskParam("task-param", "value"),
				),
				tb.PipelineDeclaredResource("name", v1alpha1.PipelineResourceTypeGit),
				tb.PipelineParamSpec("pipeline-param", v1alpha1.ParamTypeString, tb.ParamSpecDescription("param of type string"), tb.ParamSpecDefault("somethingdifferent")),
			),
		),
	}
	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run-1", "ns",
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				// pipeline run starts now
				tb.PipelineRunStartTime(clock.Now()),
				// takes 10 minutes to complete
				cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
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
		cb.UnstructuredP(pipelines[0], version),
		cb.UnstructuredPR(pipelineRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, Pipelines: pipelines, PipelineRuns: pipelineRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	pipeline := Command(p)

	// -5 : pipeline created
	//  0 : pipeline run - 1 started
	// 10 : pipeline run - 1 finished
	// 15 : <<< now run pipeline ls << - advance clock to this point

	clock.Advance(15 * time.Minute)
	got, err := test.ExecuteCommand(pipeline, "desc", "-n", "ns", "pipeline")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineDescribe_with_multiple_v1alpha1_pipelineruns(t *testing.T) {
	clock := clockwork.NewFakeClock()
	pipelines := []*v1alpha1.Pipeline{
		tb.Pipeline("pipeline", "ns",
			// created  5 minutes back
			cb.PipelineCreationTimestamp(clock.Now().Add(-5*time.Minute)),
			tb.PipelineSpec(
				tb.PipelineDescription("a test description"),
				tb.PipelineTask("task", "taskref",
					tb.RunAfter("one", "two"),
					tb.PipelineTaskTimeout(5*time.Minute),
					tb.PipelineTaskParam("task-param", "value"),
				),
				tb.PipelineDeclaredResource("name", v1alpha1.PipelineResourceTypeGit),
				tb.PipelineParamSpec("pipeline-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("somethingdifferent")),
			),
		),
	}
	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run-1", "ns",
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				// pipeline run starts now
				tb.PipelineRunStartTime(clock.Now()),
				// takes 10 minutes to complete
				cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
			),
		),
		tb.PipelineRun("pipeline-run-2", "ns",
			cb.PipelineRunCreationTimestamp(clock.Now().Add(-15*time.Minute)),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				// pipeline run started 15 minutes ago
				tb.PipelineRunStartTime(clock.Now().Add(-15*time.Minute)),
				// takes 10 minutes to complete
				cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
			),
		),
		tb.PipelineRun("pipeline-run-3", "ns",
			cb.PipelineRunCreationTimestamp(clock.Now().Add(-10*time.Minute)),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				// pipeline run started 10 minutes ago
				tb.PipelineRunStartTime(clock.Now().Add(-10*time.Minute)),
				// takes 10 minutes to complete
				cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
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
		cb.UnstructuredP(pipelines[0], version),
		cb.UnstructuredPR(pipelineRuns[0], version),
		cb.UnstructuredPR(pipelineRuns[1], version),
		cb.UnstructuredPR(pipelineRuns[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, Pipelines: pipelines, PipelineRuns: pipelineRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	pipeline := Command(p)

	clock.Advance(15 * time.Minute)
	got, err := test.ExecuteCommand(pipeline, "desc", "-n", "ns", "pipeline")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineDescribe_with_spec_multiple_resource_param_run(t *testing.T) {
	clock := clockwork.NewFakeClock()
	pipelines := []*v1alpha1.Pipeline{
		tb.Pipeline("pipeline", "ns",
			// created  5 minutes back
			cb.PipelineCreationTimestamp(clock.Now().Add(-5*time.Minute)),
			tb.PipelineSpec(
				tb.PipelineDescription("a test description"),
				tb.PipelineTask("task", "taskref",
					tb.RunAfter("one", "two"),
					tb.PipelineTaskTimeout(5*time.Minute),
					tb.PipelineTaskParam("param-1", "value"),
					tb.PipelineTaskParam("param-2", "v1", "v2"),
				),
				tb.PipelineDeclaredResource("name", v1alpha1.PipelineResourceTypeGit),
				tb.PipelineDeclaredResource("code", v1alpha1.PipelineResourceTypeGit),
				tb.PipelineDeclaredResource("code-image", v1alpha1.PipelineResourceTypeImage),
				tb.PipelineDeclaredResource("artifact-image", v1alpha1.PipelineResourceTypeImage),
				tb.PipelineDeclaredResource("repo", v1alpha1.PipelineResourceTypeGit),
				tb.PipelineParamSpec("pipeline-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("somethingdifferent")),
				tb.PipelineParamSpec("rev-param", v1alpha1.ParamTypeArray, tb.ParamSpecDefault("booms", "booms", "booms")),
				tb.PipelineParamSpec("pipeline-param2", v1alpha1.ParamTypeString, tb.ParamSpecDescription("params without the default value")),
				tb.PipelineParamSpec("rev-param2", v1alpha1.ParamTypeArray),
			),
		),
	}
	pipelineRuns := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run-1", "ns",
			cb.PipelineRunCreationTimestamp(clock.Now()),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				// pipeline run starts now
				tb.PipelineRunStartTime(clock.Now()),
				// takes 10 minutes to complete
				cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
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
		cb.UnstructuredP(pipelines[0], version),
		cb.UnstructuredPR(pipelineRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, Pipelines: pipelines, PipelineRuns: pipelineRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	pipeline := Command(p)

	// -5 : pipeline created
	//  0 : pipeline run - 1 started
	// 10 : pipeline run - 1 finished
	// 15 : <<< now run pipeline ls << - advance clock to this point

	clock.Advance(15 * time.Minute)
	got, err := test.ExecuteCommand(pipeline, "desc", "-n", "ns", "pipeline")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineDescribeV1beta1_with_spec_multiple_resource_param_run(t *testing.T) {
	clock := clockwork.NewFakeClock()
	pipelines := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Description: "",
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "task",
						TaskRef: &v1beta1.TaskRef{
							Name: "taskref",
						},
						RunAfter: []string{
							"one",
							"two",
						},
						Params: []v1beta1.Param{{
							Name: "param", Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "value"},
						}},
						Timeout: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
					},
				},
				Resources: []v1beta1.PipelineDeclaredResource{
					{
						Name: "name",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
					{
						Name: "code",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
					{
						Name: "code-image",
						Type: v1alpha1.PipelineResourceTypeImage,
					},
					{
						Name: "artifact-image",
						Type: v1alpha1.PipelineResourceTypeImage,
					},
					{
						Name: "repo",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeArray,
						Default: &v1beta1.ArrayOrString{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
					{
						Name: "pipeline-param",
						Type: v1beta1.ParamTypeString,
					},
					{
						Name: "rev-param2",
						Type: v1beta1.ParamTypeArray,
					},
				},
			},
		},
	}

	pipelineRuns := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "pipeline-run-1",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now()},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(5 * time.Minute)},
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
		cb.UnstructuredV1beta1P(pipelines[0], version),
		cb.UnstructuredV1beta1PR(pipelineRuns[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: namespaces, Pipelines: pipelines, PipelineRuns: pipelineRuns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	pipeline := Command(p)

	// -5 : pipeline created
	//  0 : pipeline run - 1 started
	// 10 : pipeline run - 1 finished
	// 15 : <<< now run pipeline ls << - advance clock to this point

	clock.Advance(15 * time.Minute)
	got, err := test.ExecuteCommand(pipeline, "desc", "-n", "ns", "pipeline")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineDescribe_custom_output(t *testing.T) {
	clock := clockwork.NewFakeClock()
	pipelines := []*v1alpha1.Pipeline{
		tb.Pipeline("pipeline", "ns",
			tb.PipelineSpec(
				tb.PipelineDeclaredResource("name", v1alpha1.PipelineResourceTypeGit),
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
		cb.UnstructuredP(pipelines[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, Pipelines: pipelines})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	pipeline := Command(p)

	got, err := test.ExecuteCommand(pipeline, "desc", "-o", "name", "-n", "ns", "pipeline")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	got = strings.TrimSpace(got)
	if got != "pipeline.tekton.dev/pipeline" {
		t.Errorf("Result should be 'pipeline.tekton.dev/pipeline' != '%s'", got)
	}
}

func TestPipelineDescribeV1beta1_custom_output(t *testing.T) {
	clock := clockwork.NewFakeClock()
	pipelines := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Resources: []v1beta1.PipelineDeclaredResource{
					{
						Name: "name",
						Type: v1alpha1.PipelineResourceTypeGit,
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
		cb.UnstructuredV1beta1P(pipelines[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: namespaces, Pipelines: pipelines})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	pipeline := Command(p)

	got, err := test.ExecuteCommand(pipeline, "desc", "-o", "name", "-n", "ns", "pipeline")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	got = strings.TrimSpace(got)
	if got != "pipeline.tekton.dev/pipeline" {
		t.Errorf("Result should be 'pipeline.tekton.dev/pipeline' != '%s'", got)
	}
}

func TestPipelineDescribeV1beta1_task_conditions(t *testing.T) {
	clock := clockwork.NewFakeClock()
	pipelines := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Resources: []v1beta1.PipelineDeclaredResource{
					{
						Name: "name",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
				},
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "task-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "task-1",
						},
						Conditions: []v1beta1.PipelineTaskCondition{
							{ConditionRef: "condition-1"},
							{ConditionRef: "condition-2"},
							{ConditionRef: "condition-3"},
						},
					},
					{
						Name: "task-2",
						TaskRef: &v1beta1.TaskRef{
							Name: "task-2",
						},
						Conditions: []v1beta1.PipelineTaskCondition{
							{ConditionRef: "condition-1"},
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
		cb.UnstructuredV1beta1P(pipelines[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: namespaces, Pipelines: pipelines})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	pipeline := Command(p)

	got, err := test.ExecuteCommand(pipeline, "desc", "-n", "ns", "pipeline")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}
