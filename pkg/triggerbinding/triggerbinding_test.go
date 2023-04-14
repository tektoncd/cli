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

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggertest "github.com/tektoncd/triggers/test"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTrigger_GetAllTriggerBinding(t *testing.T) {
	clock := test.FakeClock()

	tb := []*v1beta1.TriggerBinding{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "tb1",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	tb2 := []*v1beta1.TriggerBinding{
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

	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"triggerbinding"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TB(tb[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs2.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"triggerbinding"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1TB(tb2[0], "v1beta1"),
		cb.UnstructuredV1beta1TB(tb2[1], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Clock: clock, Dynamic: dc}
	p2 := &test.Params{Triggers: cs2.Triggers, Kube: cs2.Kube, Clock: clock, Dynamic: dc2}
	p3 := &test.Params{Triggers: cs2.Triggers, Kube: cs2.Kube, Clock: clock, Dynamic: dc2}
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
			cs, err := tp.params.Clients()
			if err != nil {
				t.Errorf("unexpected Error, not able to get clients")
			}
			got, err := GetAllTriggerBindingNames(cs, tp.params.Namespace())
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestTriggerBinding_List(t *testing.T) {
	clock := test.FakeClock()

	tb := []*v1beta1.TriggerBinding{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "tb1",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	tb2 := []*v1beta1.TriggerBinding{
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

	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"triggerbinding"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TB(tb[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs2.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"triggerbinding"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1TB(tb2[0], "v1beta1"),
		cb.UnstructuredV1beta1TB(tb2[1], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Clock: clock, Dynamic: dc}
	p2 := &test.Params{Triggers: cs2.Triggers, Kube: cs2.Kube, Clock: clock, Dynamic: dc2}
	p3 := &test.Params{Triggers: cs2.Triggers, Kube: cs2.Kube, Clock: clock, Dynamic: dc2}
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
			cs, err := tp.params.Clients()
			if err != nil {
				t.Errorf("unexpected Error, not able to get clients")
			}
			got, err := List(cs, v1.ListOptions{}, "ns")
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

func TestTriggerBinding_Get(t *testing.T) {
	clock := test.FakeClock()

	tb := []*v1beta1.TriggerBinding{
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

	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"triggerbinding"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TB(tb[0], "v1beta1"),
		cb.UnstructuredV1beta1TB(tb[1], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Clock: clock, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}
	got, err := Get(c, "tb2", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "tb2", got.Name)
}
