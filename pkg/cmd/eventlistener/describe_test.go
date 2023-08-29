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
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors/cel"
	triggertest "github.com/tektoncd/triggers/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestEventListenerDescribe_InvalidNamespace(t *testing.T) {
	cs := test.SeedTestResources(t, triggertest.Resources{})
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
	tdc := testDynamic.Options{}
	var utts []runtime.Object
	dc, err := tdc.Client(utts...)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Triggers: cs.Triggers, Dynamic: dc}

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
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
	tdc := testDynamic.Options{}
	var utts []runtime.Object
	dc, err := tdc.Client(utts...)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Triggers: cs.Triggers, Dynamic: dc}

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
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
	tdc := testDynamic.Options{}
	var utts []runtime.Object
	dc, err := tdc.Client(utts...)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Triggers: cs.Triggers, Dynamic: dc}

	eventListener := Command(p)
	out, err := test.ExecuteCommand(eventListener, "desc", "-n", "ns")
	if err == nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestEventListenerDescribe_WithMinRequiredField(t *testing.T) {
	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_OneTriggerWithOneClusterTriggerBinding(t *testing.T) {
	triggerTemplateRef := "tt1"
	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{
					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1beta1",
						},
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1beta1",
							},
						},
					},
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_WithOutputStatusURLAndName(t *testing.T) {
	triggerTemplateRef := "tt1"
	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{
					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1beta1",
						},
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1beta1",
							},
						},
					},
				},
			},
			Status: triggersv1beta1.EventListenerStatus{
				AddressStatus: duckv1beta1.AddressStatus{
					Address: &duckv1beta1.Addressable{
						URL: &apis.URL{
							Scheme: "http",
							Host:   "el-listener.default.svc.cluster.local",
						},
					},
				},
				Configuration: triggersv1beta1.EventListenerConfig{
					GeneratedResourceName: "el-listener",
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_OneTriggerWithOneTriggerBinding(t *testing.T) {
	triggerTemplateRef := "tt1"

	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				ServiceAccountName: "trigger-sa",
				Triggers: []triggersv1beta1.EventListenerTrigger{
					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1beta1",
						},
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
						},
					},
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_OneTriggerWithMultipleTriggerBinding(t *testing.T) {
	triggerTemplateRef := "tt1"
	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{
					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1beta1",
						},
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1beta1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1beta1",
							},
						},
					},
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_OneTriggerWithTriggerBindingName(t *testing.T) {
	bindingval := "somevalue"

	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{
					{
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Name:  "binding",
								Value: &bindingval,
							},
						},
						Template: &triggersv1beta1.EventListenerTemplate{
							Ref:        nil,
							APIVersion: "v1beta1",
							Spec:       nil,
						},
						Name: "tt1",
					},
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_TriggerWithTriggerTemplateRef(t *testing.T) {
	bindingval := "somevalue"
	tempRef := "someref"

	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{
					{
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Name:  "binding",
								Value: &bindingval,
							},
						},
						Template: &triggersv1beta1.EventListenerTemplate{
							Ref:        &tempRef,
							APIVersion: "v1beta1",
						},
						Name: "tt1",
					},
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_TriggerWithTriggerTemplateRefTriggerRef(t *testing.T) {
	bindingval := "somevalue"
	tempRef := "someref"

	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{
					{
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Name:  "binding",
								Value: &bindingval,
							},
						},
						Template: &triggersv1beta1.EventListenerTemplate{
							Ref:        &tempRef,
							APIVersion: "v1beta1",
						},
						Name:               "tt1",
						TriggerRef:         "triggeref",
						ServiceAccountName: "test-sa",
						Interceptors: []*triggersv1beta1.EventInterceptor{
							{
								Webhook: &triggersv1beta1.WebhookInterceptor{
									ObjectRef: &corev1.ObjectReference{
										Kind:       "Service",
										Name:       "testwebhook",
										APIVersion: "v1",
										Namespace:  "ns",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_OneTriggerWithEmptyTriggerBinding(t *testing.T) {
	triggerTemplateRef := "tt1"

	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{
					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1beta1",
						},
					},
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_MultipleTriggers(t *testing.T) {
	triggerTemplateRef1 := "tt1"
	triggerTemplateRef2 := "tt2"

	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{
					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef1,
							APIVersion: "v1beta1",
						},
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1beta1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1beta1",
							},
						},
					},
					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef2,
							APIVersion: "v1beta1",
						},
						ServiceAccountName: "sa1",
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1beta1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1beta1",
							},
						},
					},
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_WithWebhookInterceptors(t *testing.T) {
	triggerTemplateRef := "tt1"
	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Status: triggersv1beta1.EventListenerStatus{
				AddressStatus: duckv1beta1.AddressStatus{
					Address: &duckv1beta1.Addressable{
						URL: &apis.URL{
							Scheme: "http",
							Host:   "el-listener.default.svc.cluster.local",
						},
					},
				},
				Configuration: triggersv1beta1.EventListenerConfig{
					GeneratedResourceName: "el-listener",
				},
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{

					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1beta1",
						},
						Name: "foo-trig",
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1beta1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1beta1",
							},
						},
						Interceptors: []*triggersv1beta1.TriggerInterceptor{
							{
								Webhook: &triggersv1beta1.WebhookInterceptor{
									ObjectRef: &corev1.ObjectReference{
										Kind:       "Service",
										Name:       "webhookTest",
										Namespace:  "namespace",
										APIVersion: "v1",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_WithWebhookInterceptorsWithParams(t *testing.T) {
	triggerTemplateRef := "tt1"
	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{

					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1beta1",
						},
						Name: "foo-trig",
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1beta1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1beta1",
							},
						},
						Interceptors: []*triggersv1beta1.TriggerInterceptor{
							{
								Webhook: &triggersv1beta1.WebhookInterceptor{
									ObjectRef: &corev1.ObjectReference{
										Kind:       "Service",
										Name:       "foo",
										Namespace:  "namespace",
										APIVersion: "v1",
									},
									Header: []v1beta1.Param{
										{
											Name: "header",
											Value: v1beta1.ParamValue{
												Type:     "array",
												ArrayVal: []string{"value"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_MultipleTriggerWithTriggerRefAndTriggerSpec(t *testing.T) {
	triggerTemplateRef := "tt1"
	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{
					{
						TriggerRef: "test-ref",
					},
					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1beta1",
						},
						Name: "foo-trig",

						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1beta1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1beta1",
							},
						},
						Interceptors: []*triggersv1beta1.TriggerInterceptor{
							{
								Webhook: &triggersv1beta1.WebhookInterceptor{
									ObjectRef: &corev1.ObjectReference{
										Kind:       "Service",
										Name:       "foo",
										Namespace:  "namespace",
										APIVersion: "v1",
									},
									Header: []v1beta1.Param{
										{
											Name: "header",
											Value: v1beta1.ParamValue{
												Type:     "array",
												ArrayVal: []string{"value"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_WithCELInterceptors(t *testing.T) {
	triggerTemplateRef := "tt1"
	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{

					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1beta1",
						},
						Name: "foo-trig",
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1beta1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1beta1",
							},
						},
						Interceptors: []*triggersv1beta1.TriggerInterceptor{
							{
								Ref: triggersv1beta1.InterceptorRef{
									Name: "cel",
									Kind: triggersv1beta1.ClusterInterceptorKind,
								},
								Params: []triggersv1beta1.InterceptorParams{
									{
										Name:  "filter",
										Value: triggertest.ToV1JSON(t, `body.value == 'test'`),
									},
									{
										Name: "overlays",
										Value: triggertest.ToV1JSON(t, []cel.Overlay{{
											Key:        "value",
											Expression: "'testing'",
										}}),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_WithMultipleBindingAndInterceptors(t *testing.T) {
	triggerTemplateRef1 := "tt1"
	triggerTemplateRef2 := "tt2"
	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Resources: triggersv1beta1.Resources{
					KubernetesResource: &triggersv1beta1.KubernetesResource{
						ServiceType: "ClusterIP",
					},
				},
				Triggers: []triggersv1beta1.EventListenerTrigger{

					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef1,
							APIVersion: "v1beta1",
						},
						Name: "foo-trig",
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1beta1",
							},
							{
								Ref:        "",
								Kind:       "TriggerBinding",
								APIVersion: "v1beta1",
							},
						},
						Interceptors: []*triggersv1beta1.TriggerInterceptor{
							{
								Ref: triggersv1beta1.InterceptorRef{
									Name: "cel",
									Kind: triggersv1beta1.ClusterInterceptorKind,
								},
								Params: []triggersv1beta1.InterceptorParams{
									{
										Name:  "filter",
										Value: triggertest.ToV1JSON(t, `body.value == 'test'`),
									},
									{
										Name: "overlays",
										Value: triggertest.ToV1JSON(t, []cel.Overlay{{
											Key:        "value",
											Expression: "'testing'",
										}}),
									},
								},
							},
						},
					},
					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef2,
							APIVersion: "v1beta1",
						},
						ServiceAccountName: "sa1",
						Name:               "foo-trig",
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Ref:        "tb4",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb5",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1beta1",
							},
						},
						Interceptors: []*triggersv1beta1.TriggerInterceptor{
							{
								Webhook: &triggersv1beta1.WebhookInterceptor{
									ObjectRef: &corev1.ObjectReference{
										Kind:       "Service",
										Name:       "webhookTest",
										Namespace:  "namespace",
										APIVersion: "v1",
									},
								},
							},
							{
								Ref: triggersv1beta1.InterceptorRef{
									Name: "cel",
									Kind: triggersv1beta1.ClusterInterceptorKind,
								},
								Params: []triggersv1beta1.InterceptorParams{
									{
										Name:  "filter",
										Value: triggertest.ToV1JSON(t, `body.value == 'test'`),
									},
									{
										Name: "overlays",
										Value: triggertest.ToV1JSON(t, []cel.Overlay{{
											Key:        "value",
											Expression: "'testing'",
										}}),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_OutputYAMLWithMultipleBindingAndInterceptors(t *testing.T) {
	triggerTemplateRef1 := "tt1"
	triggerTemplateRef2 := "tt2"
	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{

					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef1,
							APIVersion: "v1beta1",
						},
						Name: "foo-trig",
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1beta1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1beta1",
							},
						},
						Interceptors: []*triggersv1beta1.TriggerInterceptor{
							{
								Ref: triggersv1beta1.InterceptorRef{
									Name: "cel",
									Kind: triggersv1beta1.ClusterInterceptorKind,
								},
								Params: []triggersv1beta1.InterceptorParams{
									{
										Name:  "filter",
										Value: triggertest.ToV1JSON(t, `body.value == 'test'`),
									},
									{
										Name: "overlays",
										Value: triggertest.ToV1JSON(t, []cel.Overlay{{
											Key:        "value",
											Expression: "'testing'",
										}}),
									},
								},
							},
						},
					},
					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef2,
							APIVersion: "v1beta1",
						},
						ServiceAccountName: "sa1",
						Name:               "foo-trig",
						Bindings: []*triggersv1beta1.EventListenerBinding{
							{
								Ref:        "tb4",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb5",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1beta1",
							},
						},
						Interceptors: []*triggersv1beta1.TriggerInterceptor{
							{
								Webhook: &triggersv1beta1.WebhookInterceptor{
									ObjectRef: &corev1.ObjectReference{
										Kind:       "Service",
										Name:       "webhookTest",
										Namespace:  "namespace",
										APIVersion: "v1",
									},
								},
							},
							{
								Ref: triggersv1beta1.InterceptorRef{
									Name: "cel",
									Kind: triggersv1beta1.ClusterInterceptorKind,
								},
								Params: []triggersv1beta1.InterceptorParams{
									{
										Name:  "filter",
										Value: triggertest.ToV1JSON(t, `body.value == 'test'`),
									},
									{
										Name: "overlays",
										Value: triggertest.ToV1JSON(t, []cel.Overlay{{
											Key:        "value",
											Expression: "'testing'",
										}}),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els, Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
	tdc := testDynamic.Options{}
	var utts []runtime.Object
	for _, el := range els {
		utts = append(utts, cb.UnstructuredV1beta1EL(el, "v1beta1"))
	}
	dc, err := tdc.Client(utts...)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Triggers: cs.Triggers, Dynamic: dc}

	eventListener := Command(p)
	out, err := test.ExecuteCommand(eventListener, "desc", "el1", "-n", "ns", "-o", "yaml")
	if err != nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestEventListenerDescribe_WithOutputStatusURL(t *testing.T) {
	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},

			Status: triggersv1beta1.EventListenerStatus{
				AddressStatus: duckv1beta1.AddressStatus{
					Address: &duckv1beta1.Addressable{
						URL: &apis.URL{
							Scheme: "http",
							Host:   "el-listener.default.svc.cluster.local",
						},
					},
				},
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els})

	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
	tdc := testDynamic.Options{}
	var utts []runtime.Object
	for _, el := range els {
		utts = append(utts, cb.UnstructuredV1beta1EL(el, "v1beta1"))
	}
	dc, err := tdc.Client(utts...)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Triggers: cs.Triggers, Dynamic: dc}

	eventListener := Command(p)
	out, err := test.ExecuteCommand(eventListener, "desc", "el1", "-o", "url", "-n", "ns")
	if err != nil {
		t.Errorf("Error")
	}
	test.AssertOutput(t, "http://el-listener.default.svc.cluster.local\n", out)
}

func TestEventListenerDescribe_OutputStatusURL_WithNoURL(t *testing.T) {
	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els})

	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
	tdc := testDynamic.Options{}
	var utts []runtime.Object
	for _, el := range els {
		utts = append(utts, cb.UnstructuredV1beta1EL(el, "v1beta1"))
	}
	dc, err := tdc.Client(utts...)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Triggers: cs.Triggers, Dynamic: dc}

	eventListener := Command(p)
	out, err := test.ExecuteCommand(eventListener, "desc", "el1", "-o", "url", "-n", "ns")
	if err == nil {
		t.Errorf("Error")
	}

	test.AssertOutput(t, "Error: "+err.Error()+"\n", out)
}

func TestEventListenerDescribe_AutoSelect(t *testing.T) {
	triggerTemplateRef := "tt1"
	els := []*triggersv1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: triggersv1beta1.EventListenerSpec{
				Triggers: []triggersv1beta1.EventListenerTrigger{
					{
						Template: &triggersv1beta1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1beta1",
						},
					},
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func executeEventListenerCommand(t *testing.T, els []*triggersv1beta1.EventListener) {
	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els, Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})

	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
	tdc := testDynamic.Options{}
	var utts []runtime.Object
	for _, el := range els {
		utts = append(utts, cb.UnstructuredV1beta1EL(el, "v1beta1"))
	}
	dc, err := tdc.Client(utts...)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Triggers: cs.Triggers, Dynamic: dc}

	eventListener := Command(p)
	out, err := test.ExecuteCommand(eventListener, "desc", "el1", "-n", "ns")
	if err != nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}
