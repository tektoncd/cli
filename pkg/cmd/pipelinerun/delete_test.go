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
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinetest "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestPipelineRunDelete(t *testing.T) {
	clock := clockwork.NewFakeClock()

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seeds := make([]pipelinetest.Clients, 0)
	for i := 0; i < 5; i++ {
		cs, _ := test.SeedTestData(t, pipelinetest.Data{
			Pipelines: []*v1alpha1.Pipeline{
				tb.Pipeline("pipeline", "ns",
					// created  5 minutes back
					cb.PipelineCreationTimestamp(clock.Now().Add(-5*time.Minute)),
				),
			},
			PipelineRuns: []*v1alpha1.PipelineRun{
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
				tb.PipelineRun("pipeline-run-3", "ns",
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
			},

			Namespaces: ns,
		})
		seeds = append(seeds, cs)
	}

	testParams := []struct {
		name        string
		command     []string
		input       pipelinetest.Clients
		inputStream io.Reader
		wantError   bool
		want        string
	}{
		{
			name:        "Invalid namespace",
			command:     []string{"rm", "pipeline-run-1", "-n", "invalid"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "namespaces \"invalid\" not found",
		},
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "pipeline-run-1", "-n", "ns", "-f"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "PipelineRuns deleted: \"pipeline-run-1\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "pipeline-run-1", "-n", "ns", "--force"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   false,
			want:        "PipelineRuns deleted: \"pipeline-run-1\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "pipeline-run-1", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting pipelinerun \"pipeline-run-1\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "pipeline-run-1", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete pipelinerun \"pipeline-run-1\" (y/n): PipelineRuns deleted: \"pipeline-run-1\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete pipelinerun \"nonexistent\": pipelineruns.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete pipelinerun \"nonexistent\": pipelineruns.tekton.dev \"nonexistent\" not found; failed to delete pipelinerun \"nonexistent2\": pipelineruns.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Attempt remove forgetting to include pipelinerun names",
			command:     []string{"rm", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "must provide pipelinerun name(s) or use --pipeline flag or --all flag to use delete",
		},
		{
			name:        "Remove pipelineruns of a pipeline",
			command:     []string{"rm", "--pipeline", "pipeline", "-n", "ns"},
			input:       seeds[0],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all pipelineruns related to pipeline \"pipeline\" (y/n): PipelineRuns deleted: \"pipeline-run-2\", \"pipeline-run-3\"\n",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			input:       seeds[3],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all pipelineruns in namespace \"ns\" (y/n): All PipelineRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   false,
			want:        "All PipelineRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all keeping 2",
			command:     []string{"delete", "--all", "-f", "--keep", "2", "-n", "ns"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   false,
			want:        "All but 2 PipelineRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Keep -1 is a no go",
			command:     []string{"delete", "--all", "-f", "--keep", "-1", "-n", "ns"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   true,
			want:        "keep option should not be lower than 0",
		},
		{
			name:        "Error from using pipelinerun name with --all",
			command:     []string{"delete", "pipelinerun", "--all", "-n", "ns"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube}
			pipelinerun := Command(p)

			if tp.inputStream != nil {
				pipelinerun.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(pipelinerun, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				} else {
					test.AssertOutput(t, tp.want, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}
