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
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggertest "github.com/tektoncd/triggers/test"
	tb "github.com/tektoncd/triggers/test/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListTriggerBinding(t *testing.T) {
	now := time.Now()

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random",
			},
		},
	}

	tbs := []*v1alpha1.TriggerBinding{
		tb.TriggerBinding("tb1", "foo", cb.TriggerBindingCreationTime(now.Add(-2*time.Minute))),
		tb.TriggerBinding("tb2", "foo", cb.TriggerBindingCreationTime(now.Add(-30*time.Second))),
		tb.TriggerBinding("tb3", "foo", cb.TriggerBindingCreationTime(now.Add(-200*time.Hour))),
		tb.TriggerBinding("tb4", "foo"),
	}

	tests := []struct {
		name      string
		command   *cobra.Command
		args      []string
		expected  []string
		wantError bool
	}{
		{
			name:    "Invalid namespace",
			command: command(t, tbs, now, ns),
			args:    []string{"list", "-n", "default"},
			expected: []string{
				"Error: namespaces \"default\" not found\n",
			},
			wantError: true,
		},
		{
			name:    "No TriggerBinding",
			command: command(t, tbs, now, ns),
			args:    []string{"list", "-n", "random"},
			expected: []string{
				"No triggerbindings found\n",
			},
			wantError: false,
		},
		{
			name:    "Multiple TriggerBinding",
			command: command(t, tbs, now, ns),
			args:    []string{"list", "-n", "foo"},
			expected: []string{
				"NAME   AGE",
				"tb1    2 minutes ago",
				"tb2    30 seconds ago",
				"tb3    1 week ago",
				"tb4    ---",
				"",
			},
			wantError: false,
		},
		{
			name:    "Multiple TriggerBinding with output format",
			command: command(t, tbs, now, ns),
			args:    []string{"list", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			expected: []string{
				"tb1",
				"tb2",
				"tb3",
				"tb4",
				"",
			},
			wantError: false,
		},
	}

	for _, td := range tests {
		t.Run(td.name, func(t *testing.T) {
			got, err := test.ExecuteCommand(td.command, td.args...)

			if err != nil && !td.wantError {
				t.Errorf("Unexpected error: %v", err)
			}
			test.AssertOutput(t, strings.Join(td.expected, "\n"), got)
		})
	}
}

func command(t *testing.T, tbs []*v1alpha1.TriggerBinding, now time.Time, ns []*corev1.Namespace) *cobra.Command {
	// fake clock advanced by 1 hour
	clock := clockwork.NewFakeClockAt(now)

	cs := test.SeedTestResources(t, triggertest.Resources{TriggerBindings: tbs, Namespaces: ns})

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Triggers: cs.Triggers}

	return Command(p)
}
