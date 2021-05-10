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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggertest "github.com/tektoncd/triggers/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	beta1 "knative.dev/pkg/apis/duck/v1beta1"
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
	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{
					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1alpha1",
						},
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1alpha1",
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
	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{
					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1alpha1",
						},
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1alpha1",
							},
						},
					},
				},
			},
			Status: v1alpha1.EventListenerStatus{
				AddressStatus: duckv1alpha1.AddressStatus{
					Address: &duckv1alpha1.Addressable{
						Addressable: beta1.Addressable{
							URL: &apis.URL{
								Scheme: "http",
								Host:   "el-listener.default.svc.cluster.local",
							},
						},
					},
				},
				Configuration: v1alpha1.EventListenerConfig{
					GeneratedResourceName: "el-listener",
				},
			},
		},
	}

	executeEventListenerCommand(t, els)
}

func TestEventListenerDescribe_OneTriggerWithOneTriggerBinding(t *testing.T) {
	triggerTemplateRef := "tt1"

	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				ServiceAccountName: "trigger-sa",
				Triggers: []v1alpha1.EventListenerTrigger{
					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1alpha1",
						},
						Bindings: []*v1alpha1.EventListenerBinding{
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
	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{
					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1alpha1",
						},
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1alpha1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1alpha1",
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

	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{
					{
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Name:  "binding",
								Value: &bindingval,
							},
						},
						Template: &v1alpha1.EventListenerTemplate{
							Ref:        nil,
							APIVersion: "v1alpha1",
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

	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{
					{
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Name:  "binding",
								Value: &bindingval,
							},
						},
						Template: &v1alpha1.EventListenerTemplate{
							Ref:        &tempRef,
							APIVersion: "v1alpha1",
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

	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{
					{
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Name:  "binding",
								Value: &bindingval,
							},
						},
						Template: &v1alpha1.EventListenerTemplate{
							Ref:        &tempRef,
							APIVersion: "v1alpha1",
						},
						Name:               "tt1",
						TriggerRef:         "triggeref",
						ServiceAccountName: "test-sa",
						Interceptors: []*v1alpha1.EventInterceptor{
							{
								Webhook: &v1alpha1.WebhookInterceptor{
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

	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{
					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1alpha1",
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

	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{
					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef1,
							APIVersion: "v1alpha1",
						},
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1alpha1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1alpha1",
							},
						},
					},
					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef2,
							APIVersion: "v1alpha1",
						},
						ServiceAccountName: "sa1",
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1alpha1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1alpha1",
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
	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Status: v1alpha1.EventListenerStatus{
				AddressStatus: duckv1alpha1.AddressStatus{
					Address: &duckv1alpha1.Addressable{
						Addressable: beta1.Addressable{
							URL: &apis.URL{
								Scheme: "http",
								Host:   "el-listener.default.svc.cluster.local",
							},
						},
					},
				},
				Configuration: v1alpha1.EventListenerConfig{
					GeneratedResourceName: "el-listener",
				},
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{

					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1alpha1",
						},
						Name: "foo-trig",
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1alpha1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1alpha1",
							},
						},
						Interceptors: []*v1alpha1.TriggerInterceptor{
							{
								Webhook: &v1alpha1.WebhookInterceptor{
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
	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{

					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1alpha1",
						},
						Name: "foo-trig",
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1alpha1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1alpha1",
							},
						},
						Interceptors: []*v1alpha1.TriggerInterceptor{
							{
								Webhook: &v1alpha1.WebhookInterceptor{
									ObjectRef: &corev1.ObjectReference{
										Kind:       "Service",
										Name:       "foo",
										Namespace:  "namespace",
										APIVersion: "v1",
									},
									Header: []v1beta1.Param{
										{
											Name: "header",
											Value: v1beta1.ArrayOrString{
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
	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{
					{
						TriggerRef: "test-ref",
					},
					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1alpha1",
						},
						Name: "foo-trig",

						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1alpha1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1alpha1",
							},
						},
						Interceptors: []*v1alpha1.TriggerInterceptor{
							{
								Webhook: &v1alpha1.WebhookInterceptor{
									ObjectRef: &corev1.ObjectReference{
										Kind:       "Service",
										Name:       "foo",
										Namespace:  "namespace",
										APIVersion: "v1",
									},
									Header: []v1beta1.Param{
										{
											Name: "header",
											Value: v1beta1.ArrayOrString{
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
	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{

					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1alpha1",
						},
						Name: "foo-trig",
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1alpha1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1alpha1",
							},
						},
						Interceptors: []*v1alpha1.TriggerInterceptor{
							{
								DeprecatedCEL: &v1alpha1.CELInterceptor{
									Filter: "body.value == 'test'",
									Overlays: []v1alpha1.CELOverlay{
										{
											Key:        "value",
											Expression: "'testing'",
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

func TestEventListenerDescribe_WithMultipleBindingAndInterceptors(t *testing.T) {
	triggerTemplateRef1 := "tt1"
	triggerTemplateRef2 := "tt2"
	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{

					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef1,
							APIVersion: "v1alpha1",
						},
						Name: "foo-trig",
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1alpha1",
							},
							{
								Ref:        "",
								Kind:       "TriggerBinding",
								APIVersion: "v1alpha1",
							},
						},
						Interceptors: []*v1alpha1.TriggerInterceptor{
							{
								DeprecatedCEL: &v1alpha1.CELInterceptor{
									Filter: "body.value == 'test'",
									Overlays: []v1alpha1.CELOverlay{
										{
											Key:        "value",
											Expression: "'testing'",
										},
									},
								},
							},
						},
					},
					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef2,
							APIVersion: "v1alpha1",
						},
						ServiceAccountName: "sa1",
						Name:               "foo-trig",
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Ref:        "tb4",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb5",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1alpha1",
							},
						},
						Interceptors: []*v1alpha1.TriggerInterceptor{
							{
								Webhook: &v1alpha1.WebhookInterceptor{
									ObjectRef: &corev1.ObjectReference{
										Kind:       "Service",
										Name:       "webhookTest",
										Namespace:  "namespace",
										APIVersion: "v1",
									},
								},
							},
							{
								DeprecatedCEL: &v1alpha1.CELInterceptor{
									Filter: "body.value == 'test'",
									Overlays: []v1alpha1.CELOverlay{
										{
											Key:        "value",
											Expression: "'testing'",
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

func TestEventListenerDescribe_OutputYAMLWithMultipleBindingAndInterceptors(t *testing.T) {
	triggerTemplateRef1 := "tt1"
	triggerTemplateRef2 := "tt2"
	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{

					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef1,
							APIVersion: "v1alpha1",
						},
						Name: "foo-trig",
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Ref:        "tb1",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb2",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1alpha1",
							},
							{
								Ref:        "tb3",
								Kind:       "TriggerBinding",
								APIVersion: "v1alpha1",
							},
						},
						Interceptors: []*v1alpha1.TriggerInterceptor{
							{
								DeprecatedCEL: &v1alpha1.CELInterceptor{
									Filter: "body.value == 'test'",
									Overlays: []v1alpha1.CELOverlay{
										{
											Key:        "value",
											Expression: "'testing'",
										},
									},
								},
							},
						},
					},
					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef2,
							APIVersion: "v1alpha1",
						},
						ServiceAccountName: "sa1",
						Name:               "foo-trig",
						Bindings: []*v1alpha1.EventListenerBinding{
							{
								Ref:        "tb4",
								Kind:       "TriggerBinding",
								APIVersion: "",
							},
							{
								Ref:        "tb5",
								Kind:       "ClusterTriggerBinding",
								APIVersion: "v1alpha1",
							},
						},
						Interceptors: []*v1alpha1.TriggerInterceptor{
							{
								Webhook: &v1alpha1.WebhookInterceptor{
									ObjectRef: &corev1.ObjectReference{
										Kind:       "Service",
										Name:       "webhookTest",
										Namespace:  "namespace",
										APIVersion: "v1",
									},
								},
							},
							{
								DeprecatedCEL: &v1alpha1.CELInterceptor{
									Filter: "body.value == 'test'",
									Overlays: []v1alpha1.CELOverlay{
										{
											Key:        "value",
											Expression: "'testing'",
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

	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els, Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	eventListener := Command(p)
	out, err := test.ExecuteCommand(eventListener, "desc", "el1", "-n", "ns", "-o", "json")
	if err != nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestEventListenerDescribe_WithOutputStatusURL(t *testing.T) {
	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},

			Status: v1alpha1.EventListenerStatus{
				AddressStatus: duckv1alpha1.AddressStatus{
					Address: &duckv1alpha1.Addressable{
						Addressable: beta1.Addressable{
							URL: &apis.URL{
								Scheme: "http",
								Host:   "el-listener.default.svc.cluster.local",
							},
						},
					},
				},
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els})

	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	eventListener := Command(p)
	out, err := test.ExecuteCommand(eventListener, "desc", "el1", "-o", "url", "-n", "ns")
	if err != nil {
		t.Errorf("Error")
	}
	test.AssertOutput(t, "http://el-listener.default.svc.cluster.local\n", out)
}

func TestEventListenerDescribe_OutputStatusURL_WithNoURL(t *testing.T) {
	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els})

	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	eventListener := Command(p)
	out, err := test.ExecuteCommand(eventListener, "desc", "el1", "-o", "url", "-n", "ns")
	if err == nil {
		t.Errorf("Error")
	}

	test.AssertOutput(t, "Error: "+err.Error()+"\n", out)
}

func TestEventListenerDescribe_AutoSelect(t *testing.T) {
	triggerTemplateRef := "tt1"
	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el1",
				Namespace: "ns",
			},
			Spec: v1alpha1.EventListenerSpec{
				Triggers: []v1alpha1.EventListenerTrigger{
					{
						Template: &v1alpha1.TriggerSpecTemplate{
							Ref:        &triggerTemplateRef,
							APIVersion: "v1alpha1",
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
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	eventListener := Command(p)
	out, err := test.ExecuteCommand(eventListener, "desc", "-n", "ns")
	if err != nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
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
