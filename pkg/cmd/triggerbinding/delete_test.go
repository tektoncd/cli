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

package triggerbinding

import (
	"io"
	"strings"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggertest "github.com/tektoncd/triggers/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTriggerBindingDelete(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seeds := make([]triggertest.Clients, 0)
	for i := 0; i < 5; i++ {
		tbs := []*v1alpha1.TriggerBinding{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tb-1",
					Namespace: "ns",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tb-2",
					Namespace: "ns",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tb-3",
					Namespace: "ns",
				},
			},
		}
		ctx, _ := test.SetupFakeContext(t)
		cs := triggertest.SeedResources(t, ctx, triggertest.Resources{TriggerBindings: tbs, Namespaces: ns})
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
			name:        "Invalid namespace",
			command:     []string{"rm", "tb-1", "-n", "invalid"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete TriggerBinding \"tb-1\": triggerbindings.triggers.tekton.dev \"tb-1\" not found",
		},
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "tb-1", "-n", "ns", "-f"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "TriggerBindings deleted: \"tb-1\"\n",
		},
		{
			name:        "With force delete flag (shorthand), multiple TriggerBindings",
			command:     []string{"rm", "tb-2", "tb-3", "-n", "ns", "-f"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "TriggerBindings deleted: \"tb-2\", \"tb-3\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "tb-1", "-n", "ns", "--force"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   false,
			want:        "TriggerBindings deleted: \"tb-1\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "tb-1", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting triggerbinding(s) \"tb-1\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "tb-1", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete triggerbinding(s) \"tb-1\" (y/n): TriggerBindings deleted: \"tb-1\"\n",
		},
		{
			name:        "Without force delete flag, reply yes, multiple TriggerBindings",
			command:     []string{"rm", "tb-2", "tb-3", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete triggerbinding(s) \"tb-2\", \"tb-3\" (y/n): TriggerBindings deleted: \"tb-2\", \"tb-3\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete TriggerBinding \"nonexistent\": triggerbindings.triggers.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete TriggerBinding \"nonexistent\": triggerbindings.triggers.tekton.dev \"nonexistent\" not found; failed to delete TriggerBinding \"nonexistent2\": triggerbindings.triggers.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			input:       seeds[3],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all triggerbindings in namespace \"ns\" (y/n): All TriggerBindings deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   false,
			want:        "All TriggerBindings deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from using triggerbinding name with --all",
			command:     []string{"delete", "tb", "--all", "-n", "ns"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from using triggerbinding delete with no names or --all",
			command:     []string{"delete"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   true,
			want:        "must provide triggerbinding name(s) or use --all flag with delete",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Triggers: tp.input.Triggers, Kube: tp.input.Kube}
			triggerBinding := Command(p)

			if tp.inputStream != nil {
				triggerBinding.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(triggerBinding, tp.command...)
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
