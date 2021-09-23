// Copyright © 2021 The Tekton Authors.
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
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	v1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggertest "github.com/tektoncd/triggers/test"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTrigger_GetAllTriggerBinding(t *testing.T) {
	clock := clockwork.NewFakeClock()

	tb := []*v1alpha1.TriggerBinding{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "tb1",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	tb2 := []*v1alpha1.TriggerBinding{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "tb1",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "tb2",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{TriggerBindings: tb, Namespaces: []*corev1.Namespace{{
		ObjectMeta: v1.ObjectMeta{
			Name: "ns",
		},
	}}})

	cs2 := test.SeedTestResources(t, triggertest.Resources{TriggerBindings: tb2, Namespaces: []*corev1.Namespace{{
		ObjectMeta: v1.ObjectMeta{
			Name: "ns",
		},
	}}})

	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}
	p2 := &test.Params{Triggers: cs2.Triggers, Kube: cs2.Kube}
	p3 := &test.Params{Triggers: cs2.Triggers, Kube: cs2.Kube}
	p3.SetNamespace("unknown")

	testParams := []struct {
		name   string
		params *test.Params
		want   []string
	}{
		{
			name:   "Single TriggerBinding",
			params: p,
			want:   []string{"tb1"},
		},
		{
			name:   "Multi TriggerBinding",
			params: p2,
			want:   []string{"tb1", "tb2"},
		},
		{
			name:   "Unknown namespace",
			params: p3,
			want:   []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllTriggerBindingNames(tp.params.Triggers, tp.params.Namespace())
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestTriggerBinding_List(t *testing.T) {
	clock := clockwork.NewFakeClock()

	tb := []*v1alpha1.TriggerBinding{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "tb1",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	tb2 := []*v1alpha1.TriggerBinding{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "tb1",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "tb2",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{TriggerBindings: tb, Namespaces: []*corev1.Namespace{{
		ObjectMeta: v1.ObjectMeta{
			Name: "ns",
		},
	}}})

	cs2 := test.SeedTestResources(t, triggertest.Resources{TriggerBindings: tb2, Namespaces: []*corev1.Namespace{{
		ObjectMeta: v1.ObjectMeta{
			Name: "ns",
		},
	}}})

	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}
	p2 := &test.Params{Triggers: cs2.Triggers, Kube: cs2.Kube}
	p3 := &test.Params{Triggers: cs2.Triggers, Kube: cs2.Kube}
	p3.SetNamespace("unknown")

	testParams := []struct {
		name   string
		params *test.Params
		want   []string
	}{
		{
			name:   "Single TriggerBinding",
			params: p,
			want:   []string{"tb1"},
		},
		{
			name:   "Multi TriggerBinding",
			params: p2,
			want:   []string{"tb1", "tb2"},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := List(tp.params.Triggers, "ns")
			if err != nil {
				t.Errorf("unexpected Error")
			}

			tbnames := []string{}
			for _, tb := range got.Items {
				tbnames = append(tbnames, tb.Name)
			}
			test.AssertOutput(t, tp.want, tbnames)
		})
	}
}
