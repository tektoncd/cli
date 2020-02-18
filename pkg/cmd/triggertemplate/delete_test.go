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

package triggertemplate

import (
	"io"
	"strings"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggertest "github.com/tektoncd/triggers/test"
	tb "github.com/tektoncd/triggers/test/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestTriggerTemplateDelete(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seeds := make([]triggertest.Clients, 0)
	for i := 0; i < 5; i++ {
		tts := []*v1alpha1.TriggerTemplate{
			tb.TriggerTemplate("tt-1", "ns"),
			tb.TriggerTemplate("tt-2", "ns"),
			tb.TriggerTemplate("tt-3", "ns"),
		}
		ctx, _ := rtesting.SetupFakeContext(t)
		cs := triggertest.SeedResources(t, ctx, triggertest.Resources{TriggerTemplates: tts, Namespaces: ns})
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
			command:     []string{"rm", "tt-1", "-n", "invalid"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "namespaces \"invalid\" not found",
		},
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "tt-1", "-n", "ns", "-f"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "TriggerTemplates deleted: \"tt-1\"\n",
		},
		{
			name:        "With force delete flag (shorthand), multiple triggertemplates",
			command:     []string{"rm", "tt-2", "tt-3", "-n", "ns", "-f"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "TriggerTemplates deleted: \"tt-2\", \"tt-3\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "tt-1", "-n", "ns", "--force"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   false,
			want:        "TriggerTemplates deleted: \"tt-1\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "tt-1", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting triggertemplate \"tt-1\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "tt-1", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete triggertemplate \"tt-1\" (y/n): TriggerTemplates deleted: \"tt-1\"\n",
		},
		{
			name:        "Without force delete flag, reply yes, multiple triggertemplates",
			command:     []string{"rm", "tt-2", "tt-3", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete triggertemplate \"tt-2\", \"tt-3\" (y/n): TriggerTemplates deleted: \"tt-2\", \"tt-3\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   true,
			want:        "failed to delete triggertemplate \"nonexistent\": triggertemplates.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			input:       seeds[3],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all triggertemplates in namespace \"ns\" (y/n): All TriggerTemplates deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   false,
			want:        "All TriggerTemplates deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from using triggertemplate name with --all",
			command:     []string{"delete", "tt", "--all", "-n", "ns"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from using triggertemplate delete with no names or --all",
			command:     []string{"delete"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   true,
			want:        "must provide triggertemplate name(s) or use --all flag with delete",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Triggers: tp.input.Triggers, Kube: tp.input.Kube}
			triggerTemplate := Command(p)

			if tp.inputStream != nil {
				triggerTemplate.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(triggerTemplate, tp.command...)
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
