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
)

func TestPipelinesList_invalid_namespace(t *testing.T) {
	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
	}

	tdc := testDynamic.Options{}
	dc, _ := tdc.Client()
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pipeline := Command(p)
	output, err := test.ExecuteCommand(pipeline, "list", "-n", "invalid")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "No Pipelines found\n", output)
}

func TestPipelinesList_empty(t *testing.T) {
	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pipeline := Command(p)
	output, err := test.ExecuteCommand(pipeline, "list", "-n", "foo")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "No Pipelines found\n", output)
}

func TestPipelineList_only_pipelines(t *testing.T) {
	pipelines := []pipelineDetails{
		{"tomatoes", 1 * time.Minute, "namespace"},
		{"mangoes", 20 * time.Second, "namespace"},
		{"bananas", 512 * time.Hour, "namespace"}, // 3 weeks
	}

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()
	cs, pdata := seedPipelines(t, clock, pipelines, nsList)
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(pdata[0], version),
		cb.UnstructuredP(pdata[1], version),
		cb.UnstructuredP(pdata[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}

	pipeline := Command(p)
	output, err := test.ExecuteCommand(pipeline, "list", "-n", "namespace")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineList_only_pipelines_v1beta1(t *testing.T) {
	clock := clockwork.NewFakeClock()
	version := "v1beta1"

	pdata := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bananas",
				Namespace: "namespace",
				// Created 3 weeks ago
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
	}

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1P(pdata[0], version),
		cb.UnstructuredV1beta1P(pdata[1], version),
		cb.UnstructuredV1beta1P(pdata[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Pipelines: pdata, Namespaces: nsList})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline"})
	pipeline := Command(p)

	output, err := test.ExecuteCommand(pipeline, "list", "-n", "namespace")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineList_only_pipelines_no_headers_v1beta1(t *testing.T) {
	clock := clockwork.NewFakeClock()
	version := "v1beta1"

	pdata := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bananas",
				Namespace: "namespace",
				// Created 3 weeks ago
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
	}

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1P(pdata[0], version),
		cb.UnstructuredV1beta1P(pdata[1], version),
		cb.UnstructuredV1beta1P(pdata[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Pipelines: pdata, Namespaces: nsList})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline"})
	pipeline := Command(p)

	output, err := test.ExecuteCommand(pipeline, "list", "-n", "namespace", "--no-headers")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineList_only_pipelines_all_namespaces_v1beta1(t *testing.T) {
	clock := clockwork.NewFakeClock()

	version := "v1beta1"

	pdata := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bananas",
				Namespace: "namespace",
				// Created 3 weeks ago
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomates",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangues",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bananes",
				Namespace: "espace-de-nom",
				// Created 3 weeks ago
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
	}

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1P(pdata[0], version),
		cb.UnstructuredV1beta1P(pdata[1], version),
		cb.UnstructuredV1beta1P(pdata[2], version),
		cb.UnstructuredV1beta1P(pdata[3], version),
		cb.UnstructuredV1beta1P(pdata[4], version),
		cb.UnstructuredV1beta1P(pdata[5], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Pipelines: pdata, Namespaces: nil})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline"})
	pipeline := Command(p)

	output, err := test.ExecuteCommand(pipeline, "list", "--all-namespaces")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineList_only_pipelines_all_namespaces_no_headers_v1beta1(t *testing.T) {

	clock := clockwork.NewFakeClock()
	version := "v1beta1"

	pdata := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomatoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangoes",
				Namespace:         "namespace",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bananas",
				Namespace: "namespace",
				// Created 3 weeks ago
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tomates",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mangues",
				Namespace:         "espace-de-nom",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bananes",
				Namespace: "espace-de-nom",
				// Created 3 weeks ago
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
	}

	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1P(pdata[0], version),
		cb.UnstructuredV1beta1P(pdata[1], version),
		cb.UnstructuredV1beta1P(pdata[2], version),
		cb.UnstructuredV1beta1P(pdata[3], version),
		cb.UnstructuredV1beta1P(pdata[4], version),
		cb.UnstructuredV1beta1P(pdata[5], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Pipelines: pdata, Namespaces: nil})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline"})
	pipeline := Command(p)

	output, err := test.ExecuteCommand(pipeline, "list", "--all-namespaces", "--no-headers")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelinesList_with_single_run(t *testing.T) {
	clock := clockwork.NewFakeClock()
	version := "v1alpha1"
	pdata := []*v1alpha1.Pipeline{
		tb.Pipeline("pipeline",
			tb.PipelineNamespace("ns"),
			// created  5 minutes back
			cb.PipelineCreationTimestamp(clock.Now().Add(-5*time.Minute)),
		),
	}

	prdata := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pipeline-run-1",
			tb.PipelineRunNamespace("ns"),
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

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines:    pdata,
		PipelineRuns: prdata,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(pdata[0], version),
		cb.UnstructuredPR(prdata[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	pipeline := Command(p)

	// -5 : pipeline created
	//  0 : pipeline run - 1 started
	// 10 : pipeline run - 1 finished
	// 15 : <<< now run pipeline ls << - advance clock to this point

	clock.Advance(15 * time.Minute)
	got, err := test.ExecuteCommand(pipeline, "list", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelinesList_latest_run(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()
	//  Time --->
	//  |---5m ---|------------ ││--││------------- ---│--│
	//	now      pipeline       ││  │`secondRun stated │  `*first*RunCompleted
	//                          ││  `secondRun         `*second*RunCompleted
	//	                        │`firstRun started
	//	                        `firstRun
	// NOTE: firstRun completed **after** second but latest should still be
	// second run based on creationTimestamp

	var (
		pipelineCreated = clock.Now().Add(-5 * time.Minute)
		runDuration     = 5 * time.Minute

		firstRunCreated   = clock.Now().Add(10 * time.Minute)
		firstRunStarted   = firstRunCreated.Add(2 * time.Second)
		firstRunCompleted = firstRunStarted.Add(2 * runDuration) // take twice as long

		secondRunCreated   = firstRunCreated.Add(1 * time.Minute)
		secondRunStarted   = secondRunCreated.Add(2 * time.Second)
		secondRunCompleted = secondRunStarted.Add(runDuration) // takes less thus completes
	)
	pdata := []*v1alpha1.Pipeline{
		tb.Pipeline("pipeline",
			tb.PipelineNamespace("ns"),
			// created  5 minutes back
			cb.PipelineCreationTimestamp(pipelineCreated),
		),
	}

	prdata := []*v1alpha1.PipelineRun{

		tb.PipelineRun("pipeline-run-1",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(firstRunCreated),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.PipelineRunStartTime(firstRunStarted),
				cb.PipelineRunCompletionTime(firstRunCompleted),
			),
		),
		tb.PipelineRun("pipeline-run-2",
			tb.PipelineRunNamespace("ns"),
			cb.PipelineRunCreationTimestamp(secondRunCreated),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunSpec("pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.PipelineRunStartTime(secondRunStarted),
				cb.PipelineRunCompletionTime(secondRunCompleted),
			),
		),
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines:    pdata,
		PipelineRuns: prdata,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(pdata[0], version),
		cb.UnstructuredPR(prdata[0], version),
		cb.UnstructuredPR(prdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	pipeline := Command(p)

	clock.Advance(30 * time.Minute)

	got, err := test.ExecuteCommand(pipeline, "list", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

type pipelineDetails struct {
	name      string
	age       time.Duration
	namespace string
}

func seedPipelines(t *testing.T, clock clockwork.Clock, ps []pipelineDetails, nsList []*corev1.Namespace) (pipelinetest.Clients, []*v1alpha1.Pipeline) {
	pipelines := []*v1alpha1.Pipeline{}
	for _, p := range ps {
		pipelines = append(pipelines, tb.Pipeline(p.name,
			tb.PipelineNamespace(p.namespace),
			cb.PipelineCreationTimestamp(clock.Now().Add(p.age*-1))))
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Pipelines: pipelines, Namespaces: nsList})

	return cs, pipelines
}
