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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListEventListener(t *testing.T) {
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random",
			},
		},
	}

	elStatusTrue := tb.EventListenerStatus(tb.EventListenerCondition("", "True", "", ""))
	elStatusFalse := tb.EventListenerStatus(tb.EventListenerCondition("", "False", "", ""))
	els := []*v1alpha1.EventListener{
		tb.EventListener("tb0", "bar", cb.EventListenerCreationTime(now.Add(-2*time.Minute)), tb.EventListenerStatus(
			tb.EventListenerAddress("tb0-listener.default.svc.cluster.local")), elStatusTrue),
		tb.EventListener("tb1", "foo", cb.EventListenerCreationTime(now.Add(-2*time.Minute)), tb.EventListenerStatus(
			tb.EventListenerAddress("tb1-listener.default.svc.cluster.local")), elStatusTrue),
		tb.EventListener("tb2", "foo", cb.EventListenerCreationTime(now.Add(-30*time.Second)), tb.EventListenerStatus(
			tb.EventListenerAddress("tb2-listener.default.svc.cluster.local")), elStatusTrue),
		tb.EventListener("tb3", "foo", cb.EventListenerCreationTime(now.Add(-200*time.Hour)), tb.EventListenerStatus(
			tb.EventListenerAddress("tb3-listener.default.svc.cluster.local")), elStatusTrue),
		tb.EventListener("tb4", "foo"),
		tb.EventListener("tb5", "foo", cb.EventListenerCreationTime(now.Add(-10*time.Second)), elStatusFalse),
	}

	tests := []struct {
		name      string
		command   *cobra.Command
		args      []string
		wantError bool
	}{
		{
			name:      "Invalid namespace",
			command:   command(t, els, now, ns),
			args:      []string{"list", "-n", "default"},
			wantError: true,
		},
		{
			name:      "No EventListener",
			command:   command(t, els, now, ns),
			args:      []string{"list", "-n", "random"},
			wantError: false,
		},
		{
			name:      "Multiple EventListener",
			command:   command(t, els, now, ns),
			args:      []string{"list", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "Multiple EventListener with output format",
			command:   command(t, els, now, ns),
			args:      []string{"list", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
		{
			name:      "EventListeners from all-namespaces",
			command:   command(t, els, now, ns),
			args:      []string{"list", "--all-namespaces"},
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

func command(t *testing.T, els []*v1alpha1.EventListener, now time.Time, ns []*corev1.Namespace) *cobra.Command {
	// fake clock advanced by 1 hour
	clock := clockwork.NewFakeClockAt(now)
	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Triggers: cs.Triggers}
	return Command(p)
}

func TestEventListenersList_empty(t *testing.T) {
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

	els := []*v1alpha1.EventListener{}
	listEls := command(t, els, now, ns)

	out, _ := test.ExecuteCommand(listEls, "list", "--all-namespaces")
	test.AssertOutput(t, emptyMsg+"\n", out)
}
