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
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggertest "github.com/tektoncd/triggers/test"
	"gotest.tools/v3/golden"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterTriggerBindingDescribe_NonExistedName(t *testing.T) {
	cs := test.SeedTestResources(t, triggertest.Resources{})
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Dynamic: dc}

	clusterTriggerBinding := Command(p)
	out, err := test.ExecuteCommand(clusterTriggerBinding, "desc", "bar")
	if err == nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestClusterTriggerBindingDescribe_Empty(t *testing.T) {
	cs := test.SeedTestResources(t, triggertest.Resources{})
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Dynamic: dc}

	clusterTriggerBinding := Command(p)
	out, err := test.ExecuteCommand(clusterTriggerBinding, "desc")
	if err == nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestClusterTriggerBindingDescribe_NoParams(t *testing.T) {
	ctbs := []*v1beta1.ClusterTriggerBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ctb1",
			},
		},
	}
	cs := test.SeedTestResources(t, triggertest.Resources{ClusterTriggerBindings: ctbs})
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1CTB(ctbs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Dynamic: dc}

	clusterTriggerBinding := Command(p)
	out, err := test.ExecuteCommand(clusterTriggerBinding, "desc", "ctb1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTriggerBindingDescribe_WithParams(t *testing.T) {
	ctbs := []*v1beta1.ClusterTriggerBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ctb1",
			},
			Spec: v1beta1.TriggerBindingSpec{
				Params: []v1beta1.Param{
					{
						Name:  "key",
						Value: "value",
					},
				},
			},
		},
	}
	cs := test.SeedTestResources(t, triggertest.Resources{ClusterTriggerBindings: ctbs})
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1CTB(ctbs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Dynamic: dc}

	clusterTriggerBinding := Command(p)
	out, err := test.ExecuteCommand(clusterTriggerBinding, "desc", "ctb1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestClusterTriggerBindingDescribe_WithOutputName(t *testing.T) {
	ctbs := []*v1beta1.ClusterTriggerBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ctb1",
			},
			Spec: v1beta1.TriggerBindingSpec{
				Params: []v1beta1.Param{
					{
						Name:  "key",
						Value: "value",
					},
				},
			},
		},
	}
	cs := test.SeedTestResources(t, triggertest.Resources{ClusterTriggerBindings: ctbs})
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1CTB(ctbs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Dynamic: dc}

	clusterTriggerBinding := Command(p)
	out, err := test.ExecuteCommand(clusterTriggerBinding, "desc", "-o", "name", "ctb1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestClusterTriggerBindingDescribe_WithOutputYaml(t *testing.T) {
	ctbs := []*v1beta1.ClusterTriggerBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ctb1",
			},
			Spec: v1beta1.TriggerBindingSpec{
				Params: []v1beta1.Param{
					{
						Name:  "key",
						Value: "value",
					},
				},
			},
		},
	}
	cs := test.SeedTestResources(t, triggertest.Resources{ClusterTriggerBindings: ctbs})
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1CTB(ctbs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Dynamic: dc}

	clusterTriggerBinding := Command(p)
	out, err := test.ExecuteCommand(clusterTriggerBinding, "desc", "-o", "yaml", "ctb1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestClusterTriggerBindingDescribe_WithMultipleParams(t *testing.T) {
	ctbs := []*v1beta1.ClusterTriggerBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ctb1",
			},
			Spec: v1beta1.TriggerBindingSpec{
				Params: []v1beta1.Param{
					{
						Name:  "key1",
						Value: "value1",
					},
					{
						Name:  "key2",
						Value: "value2",
					},
					{
						Name:  "key3",
						Value: "value3",
					},
					{
						Name:  "key4",
						Value: "value4",
					},
				},
			},
		},
	}
	cs := test.SeedTestResources(t, triggertest.Resources{ClusterTriggerBindings: ctbs})
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1CTB(ctbs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Dynamic: dc}

	clusterTriggerBinding := Command(p)
	out, err := test.ExecuteCommand(clusterTriggerBinding, "desc", "ctb1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestClusterTriggerBindingDescribe_AutoSelect(t *testing.T) {
	ctbs := []*v1beta1.ClusterTriggerBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ctb1",
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{ClusterTriggerBindings: ctbs})
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"clustertriggerbinding"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1CTB(ctbs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube, Dynamic: dc}

	clusterTriggerBinding := Command(p)
	out, err := test.ExecuteCommand(clusterTriggerBinding, "desc", "-n", "ns")
	if err != nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}
