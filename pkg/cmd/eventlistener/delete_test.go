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

package eventlistener

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

func TestEventListenerDelete(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seeds := make([]*test.Params, 0)
	for i := 0; i < 5; i++ {
		els := []*v1beta1.EventListener{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "el-1",
					Namespace: "ns",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "el-2",
					Namespace: "ns",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "el-3",
					Namespace: "ns",
				},
			},
		}

		cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els, Namespaces: ns})
		cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredV1beta1EL(els[0], "v1beta1"),
			cb.UnstructuredV1beta1EL(els[1], "v1beta1"),
			cb.UnstructuredV1beta1EL(els[2], "v1beta1"),
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
			command:     []string{"rm", "el-1", "-n", "invalid"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "failed to get EventListener el-1: eventlisteners.triggers.tekton.dev \"el-1\" not found",
		},
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "el-1", "-n", "ns", "-f"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "EventListeners deleted: \"el-1\"\n",
		},
		{
			name:        "With force delete flag (shorthand), multiple eventlisteners",
			command:     []string{"rm", "el-2", "el-3", "-n", "ns", "-f"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "EventListeners deleted: \"el-2\", \"el-3\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "el-1", "-n", "ns", "--force"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   false,
			want:        "EventListeners deleted: \"el-1\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "el-1", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting eventlistener(s) \"el-1\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "el-1", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete eventlistener(s) \"el-1\" (y/n): EventListeners deleted: \"el-1\"\n",
		},
		{
			name:        "Without force delete flag, reply yes, multiple eventlisteners",
			command:     []string{"rm", "el-2", "el-3", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete eventlistener(s) \"el-2\", \"el-3\" (y/n): EventListeners deleted: \"el-2\", \"el-3\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "failed to get EventListener nonexistent: eventlisteners.triggers.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "failed to get EventListener nonexistent: eventlisteners.triggers.tekton.dev \"nonexistent\" not found; failed to get EventListener nonexistent2: eventlisteners.triggers.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			input:       seeds[3],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all eventlisteners in namespace \"ns\" (y/n): All EventListeners deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   false,
			want:        "All EventListeners deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from using eventlistener name with --all",
			command:     []string{"delete", "el-2", "--all", "-n", "ns"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from using eventlistener delete with no names or --all",
			command:     []string{"delete"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   true,
			want:        "must provide eventlistener name(s) or use --all flag with delete",
		},
		{
			name:        "Delete the EventListener present and give error for non-existent EventListener",
			command:     []string{"delete", "nonexistent", "el-2"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   true,
			want:        "failed to get EventListener nonexistent: eventlisteners.triggers.tekton.dev \"nonexistent\" not found",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			eventListener := Command(tp.input)

			if tp.inputStream != nil {
				eventListener.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(eventListener, tp.command...)
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
