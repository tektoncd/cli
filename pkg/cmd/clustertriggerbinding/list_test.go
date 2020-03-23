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
	"fmt"
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
	"gotest.tools/v3/golden"
)

func TestListClusterTriggerBinding(t *testing.T) {
	now := time.Now()

	ctbs := []*v1alpha1.ClusterTriggerBinding{
		tb.ClusterTriggerBinding("ctb1", cb.ClusterTriggerBindingCreationTime(now.Add(-2*time.Minute))),
		tb.ClusterTriggerBinding("ctb2", cb.ClusterTriggerBindingCreationTime(now.Add(-30*time.Second))),
		tb.ClusterTriggerBinding("ctb3", cb.ClusterTriggerBindingCreationTime(now.Add(-200*time.Hour))),
		tb.ClusterTriggerBinding("ctb4"),
	}

	tests := []struct {
		name      string
		command   *cobra.Command
		args      []string
		wantError bool
	}{
		{
			name:      "No ClusterTriggerBindings",
			command:   command(t, []*v1alpha1.ClusterTriggerBinding{}, now),
			args:      []string{"list"},
			wantError: false,
		},
		{
			name:      "Multiple ClusterTriggerBindings",
			command:   command(t, ctbs, now),
			args:      []string{"list", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "By output as name",
			command:   command(t, ctbs, now),
			args:      []string{"list", "-n", "foo", "-o", "name"},
			wantError: false,
		},
		{
			name:      "Multiple ClusterTriggerBindings with output format",
			command:   command(t, ctbs, now),
			args:      []string{"list", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
	}

	for _, td := range tests {
		t.Run(td.name, func(t *testing.T) {
			got, err := test.ExecuteCommand(td.command, td.args...)

			if err != nil && !td.wantError {
				t.Errorf("Unexpected error: %v", err)
			}
			golden.Assert(t, got, strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
		})
	}
}

func command(t *testing.T, ctbs []*v1alpha1.ClusterTriggerBinding, now time.Time) *cobra.Command {
	// fake clock advanced by 1 hour
	clock := clockwork.NewFakeClockAt(now)

	cs := test.SeedTestResources(t, triggertest.Resources{ClusterTriggerBindings: ctbs})

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Triggers: cs.Triggers}

	return Command(p)
}
