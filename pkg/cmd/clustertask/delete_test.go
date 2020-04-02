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

package clustertask

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
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"k8s.io/client-go/dynamic"
)

func TestClusterTaskDelete(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()

	type clients struct {
		pipelineClient pipelinetest.Clients
		dynamicClient  dynamic.Interface
	}

	seeds := make([]clients, 0)
	for i := 0; i < 5; i++ {
		clustertasks := []*v1alpha1.ClusterTask{
			tb.ClusterTask("tomatoes", cb.ClusterTaskCreationTime(clock.Now().Add(-1*time.Minute))),
			tb.ClusterTask("tomatoes2", cb.ClusterTaskCreationTime(clock.Now().Add(-1*time.Minute))),
			tb.ClusterTask("tomatoes3", cb.ClusterTaskCreationTime(clock.Now().Add(-1*time.Minute))),
		}
		cs, _ := test.SeedTestData(t, pipelinetest.Data{ClusterTasks: clustertasks})
		cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredCT(clustertasks[0], version),
			cb.UnstructuredCT(clustertasks[1], version),
			cb.UnstructuredCT(clustertasks[2], version),
		)
		if err != nil {
			t.Errorf("unable to create dynamic clinet: %v", err)
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
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "tomatoes", "-f"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "ClusterTasks deleted: \"tomatoes\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "tomatoes", "--force"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "ClusterTasks deleted: \"tomatoes\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "tomatoes"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting clustertask \"tomatoes\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "tomatoes"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete clustertask \"tomatoes\" (y/n): ClusterTasks deleted: \"tomatoes\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   true,
			want:        "failed to delete clustertask \"nonexistent\": clustertasks.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   true,
			want:        "failed to delete clustertask \"nonexistent\": clustertasks.tekton.dev \"nonexistent\" not found; failed to delete clustertask \"nonexistent2\": clustertasks.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "With force delete flag, reply yes, multiple clustertasks",
			command:     []string{"rm", "tomatoes2", "tomatoes3", "-f"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "ClusterTasks deleted: \"tomatoes2\", \"tomatoes3\"\n",
		},
		{
			name:        "Without force delete flag, reply yes, multiple clustertasks",
			command:     []string{"rm", "tomatoes2", "tomatoes3"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete clustertask \"tomatoes2\", \"tomatoes3\" (y/n): ClusterTasks deleted: \"tomatoes2\", \"tomatoes3\"\n",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all clustertasks (y/n): All ClusterTasks deleted\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All ClusterTasks deleted\n",
		},
		{
			name:        "Error from using clustertask name with --all",
			command:     []string{"delete", "ct", "--all"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from using clustertask delete with no names or --all",
			command:     []string{"delete"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "must provide clustertask name(s) or use --all flag with delete",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Dynamic: tp.dynamic}
			clustertask := Command(p)

			if tp.inputStream != nil {
				clustertask.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(clustertask, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("Error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else {
				if err != nil {
					t.Errorf("Unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}

func TestClusterTaskDelete_v1beta1(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()

	type clients struct {
		pipelineClient pipelinetest.Clients
		dynamicClient  dynamic.Interface
	}

	seeds := make([]clients, 0)
	for i := 0; i < 5; i++ {
		clustertasks := []*v1alpha1.ClusterTask{
			tb.ClusterTask("tomatoes", cb.ClusterTaskCreationTime(clock.Now().Add(-1*time.Minute))),
			tb.ClusterTask("tomatoes2", cb.ClusterTaskCreationTime(clock.Now().Add(-1*time.Minute))),
			tb.ClusterTask("tomatoes3", cb.ClusterTaskCreationTime(clock.Now().Add(-1*time.Minute))),
		}
		cs, _ := test.SeedTestData(t, pipelinetest.Data{ClusterTasks: clustertasks})
		cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredCT(clustertasks[0], version),
			cb.UnstructuredCT(clustertasks[1], version),
			cb.UnstructuredCT(clustertasks[2], version),
		)
		if err != nil {
			t.Errorf("unable to create dynamic clinet: %v", err)
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
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "tomatoes", "-f"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "ClusterTasks deleted: \"tomatoes\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "tomatoes", "--force"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "ClusterTasks deleted: \"tomatoes\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "tomatoes"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting clustertask \"tomatoes\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "tomatoes"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete clustertask \"tomatoes\" (y/n): ClusterTasks deleted: \"tomatoes\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   true,
			want:        "failed to delete clustertask \"nonexistent\": clustertasks.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   true,
			want:        "failed to delete clustertask \"nonexistent\": clustertasks.tekton.dev \"nonexistent\" not found; failed to delete clustertask \"nonexistent2\": clustertasks.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "With force delete flag, reply yes, multiple clustertasks",
			command:     []string{"rm", "tomatoes2", "tomatoes3", "-f"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "ClusterTasks deleted: \"tomatoes2\", \"tomatoes3\"\n",
		},
		{
			name:        "Without force delete flag, reply yes, multiple clustertasks",
			command:     []string{"rm", "tomatoes2", "tomatoes3"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete clustertask \"tomatoes2\", \"tomatoes3\" (y/n): ClusterTasks deleted: \"tomatoes2\", \"tomatoes3\"\n",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all clustertasks (y/n): All ClusterTasks deleted\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All ClusterTasks deleted\n",
		},
		{
			name:        "Error from using clustertask name with --all",
			command:     []string{"delete", "ct", "--all"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from using clustertask delete with no names or --all",
			command:     []string{"delete"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "must provide clustertask name(s) or use --all flag with delete",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Dynamic: tp.dynamic}
			clustertask := Command(p)

			if tp.inputStream != nil {
				clustertask.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(clustertask, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("Error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else {
				if err != nil {
					t.Errorf("Unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}
