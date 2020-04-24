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

package eventlistener

import (
	"fmt"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggertest "github.com/tektoncd/triggers/test"
	el "github.com/tektoncd/triggers/test/builder"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEventListenerDescribe_InvalidNamespace(t *testing.T) {
	cs := test.SeedTestResources(t, triggertest.Resources{})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	eventListener := Command(p)
	out, err := test.ExecuteCommand(eventListener, "desc", "bar", "-n", "invalid")
	if err == nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestEventListenerDescribe_NonExistedName(t *testing.T) {
	cs := test.SeedTestResources(t, triggertest.Resources{Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	eventListener := Command(p)
	out, err := test.ExecuteCommand(eventListener, "desc", "bar", "-n", "ns")
	if err == nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestEventListenerDescribe_NoArgProvided(t *testing.T) {
	cs := test.SeedTestResources(t, triggertest.Resources{Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	eventListener := Command(p)
	out, err := test.ExecuteCommand(eventListener, "desc", "-n", "ns")
	if err == nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestEventListenerDescribe_WithMinRequiredField(t *testing.T) {
	els := []*v1alpha1.EventListener{
		el.EventListener("el1", "ns"),
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_OneTriggerWithOneClusterTriggerBinding(t *testing.T) {
	els := []*v1alpha1.EventListener{
		el.EventListener("el1", "ns",
			el.EventListenerSpec(
				el.EventListenerTrigger("tt1", "v1alpha1",
					el.EventListenerTriggerBinding("tb1", "ClusterTriggerBinding", "v1alpha1")))),
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_WithOutputStatusURLAndName(t *testing.T) {
	els := []*v1alpha1.EventListener{
		el.EventListener("el1", "ns",
			el.EventListenerSpec(
				el.EventListenerTrigger("tt1", "v1alpha1",
					el.EventListenerTriggerBinding("tb1", "ClusterTriggerBinding", "v1alpha1"))),
			el.EventListenerStatus(
				el.EventListenerAddress("el-listener.default.svc.cluster.local"),
				el.EventListenerConfig("el-listener"))),
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_OneTriggerWithOneTriggerBinding(t *testing.T) {
	els := []*v1alpha1.EventListener{
		el.EventListener("el1", "ns",
			el.EventListenerSpec(
				el.EventListenerServiceAccount("trigger-sa"),
				el.EventListenerTrigger("tt1", "v1alpha1",
					el.EventListenerTriggerBinding("tb1", "", "")))),
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_OneTriggerWithMultipleTriggerBinding(t *testing.T) {
	els := []*v1alpha1.EventListener{
		el.EventListener("el1", "ns",
			el.EventListenerSpec(
				el.EventListenerTrigger("tt1", "v1alpha1",
					el.EventListenerTriggerBinding("tb1", "", ""),
					el.EventListenerTriggerBinding("tb2", "ClusterTriggerBindingKind", "v1alpha1"),
					el.EventListenerTriggerBinding("tb3", "", "v1alpha1")))),
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_TwoTriggers(t *testing.T) {
	els := []*v1alpha1.EventListener{
		el.EventListener("el1", "ns",
			el.EventListenerSpec(
				el.EventListenerTrigger("tt1", "v1alpha1",
					el.EventListenerTriggerBinding("tb1", "", ""),
					el.EventListenerTriggerBinding("tb2", "ClusterTriggerBindingKind", "v1alpha1"),
					el.EventListenerTriggerBinding("tb3", "", "v1alpha1")),
				el.EventListenerTrigger("tt2", "v1alpha1",
					el.EventListenerTriggerBinding("tb1", "", ""),
					el.EventListenerTriggerBinding("tb2", "ClusterTriggerBindingKind", "v1alpha1"),
					el.EventListenerTriggerBinding("tb3", "", "v1alpha1"),
					el.EventListenerTriggerServiceAccount("sa1", "ns1")))),
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_WithWebhookInterceptors(t *testing.T) {
	els := []*v1alpha1.EventListener{
		el.EventListener("el1", "ns",
			el.EventListenerSpec(
				el.EventListenerTrigger("tt1", "v1alpha1",
					el.EventListenerTriggerBinding("tb1", "", ""),
					el.EventListenerTriggerBinding("tb2", "ClusterTriggerBindingKind", "v1alpha1"),
					el.EventListenerTriggerBinding("tb3", "", "v1alpha1"),
					el.EventListenerTriggerName("foo-trig"),
					el.EventListenerTriggerInterceptor("webhookTest", "v1", "Service", "namespace"))),
			el.EventListenerStatus(
				el.EventListenerAddress("el-listener.default.svc.cluster.local"),
				el.EventListenerConfig("el-listener"))),
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_WithWebhookInterceptorsWithParams(t *testing.T) {
	els := []*v1alpha1.EventListener{
		el.EventListener("el1", "ns",
			el.EventListenerSpec(
				el.EventListenerTrigger("tt1", "v1alpha1",
					el.EventListenerTriggerBinding("tb1", "", ""),
					el.EventListenerTriggerBinding("tb2", "ClusterTriggerBindingKind", "v1alpha1"),
					el.EventListenerTriggerBinding("tb3", "", "v1alpha1"),
					el.EventListenerTriggerName("foo-trig"),
					el.EventListenerTriggerInterceptor("foo", "v1", "Service", "namespace",
						el.EventInterceptorParam("header", "value"))))),
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_WithCELInterceptors(t *testing.T) {
	els := []*v1alpha1.EventListener{
		el.EventListener("el1", "ns",
			el.EventListenerSpec(
				el.EventListenerTrigger("tt1", "v1alpha1",
					el.EventListenerTriggerBinding("tb1", "", ""),
					el.EventListenerTriggerBinding("tb2", "ClusterTriggerBindingKind", "v1alpha1"),
					el.EventListenerTriggerBinding("tb3", "", "v1alpha1"),
					el.EventListenerTriggerName("foo-trig"),
					el.EventListenerCELInterceptor("body.value == 'test'", el.EventListenerCELOverlay("value", "'testing'"))))),
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_WithMultipleBindingAndInterceptors(t *testing.T) {
	els := []*v1alpha1.EventListener{
		el.EventListener("el1", "ns",
			el.EventListenerSpec(
				el.EventListenerTrigger("tt1", "v1alpha1",
					el.EventListenerTriggerBinding("tb1", "", ""),
					el.EventListenerTriggerBinding("tb2", "ClusterTriggerBindingKind", "v1alpha1"),
					el.EventListenerTriggerBinding("tb3", "", "v1alpha1"),
					el.EventListenerTriggerName("foo-trig"),
					el.EventListenerCELInterceptor("body.value == 'test'", el.EventListenerCELOverlay("value", "'testing'"))),
				el.EventListenerTrigger("tt2", "v1alpha1",
					el.EventListenerTriggerBinding("tb4", "", ""),
					el.EventListenerTriggerBinding("tb5", "ClusterTriggerBindingKind", "v1alpha1"),
					el.EventListenerTriggerServiceAccount("sa1", "ns1"),
					el.EventListenerTriggerName("foo-trig"),
					el.EventListenerTriggerInterceptor("webhookTest", "v1", "Service", "namespace"),
					el.EventListenerCELInterceptor("body.value == 'test'", el.EventListenerCELOverlay("value", "'testing'"))))),
	}

	executeEventListenerCommand(t, els)
}

func executeEventListenerCommand(t *testing.T, els []*v1alpha1.EventListener) {
	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els, Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	eventListener := Command(p)
	out, err := test.ExecuteCommand(eventListener, "desc", "el1", "-n", "ns")
	if err != nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}
