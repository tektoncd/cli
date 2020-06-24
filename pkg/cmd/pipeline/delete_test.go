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
	"io"
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestPipelineDelete(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()

	pdata := []*v1alpha1.Pipeline{
		tb.Pipeline("pipeline",
			tb.PipelineNamespace("ns"),
			// created  5 minutes back
			cb.PipelineCreationTimestamp(clock.Now().Add(-5*time.Minute)),
		),
		tb.Pipeline("pipeline2",
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
		tb.PipelineRun("pipeline-run-2",
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
	type clients struct {
		pipelineClient pipelinetest.Clients
		dynamicClient  dynamic.Interface
	}
	seeds := make([]clients, 0)

	for i := 0; i < 8; i++ {
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
			cb.UnstructuredP(pdata[1], version),
			cb.UnstructuredPR(prdata[0], version),
			cb.UnstructuredPR(prdata[1], version),
		)
		if err != nil {
			t.Errorf("unable to create dynamic client: %v", err)
		}

		seeds = append(seeds, clients{cs, dc})
	}

	testParams := []struct {
		name        string
		command     []string
		dynamic     dynamic.Interface
		input       pipelinetest.Clients
		inputStream io.Reader
		wantError   bool
		want        string
	}{
		{
			name:        "Invalid namespace",
			command:     []string{"rm", "pipeline", "-n", "invalid"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "namespaces \"invalid\" not found",
		},
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "pipeline", "-n", "ns", "-f"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Pipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "pipeline", "-n", "ns", "--force"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Pipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "pipeline", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting pipeline \"pipeline\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "pipeline", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete pipeline \"pipeline\" (y/n): Pipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete pipeline \"nonexistent\": pipelines.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete pipeline \"nonexistent\": pipelines.tekton.dev \"nonexistent\" not found; failed to delete pipeline \"nonexistent2\": pipelines.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Remove multiple non existent resources with --prs flag",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns", "--prs"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete pipeline \"nonexistent\": pipelines.tekton.dev \"nonexistent\" not found; failed to delete pipeline \"nonexistent2\": pipelines.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "With --prs flag, reply yes",
			command:     []string{"rm", "pipeline", "-n", "ns", "--prs"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete pipeline and related resources \"pipeline\" (y/n): PipelineRuns deleted: \"pipeline-run-1\", \"pipeline-run-2\"\nPipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "With --prs and force delete flag",
			command:     []string{"rm", "pipeline", "-n", "ns", "-f", "--prs"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "PipelineRuns deleted: \"pipeline-run-1\", \"pipeline-run-2\"\nPipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all pipelines in namespace \"ns\" (y/n): All Pipelines deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All Pipelines deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from using pipeline name with --all",
			command:     []string{"delete", "pipeline", "--all", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from using --all with --prs",
			command:     []string{"delete", "--all", "--prs", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "With force delete flag (shorthand), multiple pipelines",
			command:     []string{"rm", "pipeline", "pipeline2", "-n", "ns", "-f"},
			dynamic:     seeds[7].dynamicClient,
			input:       seeds[7].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Pipelines deleted: \"pipeline\", \"pipeline2\"\n",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube, Dynamic: tp.dynamic}
			pipeline := Command(p)

			if tp.inputStream != nil {
				pipeline.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(pipeline, tp.command...)
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

func TestPipelineDelete_v1beta1(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()

	pdata := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: "ns",
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline2",
				Namespace: "ns",
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	prdata := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run-1",
				Namespace:         "ns",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
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
					// pipeline run starts now
					StartTime: &metav1.Time{Time: clock.Now()},
					// takes 10 minutes to complete
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline-run-2",
				Namespace:         "ns",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
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
					// pipeline run starts now
					StartTime: &metav1.Time{Time: clock.Now()},
					// takes 10 minutes to complete
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
				},
			},
		},
	}

	type clients struct {
		pipelineClient pipelinev1beta1test.Clients
		dynamicClient  dynamic.Interface
	}
	seeds := make([]clients, 0)

	for i := 0; i < 8; i++ {
		cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
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
			cb.UnstructuredV1beta1P(pdata[0], version),
			cb.UnstructuredV1beta1P(pdata[1], version),
			cb.UnstructuredV1beta1PR(prdata[0], version),
			cb.UnstructuredV1beta1PR(prdata[1], version),
		)
		if err != nil {
			t.Errorf("unable to create dynamic client: %v", err)
		}

		seeds = append(seeds, clients{cs, dc})
	}

	testParams := []struct {
		name        string
		command     []string
		dynamic     dynamic.Interface
		input       pipelinev1beta1test.Clients
		inputStream io.Reader
		wantError   bool
		want        string
	}{
		{
			name:        "Invalid namespace",
			command:     []string{"rm", "pipeline", "-n", "invalid"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "namespaces \"invalid\" not found",
		},
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "pipeline", "-n", "ns", "-f"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Pipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "pipeline", "-n", "ns", "--force"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Pipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "pipeline", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting pipeline \"pipeline\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "pipeline", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete pipeline \"pipeline\" (y/n): Pipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete pipeline \"nonexistent\": pipelines.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete pipeline \"nonexistent\": pipelines.tekton.dev \"nonexistent\" not found; failed to delete pipeline \"nonexistent2\": pipelines.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Remove multiple non existent resources with --prs flag",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns", "--prs"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete pipeline \"nonexistent\": pipelines.tekton.dev \"nonexistent\" not found; failed to delete pipeline \"nonexistent2\": pipelines.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "With delete all flag, reply yes",
			command:     []string{"rm", "pipeline", "-n", "ns", "--prs"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete pipeline and related resources \"pipeline\" (y/n): PipelineRuns deleted: \"pipeline-run-1\", \"pipeline-run-2\"\nPipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "With delete all and force delete flag",
			command:     []string{"rm", "pipeline", "-n", "ns", "-f", "--prs"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "PipelineRuns deleted: \"pipeline-run-1\", \"pipeline-run-2\"\nPipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all pipelines in namespace \"ns\" (y/n): All Pipelines deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All Pipelines deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from using pipeline name with --all",
			command:     []string{"delete", "pipeline", "--all", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from using --all with --prs",
			command:     []string{"delete", "--all", "--prs", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "With force delete flag (shorthand), multiple pipelines",
			command:     []string{"rm", "pipeline", "pipeline2", "-n", "ns", "-f"},
			dynamic:     seeds[7].dynamicClient,
			input:       seeds[7].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Pipelines deleted: \"pipeline\", \"pipeline2\"\n",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube, Dynamic: tp.dynamic}
			pipeline := Command(p)

			if tp.inputStream != nil {
				pipeline.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(pipeline, tp.command...)
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
