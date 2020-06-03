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
	"github.com/spf13/cobra"
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
	"k8s.io/client-go/dynamic"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestListPipelineRuns(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()
	runDuration := 1 * time.Minute

	pr1Started := clock.Now().Add(10 * time.Second)
	pr2Started := clock.Now().Add(-2 * time.Hour)
	pr3Started := clock.Now().Add(-1 * time.Hour)

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pr0-1",
			tb.PipelineRunNamespace("namespace"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "random"),
			tb.PipelineRunStatus(),
		),
		tb.PipelineRun("pr1-1",
			tb.PipelineRunNamespace("namespace"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipeline"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.PipelineRunStartTime(pr1Started),
				cb.PipelineRunCompletionTime(pr1Started.Add(runDuration)),
			),
		),
		tb.PipelineRun("pr2-1",
			tb.PipelineRunNamespace("namespace"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "random"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonRunning,
				}),
				tb.PipelineRunStartTime(pr2Started),
			),
		),
		tb.PipelineRun("pr2-2",
			tb.PipelineRunNamespace("namespace"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "random"),
			tb.PipelineRunLabel("viva", "galapagos"),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionFalse,
					Reason: resources.ReasonFailed,
				}),
				tb.PipelineRunStartTime(pr3Started),
				cb.PipelineRunCompletionTime(pr3Started.Add(runDuration)),
			),
		),
		tb.PipelineRun("pr3-1",
			tb.PipelineRunNamespace("namespace"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "random"),
			tb.PipelineRunLabel("viva", "wakanda"),
			tb.PipelineRunStatus(),
		),
	}

	prsMultipleNs := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pr4-1",
			tb.PipelineRunNamespace("namespace-tout"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "random"),
			tb.PipelineRunStatus(),
		),
		tb.PipelineRun("pr4-2",
			tb.PipelineRunNamespace("namespace-lacher"),
			tb.PipelineRunLabel("tekton.dev/pipeline", "random"),
			tb.PipelineRunStatus(),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	tdc1 := testDynamic.Options{}
	dc1, err := tdc1.Client(
		cb.UnstructuredPR(prs[0], version),
		cb.UnstructuredPR(prs[1], version),
		cb.UnstructuredPR(prs[2], version),
		cb.UnstructuredPR(prs[3], version),
		cb.UnstructuredPR(prs[4], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredPR(prsMultipleNs[0], version),
		cb.UnstructuredPR(prsMultipleNs[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	tests := []struct {
		name      string
		command   *cobra.Command
		args      []string
		wantError bool
	}{
		{
			name:      "Invalid namespace",
			command:   commandV1alpha1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "invalid"},
			wantError: true,
		},
		{
			name:      "by pipeline name",
			command:   commandV1alpha1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "pipeline", "-n", "namespace"},
			wantError: false,
		},
		{
			name:      "by output as name",
			command:   commandV1alpha1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "-o", "name"},
			wantError: false,
		},
		{
			name:      "all in namespace",
			command:   commandV1alpha1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace"},
			wantError: false,
		},
		{
			name:      "by template",
			command:   commandV1alpha1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
		{
			name:      "limit pipelineruns returned to 1",
			command:   commandV1alpha1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "--limit", fmt.Sprintf("%d", 1)},
			wantError: false,
		},
		{
			name:      "limit pipelineruns negative case",
			command:   commandV1alpha1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "--limit", fmt.Sprintf("%d", -1)},
			wantError: false,
		},
		{
			name:      "filter pipelineruns by label with in query",
			command:   commandV1alpha1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "--label", "viva in (wakanda,galapagos)"},
			wantError: false,
		},
		{
			name:      "filter pipelineruns by label",
			command:   commandV1alpha1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "--label", "viva=wakanda"},
			wantError: false,
		},
		{
			name:      "no mixing pipelinename and label",
			command:   commandV1alpha1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "--label", "viva=wakanda", "pr3-1"},
			wantError: true,
		},

		{
			name:      "limit pipelineruns greater than maximum case",
			command:   commandV1alpha1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "--limit", fmt.Sprintf("%d", 7)},
			wantError: false,
		},
		{
			name:      "limit pipelineruns with output flag set",
			command:   commandV1alpha1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}", "--limit", fmt.Sprintf("%d", 2)},
			wantError: false,
		},
		{
			name:      "print in reverse",
			command:   commandV1alpha1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "--reverse", "-n", "namespace"},
			wantError: false,
		},
		{
			name:      "print in reverse with output flag",
			command:   commandV1alpha1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "--reverse", "-n", "namespace", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
		{
			name:      "print pipelineruns in all namespaces",
			command:   commandV1alpha1(t, prsMultipleNs, clock.Now(), ns, version, dc2),
			args:      []string{"list", "--all-namespaces"},
			wantError: false,
		},
	}

	for _, td := range tests {
		t.Run(td.name, func(t *testing.T) {
			got, err := test.ExecuteCommand(td.command, td.args...)

			if !td.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			golden.Assert(t, got, strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
		})
	}
}

