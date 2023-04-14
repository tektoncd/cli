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

package clustertriggerbinding

import (
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggertest "github.com/tektoncd/triggers/test"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTrigger_GetAllClusterTriggerBinding(t *testing.T) {
	clock := test.FakeClock()

	ctb := []*v1beta1.ClusterTriggerBinding{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "ctb1",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	ctb2 := []*v1beta1.ClusterTriggerBinding{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "ctb1",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "ctb2",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{ClusterTriggerBindings: ctb})
	cs2 := test.SeedTestResources(t, triggertest.Resources{ClusterTriggerBindings: ctb2})

	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1CTB(ctb[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs2.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1CTB(ctb2[0], "v1beta1"),
		cb.UnstructuredV1beta1CTB(ctb2[1], "v1beta1"),
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
			name:   "Single ClusterTriggerBinding",
			params: p,
			want:   []string{"ctb1"},
		},
		{
			name:   "Multi ClusterTriggerBinding",
			params: p2,
			want:   []string{"ctb1", "ctb2"},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			cs, err := tp.params.Clients()
			if err != nil {
				t.Errorf("unexpected Error, not able to get clients")
			}
			got, err := GetAllClusterTriggerBindingNames(cs)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestClusterTriggerBinding_List(t *testing.T) {
	clock := test.FakeClock()

	ctb := []*v1beta1.ClusterTriggerBinding{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "ctb1",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	ctb2 := []*v1beta1.ClusterTriggerBinding{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "ctb1",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "ctb2",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{ClusterTriggerBindings: ctb})
	cs2 := test.SeedTestResources(t, triggertest.Resources{ClusterTriggerBindings: ctb2})

	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1CTB(ctb[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs2.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1CTB(ctb2[0], "v1beta1"),
		cb.UnstructuredV1beta1CTB(ctb2[1], "v1beta1"),
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
			name:   "Single ClusterTriggerBinding",
			params: p,
			want:   []string{"ctb1"},
		},
		{
			name:   "Multi ClusterTriggerBinding",
			params: p2,
			want:   []string{"ctb1", "ctb2"},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			cs, err := tp.params.Clients()
			if err != nil {
				t.Errorf("unexpected Error, not able to get clients")
			}
			got, err := List(cs, v1.ListOptions{})
			if err != nil {
				t.Errorf("unexpected Error")
			}

			ctbnames := []string{}
			for _, ctb := range got.Items {
				ctbnames = append(ctbnames, ctb.Name)
			}
			test.AssertOutput(t, tp.want, ctbnames)
		})
	}
}

func TestClusterTriggerBinding_Get(t *testing.T) {
	clock := test.FakeClock()

	ctb := []*v1beta1.ClusterTriggerBinding{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "ctb1",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "ctb2",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{ClusterTriggerBindings: ctb})

	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1CTB(ctb[0], "v1beta1"),
		cb.UnstructuredV1beta1CTB(ctb[1], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Clock: clock, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}
	got, err := Get(c, "ctb2", v1.GetOptions{})
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "ctb2", got.Name)
}
