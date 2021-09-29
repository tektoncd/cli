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
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggertest "github.com/tektoncd/triggers/test"
	"gotest.tools/v3/golden"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestListClusterTriggerBinding(t *testing.T) {
	now := time.Now()

	ctbs := []*v1beta1.ClusterTriggerBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "ctb1",
				CreationTimestamp: metav1.Time{Time: now.Add(-2 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "ctb2",
				CreationTimestamp: metav1.Time{Time: now.Add(-30 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "ctb3",
				CreationTimestamp: metav1.Time{Time: now.Add(-200 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ctb4",
			},
		},
	}

	tests := []struct {
		name      string
		command   *cobra.Command
		args      []string
		wantError bool
	}{
		{
			name:      "No ClusterTriggerBindings",
			command:   command(t, []*v1beta1.ClusterTriggerBinding{}, now),
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
		{
			name:      "List ClusterTriggerBindings without headers",
			command:   command(t, ctbs, now),
			args:      []string{"list", "--no-headers"},
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

func command(t *testing.T, ctbs []*v1beta1.ClusterTriggerBinding, now time.Time) *cobra.Command {
	// fake clock advanced by 1 hour
	clock := clockwork.NewFakeClockAt(now)

	cs := test.SeedTestResources(t, triggertest.Resources{ClusterTriggerBindings: ctbs})
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
	tdc := testDynamic.Options{}
	var utts []runtime.Object
	for _, ctb := range ctbs {
		utts = append(utts, cb.UnstructuredV1beta1CTB(ctb, "v1beta1"))
	}
	dc, err := tdc.Client(utts...)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Dynamic: dc, Tekton: cs.Pipeline, Clock: clock}
	return Command(p)
}
