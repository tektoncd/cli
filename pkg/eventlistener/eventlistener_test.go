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

package eventlistener

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

func TestEventListener_GetAllEventListenerNames(t *testing.T) {
	clock := test.FakeClock()

	els := []*v1beta1.EventListener{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "el1",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	els2 := []*v1beta1.EventListener{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "el1",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "el2",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els, Namespaces: []*corev1.Namespace{{
		ObjectMeta: v1.ObjectMeta{
			Name: "ns",
		},
	}}})
	cs2 := test.SeedTestResources(t, triggertest.Resources{EventListeners: els2, Namespaces: []*corev1.Namespace{{
		ObjectMeta: v1.ObjectMeta{
			Name: "ns",
		},
	}}})

	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1EL(els[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs2.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1EL(els2[0], "v1beta1"),
		cb.UnstructuredV1beta1EL(els2[1], "v1beta1"),
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
			name:   "Single EventListener",
			params: p,
			want:   []string{"el1"},
		},
		{
			name:   "Multi EventListeners",
			params: p2,
			want:   []string{"el1", "el2"},
		},
		{
			name:   "Unknown namespace",
			params: p3,
			want:   []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			c, err := tp.params.Clients()
			if err != nil {
				t.Errorf("unexpected Error, not able to get clients")
			}
			got, err := GetAllEventListenerNames(c, tp.params.Namespace())
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestEventListener_List(t *testing.T) {
	clock := test.FakeClock()

	els := []*v1beta1.EventListener{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "el1",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	els2 := []*v1beta1.EventListener{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "el1",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "el2",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els, Namespaces: []*corev1.Namespace{{
		ObjectMeta: v1.ObjectMeta{
			Name: "ns",
		},
	}}})
	cs2 := test.SeedTestResources(t, triggertest.Resources{EventListeners: els2, Namespaces: []*corev1.Namespace{{
		ObjectMeta: v1.ObjectMeta{
			Name: "ns",
		},
	}}})
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1EL(els[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs2.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1EL(els2[0], "v1beta1"),
		cb.UnstructuredV1beta1EL(els2[1], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Clock: clock, Dynamic: dc}
	p2 := &test.Params{Triggers: cs2.Triggers, Kube: cs2.Kube, Clock: clock, Dynamic: dc2}

	testParams := []struct {
		name   string
		params *test.Params
		want   []string
	}{
		{
			name:   "Single EventListener",
			params: p,
			want:   []string{"el1"},
		},
		{
			name:   "Multi EventListeners",
			params: p2,
			want:   []string{"el1", "el2"},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			c, err := tp.params.Clients()
			if err != nil {
				t.Errorf("unexpected Error, not able to get clients")
			}
			got, err := List(c, v1.ListOptions{}, "ns")
			if err != nil {
				t.Errorf("unexpected Error")
			}
			elnames := []string{}
			for _, el := range got.Items {
				elnames = append(elnames, el.Name)
			}
			test.AssertOutput(t, tp.want, elnames)
		})
	}
}

func TestEventListener_Get(t *testing.T) {
	clock := test.FakeClock()

	els := []*v1beta1.EventListener{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "el1",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "el2",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els, Namespaces: []*corev1.Namespace{{
		ObjectMeta: v1.ObjectMeta{
			Name: "ns",
		},
	}}})

	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1EL(els[0], "v1beta1"),
		cb.UnstructuredV1beta1EL(els[1], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Clock: clock, Dynamic: dc}

	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}
	got, err := Get(c, "el1", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "el1", got.Name)
}

func TestEventListener_GetError(t *testing.T) {
	clock := test.FakeClock()

	els := []*v1beta1.EventListener{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "el1",
				Namespace:         "ns",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els, Namespaces: []*corev1.Namespace{{
		ObjectMeta: v1.ObjectMeta{
			Name: "ns",
		},
	}}})

	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1EL(els[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Clock: clock, Dynamic: dc}

	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}
	_, err = Get(c, "el2", v1.GetOptions{}, "ns")
	if err == nil {
		t.Errorf("expected Error but found nil")
	}
	test.AssertOutput(t, err.Error(), "failed to get EventListener el2: eventlisteners.triggers.tekton.dev \"el2\" not found")
}
