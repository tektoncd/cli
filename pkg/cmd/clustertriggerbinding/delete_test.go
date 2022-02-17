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
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggertest "github.com/tektoncd/triggers/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterTriggerBindingDelete(t *testing.T) {
	seeds := make([]*test.Params, 0)
	for i := 0; i < 5; i++ {

		ctbs := []*v1beta1.ClusterTriggerBinding{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ctb-1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ctb-2",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ctb-3",
				},
			},
		}
		cs := test.SeedTestResources(t, triggertest.Resources{ClusterTriggerBindings: ctbs})
		cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredV1beta1CTB(ctbs[0], "v1beta1"),
			cb.UnstructuredV1beta1CTB(ctbs[1], "v1beta1"),
			cb.UnstructuredV1beta1CTB(ctbs[2], "v1beta1"),
		)
		if err != nil {
			t.Errorf("unable to create dynamic client: %v", err)
		}
		p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Dynamic: dc}
		seeds = append(seeds, p)
	}

	testParams := []struct {
		name        string
		command     []string
		input       *test.Params
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
			want:        "canceled deleting clustertriggerbinding(s) \"ctb-1\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "ctb-1"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete clustertriggerbinding(s) \"ctb-1\" (y/n): ClusterTriggerBindings deleted: \"ctb-1\"\n",
		},
		{
			name:        "Without force delete flag, reply yes, multiple ClusterTriggerBindings",
			command:     []string{"rm", "ctb-2", "ctb-3"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete clustertriggerbinding(s) \"ctb-2\", \"ctb-3\" (y/n): ClusterTriggerBindings deleted: \"ctb-2\", \"ctb-3\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "clustertriggerbindings.triggers.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "clustertriggerbindings.triggers.tekton.dev \"nonexistent\" not found; clustertriggerbindings.triggers.tekton.dev \"nonexistent2\" not found",
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
			command:     []string{"delete", "ctb-2", "--all"},
			input:       seeds[1],
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
		{
			name:        "Delete the ClusterTriggerBinding present and give error for non-existent ClusterTriggerBinding",
			command:     []string{"delete", "nonexistent", "ctb-2"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   true,
			want:        "clustertriggerbindings.triggers.tekton.dev \"nonexistent\" not found",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			clustertriggerBinding := Command(tp.input)

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
