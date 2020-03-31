// Copyright © 2020 The Tekton Authors.
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

package clustertriggerbinding

import (
	"io"
	"strings"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggertest "github.com/tektoncd/triggers/test"
	tb "github.com/tektoncd/triggers/test/builder"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestClusterTriggerBindingDelete(t *testing.T) {
	seeds := make([]triggertest.Clients, 0)
	for i := 0; i < 5; i++ {
		ctbs := []*v1alpha1.ClusterTriggerBinding{
			tb.ClusterTriggerBinding("ctb-1"),
			tb.ClusterTriggerBinding("ctb-2"),
			tb.ClusterTriggerBinding("ctb-3"),
		}
		ctx, _ := rtesting.SetupFakeContext(t)
		cs := triggertest.SeedResources(t, ctx, triggertest.Resources{ClusterTriggerBindings: ctbs})
		seeds = append(seeds, cs)
	}

	testParams := []struct {
		name        string
		command     []string
		input       triggertest.Clients
		inputStream io.Reader
		wantError   bool
		want        string
	}{
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "ctb-1", "-f"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "ClusterTriggerBindings deleted: \"ctb-1\"\n",
		},
		{
			name:        "With force delete flag (shorthand), multiple ClusterTriggerBindings",
			command:     []string{"rm", "ctb-2", "ctb-3", "-f"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "ClusterTriggerBindings deleted: \"ctb-2\", \"ctb-3\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "ctb-1", "--force"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   false,
			want:        "ClusterTriggerBindings deleted: \"ctb-1\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "ctb-1"},
			input:       seeds[2],
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting clustertriggerbinding \"ctb-1\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "ctb-1"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete clustertriggerbinding \"ctb-1\" (y/n): ClusterTriggerBindings deleted: \"ctb-1\"\n",
		},
		{
			name:        "Without force delete flag, reply yes, multiple ClusterTriggerBindings",
			command:     []string{"rm", "ctb-2", "ctb-3"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete clustertriggerbinding \"ctb-2\", \"ctb-3\" (y/n): ClusterTriggerBindings deleted: \"ctb-2\", \"ctb-3\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete clustertriggerbinding \"nonexistent\": clustertriggerbindings.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete clustertriggerbinding \"nonexistent\": clustertriggerbindings.tekton.dev \"nonexistent\" not found; failed to delete clustertriggerbinding \"nonexistent2\": clustertriggerbindings.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all"},
			input:       seeds[3],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all clustertriggerbindings (y/n): All ClusterTriggerBindings deleted\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   false,
			want:        "All ClusterTriggerBindings deleted\n",
		},
		{
			name:        "Error from using clustertriggerbinding name with --all",
			command:     []string{"delete", "ctb", "--all"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from using clustertriggerbinding delete with no names or --all",
			command:     []string{"delete"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   true,
			want:        "must provide clustertriggerbinding name(s) or use --all flag with delete",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Triggers: tp.input.Triggers}
			clustertriggerBinding := Command(p)

			if tp.inputStream != nil {
				clustertriggerBinding.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(clustertriggerBinding, tp.command...)
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
