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
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggertest "github.com/tektoncd/triggers/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTriggerTemplateDelete(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seeds := make([]*test.Params, 0)
	for i := 0; i < 5; i++ {
		tts := []*v1beta1.TriggerTemplate{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tt-1",
					Namespace: "ns",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tt-2",
					Namespace: "ns",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tt-3",
					Namespace: "ns",
				},
			},
		}
		cs := test.SeedTestResources(t, triggertest.Resources{TriggerTemplates: tts, Namespaces: ns})
		cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"triggertemplate"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredV1beta1TT(tts[0], "v1beta1"),
			cb.UnstructuredV1beta1TT(tts[1], "v1beta1"),
			cb.UnstructuredV1beta1TT(tts[2], "v1beta1"),
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
			name:        "Invalid namespace",
			command:     []string{"rm", "tt-1", "-n", "invalid"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "triggertemplates.triggers.tekton.dev \"tt-1\" not found",
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
			want:        "canceled deleting triggertemplate(s) \"tt-1\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "tt-1", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete triggertemplate(s) \"tt-1\" (y/n): TriggerTemplates deleted: \"tt-1\"\n",
		},
		{
			name:        "Without force delete flag, reply yes, multiple triggertemplates",
			command:     []string{"rm", "tt-2", "tt-3", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete triggertemplate(s) \"tt-2\", \"tt-3\" (y/n): TriggerTemplates deleted: \"tt-2\", \"tt-3\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "triggertemplates.triggers.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "triggertemplates.triggers.tekton.dev \"nonexistent\" not found; triggertemplates.triggers.tekton.dev \"nonexistent2\" not found",
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
			command:     []string{"delete", "tt-2", "--all", "-n", "ns"},
			input:       seeds[1],
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
		{
			name:        "Delete the TriggerTemplate present and give error for non-existent TriggerTemplate",
			command:     []string{"delete", "nonexistent", "tt-2"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   true,
			want:        "triggertemplates.triggers.tekton.dev \"nonexistent\" not found",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			triggerTemplate := Command(tp.input)
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
