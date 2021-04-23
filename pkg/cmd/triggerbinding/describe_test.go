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

package triggerbinding

import (
	"fmt"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggertest "github.com/tektoncd/triggers/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTriggerBindingDescribe_Invalid_Namespace(t *testing.T) {
	cs := test.SeedTestResources(t, triggertest.Resources{})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerBinding := Command(p)
	out, err := test.ExecuteCommand(triggerBinding, "desc", "bar", "-n", "invalid")
	if err == nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTriggerBindingDescribe_NonExistedName(t *testing.T) {
	cs := test.SeedTestResources(t, triggertest.Resources{Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerBinding := Command(p)
	out, err := test.ExecuteCommand(triggerBinding, "desc", "bar", "-n", "ns")
	if err == nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTriggerBindingDescribe_Empty(t *testing.T) {
	cs := test.SeedTestResources(t, triggertest.Resources{Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerBinding := Command(p)
	out, err := test.ExecuteCommand(triggerBinding, "desc", "-n", "ns")
	if err == nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTriggerBindingDescribe_NoParams(t *testing.T) {
	tbs := []*v1alpha1.TriggerBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tb1",
				Namespace: "ns",
			},
		},
	}
	cs := test.SeedTestResources(t, triggertest.Resources{TriggerBindings: tbs, Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerBinding := Command(p)
	out, err := test.ExecuteCommand(triggerBinding, "desc", "tb1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTriggerBindingDescribe_WithParams(t *testing.T) {
	tbs := []*v1alpha1.TriggerBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tb1",
				Namespace: "ns",
			},
			Spec: v1alpha1.TriggerBindingSpec{
				Params: []v1alpha1.Param{
					{
						Name:  "key",
						Value: "value",
					},
				},
			},
		},
	}
	cs := test.SeedTestResources(t, triggertest.Resources{TriggerBindings: tbs, Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerBinding := Command(p)
	out, err := test.ExecuteCommand(triggerBinding, "desc", "tb1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTriggerBindingDescribe_WithOutputName(t *testing.T) {
	tbs := []*v1alpha1.TriggerBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tb1",
				Namespace: "ns",
			},
			Spec: v1alpha1.TriggerBindingSpec{
				Params: []v1alpha1.Param{
					{
						Name:  "key",
						Value: "value",
					},
				},
			},
		},
	}
	cs := test.SeedTestResources(t, triggertest.Resources{TriggerBindings: tbs, Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerBinding := Command(p)
	out, err := test.ExecuteCommand(triggerBinding, "desc", "-o", "name", "-n", "ns", "tb1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTriggerBindingDescribe_WithOutputYaml(t *testing.T) {
	tbs := []*v1alpha1.TriggerBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tb1",
				Namespace: "ns",
			},
			Spec: v1alpha1.TriggerBindingSpec{
				Params: []v1alpha1.Param{
					{
						Name:  "key",
						Value: "value",
					},
				},
			},
		},
	}
	cs := test.SeedTestResources(t, triggertest.Resources{TriggerBindings: tbs, Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerBinding := Command(p)
	out, err := test.ExecuteCommand(triggerBinding, "desc", "-o", "yaml", "-n", "ns", "tb1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTriggerBindingDescribe_WithMultipleParams(t *testing.T) {
	tbs := []*v1alpha1.TriggerBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tb1",
				Namespace: "ns",
			},
			Spec: v1alpha1.TriggerBindingSpec{
				Params: []v1alpha1.Param{
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
	cs := test.SeedTestResources(t, triggertest.Resources{TriggerBindings: tbs, Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerBinding := Command(p)
	out, err := test.ExecuteCommand(triggerBinding, "desc", "tb1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTriggerBindingDescribe_AutoSelect(t *testing.T) {
	tbs := []*v1alpha1.TriggerBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tb1",
				Namespace: "ns",
			},
			Spec: v1alpha1.TriggerBindingSpec{
				Params: []v1alpha1.Param{
					{
						Name:  "key",
						Value: "value",
					},
				},
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{TriggerBindings: tbs, Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerBinding := Command(p)
	out, err := test.ExecuteCommand(triggerBinding, "desc", "-n", "ns")
	if err != nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}
