/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1beta1 "github.com/tektoncd/triggers/pkg/client/clientset/versioned/typed/triggers/v1beta1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeTriggersV1beta1 struct {
	*testing.Fake
}

func (c *FakeTriggersV1beta1) ClusterTriggerBindings() v1beta1.ClusterTriggerBindingInterface {
	return newFakeClusterTriggerBindings(c)
}

func (c *FakeTriggersV1beta1) EventListeners(namespace string) v1beta1.EventListenerInterface {
	return newFakeEventListeners(c, namespace)
}

func (c *FakeTriggersV1beta1) Triggers(namespace string) v1beta1.TriggerInterface {
	return newFakeTriggers(c, namespace)
}

func (c *FakeTriggersV1beta1) TriggerBindings(namespace string) v1beta1.TriggerBindingInterface {
	return newFakeTriggerBindings(c, namespace)
}

func (c *FakeTriggersV1beta1) TriggerTemplates(namespace string) v1beta1.TriggerTemplateInterface {
	return newFakeTriggerTemplates(c, namespace)
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeTriggersV1beta1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
