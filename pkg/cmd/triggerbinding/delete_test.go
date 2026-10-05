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
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
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

	seeds := make([]*test.Params, 0)
	for i := 0; i < 5; i++ {
		tbs := []*v1beta1.TriggerBinding{
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
		cs := test.SeedTestResources(t, triggertest.Resources{TriggerBindings: tbs, Namespaces: ns})
		cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"triggerbinding"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredV1beta1TB(tbs[0], "v1beta1"),
			cb.UnstructuredV1beta1TB(tbs[1], "v1beta1"),
			cb.UnstructuredV1beta1TB(tbs[2], "v1beta1"),
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
			command:     []string{"rm", "tb-1", "-n", "invalid"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "triggerbindings.triggers.tekton.dev \"tb-1\" not found",
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
			want:        "triggerbindings.triggers.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "triggerbindings.triggers.tekton.dev \"nonexistent\" not found; triggerbindings.triggers.tekton.dev \"nonexistent2\" not found",
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
			command:     []string{"delete", "tb-2", "--all", "-n", "ns"},
			input:       seeds[1],
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
		{
			name:        "Delete the TriggerBinding present and give error for non-existent TriggerBinding",
			command:     []string{"delete", "nonexistent", "tb-2"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   true,
			want:        "triggerbindings.triggers.tekton.dev \"nonexistent\" not found",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			triggerBinding := Command(tp.input)

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

func TestTriggerBindingDeleteStructuredOutput(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seeds := make([]*test.Params, 0)
	for i := 0; i < 3; i++ {
		tbs := []*v1beta1.TriggerBinding{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tb-1",
					Namespace: "ns",
				},
			},
		}

		cs := test.SeedTestResources(t, triggertest.Resources{TriggerBindings: tbs, Namespaces: ns})
		cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"triggerbinding"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredV1beta1TB(tbs[0], "v1beta1"),
		)
		if err != nil {
			t.Errorf("unable to create dynamic client: %v", err)
		}
		p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Dynamic: dc}
		seeds = append(seeds, p)
	}

	testParams := []struct {
		name      string
		command   []string
		input     *test.Params
		wantError bool
		want      string
	}{
		{
			name:      "Delete triggerbinding with output as json",
			command:   []string{"rm", "tb-1", "-n", "ns", "-o", "json"},
			input:     seeds[0],
			wantError: false,
			want:      "{\n    \"deleted\": [\n        \"tb-1\"\n    ]\n}\n",
		},
		{
			name:      "Delete triggerbinding with output as yaml",
			command:   []string{"rm", "tb-1", "-n", "ns", "-o", "yaml"},
			input:     seeds[1],
			wantError: false,
			want:      "deleted:\n- tb-1\n",
		},
		{
			name:      "Delete triggerbinding with invalid output format",
			command:   []string{"rm", "tb-1", "-n", "ns", "-o", "csv"},
			input:     seeds[2],
			wantError: true,
			want:      "invalid output format \"csv\": must be json or yaml",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			tb := Command(tp.input)

			out, err := test.ExecuteCommand(tb, tp.command...)
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

func TestTriggerBindingDeleteBulkStructuredOutput(t *testing.T) {
	ns := []*corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: "ns"}}}
	tbs := []*v1beta1.TriggerBinding{
		{ObjectMeta: metav1.ObjectMeta{Name: "tb-1", Namespace: "ns"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "tb-2", Namespace: "ns"}},
	}

	seed := func(t *testing.T) *test.Params {
		t.Helper()
		cs := test.SeedTestResources(t, triggertest.Resources{TriggerBindings: tbs, Namespaces: ns})
		cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"triggerbinding"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredV1beta1TB(tbs[0], "v1beta1"),
			cb.UnstructuredV1beta1TB(tbs[1], "v1beta1"),
		)
		if err != nil {
			t.Fatalf("unable to create dynamic client: %v", err)
		}
		return &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Dynamic: dc}
	}

	out, err := test.ExecuteCommand(Command(seed(t)), "delete", "--all", "-n", "ns", "-o", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	test.AssertOutput(t, "{\n    \"deleted\": [\n        \"tb-1\",\n        \"tb-2\"\n    ]\n}\n", out)

	out, err = test.ExecuteCommand(Command(seed(t)), "delete", "--all", "-n", "ns", "-o", "yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	test.AssertOutput(t, "deleted:\n- tb-1\n- tb-2\n", out)
}
