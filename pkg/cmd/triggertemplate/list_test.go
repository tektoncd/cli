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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggertest "github.com/tektoncd/triggers/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestListTriggerTemplate(t *testing.T) {
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "bar",
			},
		},
	}

	tts := []*v1beta1.TriggerTemplate{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tt1",
				Namespace:         "foo",
				CreationTimestamp: metav1.Time{Time: now.Add(-2 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tt2",
				Namespace:         "foo",
				CreationTimestamp: metav1.Time{Time: now.Add(-30 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tt3",
				Namespace:         "foo",
				CreationTimestamp: metav1.Time{Time: now.Add(-200 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tt4",
				Namespace: "foo",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tt5",
				Namespace: "bar",
			},
		},
	}

	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "Invalid namespace",
			args:      []string{"list", "-n", "default"},
			wantError: true,
		},
		{
			name:      "No TriggerTemplate",
			args:      []string{"list", "-n", "random"},
			wantError: false,
		},
		{
			name:      "Multiple TriggerTemplates",
			args:      []string{"list", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "Multiple TriggerTemplates with output format",
			args:      []string{"list", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
		{
			name:      "TriggerTemplates from all namespaces",
			args:      []string{"list", "--all-namespaces"},
			wantError: false,
		},
		{
			name:      "List TriggerTemplates without headers",
			args:      []string{"list", "--no-headers"},
			wantError: false,
		},
		{
			name:      "List TriggerTemplates from all namespaces without headers",
			args:      []string{"list", "--no-headers", "--all-namespaces"},
			wantError: false,
		},
	}
	p := command(t, tts, now, ns)

	for _, td := range tests {
		t.Run(td.name, func(t *testing.T) {
			got, err := test.ExecuteCommand(Command(p), td.args...)

			if err != nil && !td.wantError {
				t.Errorf("Unexpected error: %v", err)
			}
			golden.Assert(t, got, strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
		})
	}
}

func TestTriggerTemplateList_empty(t *testing.T) {
	now := time.Now()

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "bar",
			},
		},
	}

	tt := []*v1beta1.TriggerTemplate{}
	listtt := command(t, tt, now, ns)

	out, _ := test.ExecuteCommand(Command(listtt), "list", "--all-namespaces")
	test.AssertOutput(t, emptyMsg+"\n", out)
}

func command(t *testing.T, tts []*v1beta1.TriggerTemplate, now time.Time, ns []*corev1.Namespace) *test.Params {
	// fake clock advanced by 1 hour
	clock := clockwork.NewFakeClockAt(now)

	cs := test.SeedTestResources(t, triggertest.Resources{TriggerTemplates: tts, Namespaces: ns})
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"triggertemplate"})
	tdc := testDynamic.Options{}
	var utts []runtime.Object
	for _, tt := range tts {
		utts = append(utts, cb.UnstructuredV1beta1TT(tt, "v1beta1"))
	}
	dc, err := tdc.Client(utts...)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	return &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Triggers: cs.Triggers, Dynamic: dc}
}