func TestListPipelineRuns_v1beta1(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	runDuration := 1 * time.Minute

	pr1Started := clock.Now().Add(10 * time.Second)
	pr2Started := clock.Now().Add(-2 * time.Hour)
	pr3Started := clock.Now().Add(-1 * time.Hour)

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
				Name:      "pr0-1",
				Labels:    map[string]string{"tekton.dev/pipeline": "random"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
				Name:      "pr1-1",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline"},
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
					StartTime:      &metav1.Time{Time: pr1Started},
					CompletionTime: &metav1.Time{Time: pr1Started.Add(runDuration)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
				Name:      "pr2-1",
				Labels:    map[string]string{"tekton.dev/pipeline": "random"},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonRunning,
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime: &metav1.Time{Time: pr2Started},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
				Name:      "pr2-2",
				Labels:    map[string]string{"tekton.dev/pipeline": "random", "viva": "galapagos"},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: resources.ReasonFailed,
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: pr3Started},
					CompletionTime: &metav1.Time{Time: pr3Started.Add(runDuration)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
				Name:      "pr3-1",
				Labels:    map[string]string{"tekton.dev/pipeline": "random", "viva": "wakanda"},
			},
		},
	}

	prsMultipleNs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace-tout",
				Name:      "pr4-1",
				Labels:    map[string]string{"tekton.dev/pipeline": "random"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace-lacher",
				Name:      "pr4-2",
				Labels:    map[string]string{"tekton.dev/pipeline": "random"},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	tdc1 := testDynamic.Options{}
	dc1, err := tdc1.Client(
		cb.UnstructuredV1beta1PR(prs[0], version),
		cb.UnstructuredV1beta1PR(prs[1], version),
		cb.UnstructuredV1beta1PR(prs[2], version),
		cb.UnstructuredV1beta1PR(prs[3], version),
		cb.UnstructuredV1beta1PR(prs[4], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	tests := []struct {
		name      string
		command   *cobra.Command
		args      []string
		wantError bool
	}{
		{
			name:      "Invalid namespace",
			command:   commandV1beta1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "invalid"},
			wantError: true,
		},
		{
			name:      "by pipeline name",
			command:   commandV1beta1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "pipeline", "-n", "namespace"},
			wantError: false,
		},
		{
			name:      "by output as name",
			command:   commandV1beta1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "-o", "name"},
			wantError: false,
		},
		{
			name:      "all in namespace",
			command:   commandV1beta1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace"},
			wantError: false,
		},
		{
			name:      "by template",
			command:   commandV1beta1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
		{
			name:      "limit pipelineruns returned to 1",
			command:   commandV1beta1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "--limit", fmt.Sprintf("%d", 1)},
			wantError: false,
		},
		{
			name:      "limit pipelineruns negative case",
			command:   commandV1beta1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "--limit", fmt.Sprintf("%d", -1)},
			wantError: false,
		},
		{
			name:      "filter pipelineruns by label with in query",
			command:   commandV1beta1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "--label", "viva in (wakanda,galapagos)"},
			wantError: false,
		},
		{
			name:      "filter pipelineruns by label",
			command:   commandV1beta1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "--label", "viva=wakanda"},
			wantError: false,
		},
		{
			name:      "no mixing pipelinename and label",
			command:   commandV1beta1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "--label", "viva=wakanda", "pr3-1"},
			wantError: true,
		},

		{
			name:      "limit pipelineruns greater than maximum case",
			command:   commandV1beta1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "--limit", fmt.Sprintf("%d", 7)},
			wantError: false,
		},
		{
			name:      "limit pipelineruns with output flag set",
			command:   commandV1beta1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "-n", "namespace", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}", "--limit", fmt.Sprintf("%d", 2)},
			wantError: false,
		},
		{
			name:      "print in reverse",
			command:   commandV1beta1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "--reverse", "-n", "namespace"},
			wantError: false,
		},
		{
			name:      "print in reverse with output flag",
			command:   commandV1beta1(t, prs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "--reverse", "-n", "namespace", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
		{
			name:      "print pipelineruns in all namespaces",
			command:   commandV1beta1(t, prsMultipleNs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "--all-namespaces"},
			wantError: false,
		},
		{
			name:      "print pipelineruns without headers",
			command:   commandV1beta1(t, prsMultipleNs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "--no-headers"},
			wantError: false,
		},
		{
			name:      "print pipelineruns in all namespaces without headers",
			command:   commandV1beta1(t, prsMultipleNs, clock.Now(), ns, version, dc1),
			args:      []string{"list", "--all-namespaces", "--no-headers"},
			wantError: false,
		},
	}

	for _, td := range tests {
		t.Run(td.name, func(t *testing.T) {
			got, err := test.ExecuteCommand(td.command, td.args...)

			if !td.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			golden.Assert(t, got, strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
		})
	}
}

func TestListPipeline_empty(t *testing.T) {
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
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pipeline := Command(p)
	output, err := test.ExecuteCommand(pipeline, "list", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "No PipelineRuns found\n", output)
}

func commandV1alpha1(t *testing.T, prs []*v1alpha1.PipelineRun, now time.Time, ns []*corev1.Namespace, version string, dc dynamic.Interface) *cobra.Command {
	// fake clock advanced by 1 hour
	clock := clockwork.NewFakeClockAt(now)
	clock.Advance(time.Duration(60) * time.Minute)

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}

	return Command(p)
}

func commandV1beta1(t *testing.T, prs []*v1beta1.PipelineRun, now time.Time, ns []*corev1.Namespace, version string, dc dynamic.Interface) *cobra.Command {
	// fake clock advanced by 1 hour
	clock := clockwork.NewFakeClockAt(now)
	clock.Advance(time.Duration(60) * time.Minute)

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{PipelineRuns: prs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}

	return Command(p)
}
