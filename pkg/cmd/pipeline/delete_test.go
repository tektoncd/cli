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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinetest "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestPipelineDelete(t *testing.T) {
	clock := clockwork.NewFakeClock()

	seeds := make([]pipelinetest.Clients, 0)
	for i := 0; i < 8; i++ {
		cs, _ := test.SeedTestData(t, pipelinetest.Data{
			Pipelines: []*v1alpha1.Pipeline{
				tb.Pipeline("pipeline", "ns",
					// created  5 minutes back
					cb.PipelineCreationTimestamp(clock.Now().Add(-5*time.Minute)),
				),
				tb.Pipeline("pipeline2", "ns",
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
			},
			Namespaces: []*corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns",
					},
				},
			},
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
			command:     []string{"rm", "pipeline", "-n", "invalid"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "namespaces \"invalid\" not found",
		},
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "pipeline", "-n", "ns", "-f"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "Pipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "pipeline", "-n", "ns", "--force"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   false,
			want:        "Pipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "pipeline", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting pipeline \"pipeline\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "pipeline", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete pipeline \"pipeline\" (y/n): Pipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete pipeline \"nonexistent\": pipelines.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete pipeline \"nonexistent\": pipelines.tekton.dev \"nonexistent\" not found; failed to delete pipeline \"nonexistent2\": pipelines.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Remove multiple non existent resources with --prs flag",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns", "--prs"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete pipeline \"nonexistent\": pipelines.tekton.dev \"nonexistent\" not found; failed to delete pipeline \"nonexistent2\": pipelines.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "With delete all flag, reply yes",
			command:     []string{"rm", "pipeline", "-n", "ns", "--prs"},
			input:       seeds[3],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete pipeline and related resources \"pipeline\" (y/n): PipelineRuns deleted: \"pipeline-run-1\", \"pipeline-run-2\"\nPipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "With delete all and force delete flag",
			command:     []string{"rm", "pipeline", "-n", "ns", "-f", "--prs"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   false,
			want:        "PipelineRuns deleted: \"pipeline-run-1\", \"pipeline-run-2\"\nPipelines deleted: \"pipeline\"\n",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			input:       seeds[5],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all pipelines in namespace \"ns\" (y/n): All Pipelines deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			input:       seeds[6],
			inputStream: nil,
			wantError:   false,
			want:        "All Pipelines deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from using pipeline name with --all",
			command:     []string{"delete", "pipeline", "--all", "-n", "ns"},
			input:       seeds[6],
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from using --all with --prs",
			command:     []string{"delete", "--all", "--prs", "-n", "ns"},
			input:       seeds[6],
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "With force delete flag (shorthand), multiple pipelines",
			command:     []string{"rm", "pipeline", "pipeline2", "-n", "ns", "-f"},
			input:       seeds[7],
			inputStream: nil,
			wantError:   false,
			want:        "Pipelines deleted: \"pipeline\", \"pipeline2\"\n",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube}
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
