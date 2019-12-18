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
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggertest "github.com/tektoncd/triggers/test"
	tb "github.com/tektoncd/triggers/test/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTriggerTemplateList_Invalid_Namespace(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		},
	}
	cs := test.SeedTestResources(t, triggertest.Resources{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Triggers: cs.Triggers}

	triggertemplate := Command(p)
	_, err := test.ExecuteCommand(triggertemplate, "list", "-n", "foo")
	if err == nil {
		t.Errorf("Expected error")
	}
	test.AssertOutput(t, "namespaces \"foo\" not found", err.Error())
}

func TestTriggerTemplateList_Empty(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
	}
	cs := test.SeedTestResources(t, triggertest.Resources{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Triggers: cs.Triggers}

	triggertemplate := Command(p)
	output, err := test.ExecuteCommand(triggertemplate, "list", "-n", "foo")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, "No triggertemplates found\n", output)
}

func TestTriggerTemplateList_Multiple(t *testing.T) {
	clock := clockwork.NewFakeClock()
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
	}
	tt := []*v1alpha1.TriggerTemplate{
		tb.TriggerTemplate("tt1", "foo", cb.TriggerTemplateCreationTime(clock.Now().Add(-2*time.Minute))),
		tb.TriggerTemplate("tt2", "foo", cb.TriggerTemplateCreationTime(clock.Now().Add(-30*time.Second))),
		tb.TriggerTemplate("tt3", "foo", cb.TriggerTemplateCreationTime(clock.Now().Add(-200*time.Hour))),
		tb.TriggerTemplate("tt4", "foo"),
	}
	cs := test.SeedTestResources(t, triggertest.Resources{Namespaces: ns, TriggerTemplates: tt})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Triggers: cs.Triggers, Clock: clock}

	triggertemplate := Command(p)
	output, err := test.ExecuteCommand(triggertemplate, "list", "-n", "foo")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := []string{
		"NAME   AGE",
		"tt1    2 minutes ago",
		"tt2    30 seconds ago",
		"tt3    1 week ago",
		"tt4    ---",
		"",
	}
	text := strings.Join(expected, "\n")
	test.AssertOutput(t, text, output)
}

func TestTriggerTemplateList_Output(t *testing.T) {
	clock := clockwork.NewFakeClock()
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
	}
	tt := []*v1alpha1.TriggerTemplate{
		tb.TriggerTemplate("tt1", "foo", cb.TriggerTemplateCreationTime(clock.Now().Add(-2*time.Minute))),
		tb.TriggerTemplate("tt2", "foo", cb.TriggerTemplateCreationTime(clock.Now().Add(-30*time.Second))),
		tb.TriggerTemplate("tt3", "foo", cb.TriggerTemplateCreationTime(clock.Now().Add(-200*time.Hour))),
		tb.TriggerTemplate("tt4", "foo"),
	}
	cs := test.SeedTestResources(t, triggertest.Resources{Namespaces: ns, TriggerTemplates: tt})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Triggers: cs.Triggers, Clock: clock}

	triggertemplate := Command(p)
	output, err := test.ExecuteCommand(triggertemplate, "list", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := []string{
		"tt1",
		"tt2",
		"tt3",
		"tt4",
		"",
	}
	text := strings.Join(expected, "\n")
	test.AssertOutput(t, text, output)
}
