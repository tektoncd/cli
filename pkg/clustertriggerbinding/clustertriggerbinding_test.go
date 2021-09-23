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

// TODO: properly move to v1beta1
import (
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	v1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggertest "github.com/tektoncd/triggers/test"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTrigger_GetAllClusterTriggerBinding(t *testing.T) {
	clock := clockwork.NewFakeClock()

	ctb := []*v1alpha1.ClusterTriggerBinding{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "ctb1",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	ctb2 := []*v1alpha1.ClusterTriggerBinding{
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

	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}
	p2 := &test.Params{Triggers: cs2.Triggers, Kube: cs2.Kube}

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
			got, err := GetAllClusterTriggerBindingNames(tp.params.Triggers, tp.params.Namespace())
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestClusterTriggerBinding_List(t *testing.T) {
	clock := clockwork.NewFakeClock()

	ctb := []*v1alpha1.ClusterTriggerBinding{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              "ctb1",
				CreationTimestamp: v1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	ctb2 := []*v1alpha1.ClusterTriggerBinding{
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

	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}
	p2 := &test.Params{Triggers: cs2.Triggers, Kube: cs2.Kube}

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
