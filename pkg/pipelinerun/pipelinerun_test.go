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
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"github.com/tektoncd/pipeline/test/diff"
	"github.com/tektoncd/pipeline/test/parse"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestPipelineRunsList_with_single_run(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	pr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute
	prdata := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipelinerun",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "random"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "random",
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: pr1Started},
					CompletionTime: &metav1.Time{Time: pr1Started.Add(runDuration)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipelinerun1",
				Namespace: "ns",
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
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: pr1Started},
					CompletionTime: &metav1.Time{Time: pr1Started.Add(runDuration)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipelinerun2",
				Namespace: "ns",
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
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: pr1Started},
					CompletionTime: &metav1.Time{Time: pr1Started.Add(runDuration)},
				},
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{
		PipelineRuns: prdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prdata[0], version),
		cb.UnstructuredV1beta1PR(prdata[1], version),
		cb.UnstructuredV1beta1PR(prdata[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2.SetNamespace("unknown")

	testParams := []struct {
		name        string
		params      *test.Params
		listOptions metav1.ListOptions
		want        []string
	}{
		{
			name:        "Specify related pipeline",
			params:      p,
			listOptions: metav1.ListOptions{LabelSelector: "tekton.dev/pipeline=pipeline"},
			want: []string{
				"pipelinerun1 started -10 seconds ago",
				"pipelinerun2 started -10 seconds ago",
			},
		},
		{
			name:        "Not specify related pipeline",
			params:      p,
			listOptions: metav1.ListOptions{},
			want: []string{
				"pipelinerun started -10 seconds ago",
				"pipelinerun1 started -10 seconds ago",
				"pipelinerun2 started -10 seconds ago",
			},
		},
		{
			name:        "Specify unknown namespace",
			params:      p2,
			listOptions: metav1.ListOptions{},
			want:        []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllPipelineRuns(tp.params, tp.listOptions, 5)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestPipelineRunGet(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	pr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute

	prdata := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipelinerun1",
				Namespace: "ns",
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
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: pr1Started},
					CompletionTime: &metav1.Time{Time: pr1Started.Add(runDuration)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipelinerun2",
				Namespace: "ns",
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
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: pr1Started},
					CompletionTime: &metav1.Time{Time: pr1Started.Add(runDuration)},
				},
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{
		PipelineRuns: prdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prdata[0], version),
		cb.UnstructuredV1beta1PR(prdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Get(c, "pipelinerun1", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "pipelinerun1", got.Name)
}

func TestPipelineRunGet_MinimalEmbeddedStatus(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	pr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute

	prdata := []*v1beta1.PipelineRun{
		parse.MustParseV1beta1PipelineRun(t, fmt.Sprintf(`
metadata:
  name: pipelinerun1
  namespace: ns
  labels:
    tekton.dev/pipeline: pipeline
spec:
  pipelineRef:
    name: pipeline
status:
  conditions:
  - lastTransitionTime: null
    message: All Tasks have completed executing
    reason: Succeeded
    status: "True"
    type: Succeeded
  startTime: %s
  completionTime: %s
  childReferences:
  - apiVersion: tekton.dev/v1beta1
    kind: TaskRun
    name: task-run-1
    pipelineTaskName: tr1
  - apiVersion: tekton.dev/v1beta1
    kind: TaskRun
    name: task-run-2
    pipelineTaskName: tr2
  - apiVersion: tekton.dev/v1alpha1
    kind: Run
    name: run-1
    pipelineTaskName: r1
  - apiVersion: tekton.dev/v1alpha1
    kind: Run
    name: run-2
    pipelineTaskName: r2
`, pr1Started.Format(time.RFC3339), pr1Started.Add(runDuration).Format(time.RFC3339))),
		parse.MustParseV1beta1PipelineRun(t, fmt.Sprintf(`
metadata:
  name: pipelinerun2
  namespace: ns
  labels:
    tekton.dev/pipeline: pipeline
spec:
  pipelineRef:
    name: pipeline
status:
  conditions:
  - lastTransitionTime: null
    message: All Tasks have completed executing
    reason: Succeeded
    status: "True"
    type: Succeeded
  startTime: %s
  completionTime: %s
  taskRuns:
    task-run-1:
      pipelineTaskName: tr1
      status:
        conditions:
        - reason: Succeeded
          status: "True"
          type: Succeeded
    task-run-2:
      pipelineTaskName: tr2
      status:
        conditions:
        - reason: Failed
          status: "False"
          type: Succeeded
  runs:
    run-1:
      pipelineTaskName: r1
      status:
        conditions:
        - reason: Succeeded
          status: "True"
          type: Succeeded
    run-2:
      pipelineTaskName: r2
      status:
        conditions:
        - reason: Failed
          status: "False"
          type: Succeeded
`, pr1Started.Format(time.RFC3339), pr1Started.Add(runDuration).Format(time.RFC3339))),
	}

	trData := []*v1beta1.TaskRun{
		parse.MustParseV1beta1TaskRun(t, `
metadata:
  name: task-run-1
  namespace: ns
spec:
  taskRef:
    name: someTask
status:
  conditions:
  - reason: Succeeded
    status: "True"
    type: Succeeded
`),
		parse.MustParseV1beta1TaskRun(t, `
metadata:
  name: task-run-2
  namespace: ns
spec:
  taskRef:
    name: someTask
status:
  conditions:
  - reason: Failed
    status: "False"
    type: Succeeded
`),
	}

	runsData := []*v1alpha1.Run{
		parse.MustParseRun(t, `
metadata:
  name: run-1
  namespace: ns
spec:
  ref:
    name: someCustomTask
status:
  conditions:
  - reason: Succeeded
    status: "True"
    type: Succeeded
`),
		parse.MustParseRun(t, `
metadata:
  name: run-2
  namespace: ns
spec:
  ref:
    name: someCustomTask
status:
  conditions:
  - reason: Failed
    status: "False"
    type: Succeeded
`),
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{
		PipelineRuns: prdata,
		TaskRuns:     trData,
		Runs:         runsData,
	})

	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prdata[0], version),
		cb.UnstructuredV1beta1PR(prdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Get(c, "pipelinerun1", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "pipelinerun1", got.Name)

	tr1 := got.Status.TaskRuns[trData[0].Name]
	if tr1 == nil {
		t.Fatalf("TaskRun status map does not contain expected TaskRun %s", trData[0].Name)
	}
	test.AssertOutput(t, string(v1beta1.TaskRunReasonSuccessful), tr1.Status.GetCondition(apis.ConditionSucceeded).Reason)
	tr2 := got.Status.TaskRuns[trData[1].Name]
	if tr2 == nil {
		t.Fatalf("TaskRun status map does not contain expected TaskRun %s", trData[1].Name)
	}
	test.AssertOutput(t, string(v1beta1.TaskRunReasonFailed), tr2.Status.GetCondition(apis.ConditionSucceeded).Reason)

	r1 := got.Status.Runs[runsData[0].Name]
	if r1 == nil {
		t.Fatalf("Run status map does not contain expected Run %s", runsData[0].Name)
	}
	test.AssertOutput(t, "Succeeded", r1.Status.GetCondition(apis.ConditionSucceeded).Reason)
	r2 := got.Status.Runs[runsData[1].Name]
	if r2 == nil {
		t.Fatalf("Run status map does not contain expected Run %s", runsData[1].Name)
	}
	test.AssertOutput(t, "Failed", r2.Status.GetCondition(apis.ConditionSucceeded).Reason)

	gotFull, err := Get(c, "pipelinerun2", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}

	if d := cmp.Diff(got.Status.TaskRuns, gotFull.Status.TaskRuns); d != "" {
		t.Errorf("mismatch between minimal and full TaskRun statuses: %s", diff.PrintWantGot(d))
	}
	if d := cmp.Diff(got.Status.Runs, gotFull.Status.Runs); d != "" {
		t.Errorf("mismatch between minimal and full Run statuses: %s", diff.PrintWantGot(d))
	}
}

func TestPipelineRunList_MinimalEmbeddedStatus(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	pr1Started := clock.Now().Add(10 * time.Second)
	runDuration := 1 * time.Minute

	prdata := []*v1beta1.PipelineRun{parse.MustParseV1beta1PipelineRun(t, fmt.Sprintf(`
metadata:
  name: pipelinerun1
  namespace: ns
  labels:
    tekton.dev/pipeline: pipeline
spec:
  pipelineRef:
    name: pipeline
status:
  conditions:
  - lastTransitionTime: null
    message: All Tasks have completed executing
    reason: Succeeded
    status: "True"
    type: Succeeded
  startTime: %s
  completionTime: %s
  childReferences:
  - apiVersion: tekton.dev/v1beta1
    kind: TaskRun
    name: task-run-1
    pipelineTaskName: tr1
  - apiVersion: tekton.dev/v1beta1
    kind: TaskRun
    name: task-run-2
    pipelineTaskName: tr2
  - apiVersion: tekton.dev/v1alpha1
    kind: Run
    name: run-1
    pipelineTaskName: r1
  - apiVersion: tekton.dev/v1alpha1
    kind: Run
    name: run-2
    pipelineTaskName: r2
`, pr1Started.Format(time.RFC3339), pr1Started.Add(runDuration).Format(time.RFC3339))),
	}

	trData := []*v1beta1.TaskRun{
		parse.MustParseV1beta1TaskRun(t, `
metadata:
  name: task-run-1
  namespace: ns
spec:
  taskRef:
    name: someTask
status:
  conditions:
  - reason: Succeeded
    status: "True"
    type: Succeeded
`),
		parse.MustParseV1beta1TaskRun(t, `
metadata:
  name: task-run-2
  namespace: ns
spec:
  taskRef:
    name: someTask
status:
  conditions:
  - reason: Failed
    status: "False"
    type: Succeeded
`),
	}

	runsData := []*v1alpha1.Run{
		parse.MustParseRun(t, `
metadata:
  name: run-1
  namespace: ns
spec:
  ref:
    name: someCustomTask
status:
  conditions:
  - reason: Succeeded
    status: "True"
    type: Succeeded
`),
		parse.MustParseRun(t, `
metadata:
  name: run-2
  namespace: ns
spec:
  ref:
    name: someCustomTask
status:
  conditions:
  - reason: Failed
    status: "False"
    type: Succeeded
`),
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{
		PipelineRuns: prdata,
		TaskRuns:     trData,
		Runs:         runsData,
	})

	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(prdata[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	prList, err := List(c, metav1.ListOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, 1, len(prList.Items))

	got := prList.Items[0]
	test.AssertOutput(t, "pipelinerun1", got.Name)

	tr1 := got.Status.TaskRuns[trData[0].Name]
	if tr1 == nil {
		t.Fatalf("TaskRun status map does not contain expected TaskRun %s", trData[0].Name)
	}
	test.AssertOutput(t, string(v1beta1.TaskRunReasonSuccessful), tr1.Status.GetCondition(apis.ConditionSucceeded).Reason)
	tr2 := got.Status.TaskRuns[trData[1].Name]
	if tr2 == nil {
		t.Fatalf("TaskRun status map does not contain expected TaskRun %s", trData[1].Name)
	}
	test.AssertOutput(t, string(v1beta1.TaskRunReasonFailed), tr2.Status.GetCondition(apis.ConditionSucceeded).Reason)

	r1 := got.Status.Runs[runsData[0].Name]
	if r1 == nil {
		t.Fatalf("Run status map does not contain expected Run %s", runsData[0].Name)
	}
	test.AssertOutput(t, "Succeeded", r1.Status.GetCondition(apis.ConditionSucceeded).Reason)
	r2 := got.Status.Runs[runsData[1].Name]
	if r2 == nil {
		t.Fatalf("Run status map does not contain expected Run %s", runsData[1].Name)
	}
	test.AssertOutput(t, "Failed", r2.Status.GetCondition(apis.ConditionSucceeded).Reason)
}

func TestPipelineRunCreate(t *testing.T) {
	version := "v1beta1"
	prdata := v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pipelinerun1",
			Namespace: "ns",
			Labels:    map[string]string{"tekton.dev/pipeline": "pipeline"},
		},
		Spec: v1beta1.PipelineRunSpec{
			PipelineRef: &v1beta1.PipelineRef{
				Name: "pipeline",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Create(c, &prdata, metav1.CreateOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "pipelinerun1", got.Name)
}
