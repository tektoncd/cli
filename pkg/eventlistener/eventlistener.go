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
	"fmt"
	"os"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var eventlistenerGroupResource = schema.GroupVersionResource{Group: "triggers.tekton.dev", Resource: "eventlisteners"}

func GetAllEventListenerNames(client *cli.Clients, namespace string) ([]string, error) {
	ps, err := List(client, metav1.ListOptions{}, namespace)
	if err != nil {
		return nil, err
	}

	ret := []string{}
	for _, item := range ps.Items {
		ret = append(ret, item.ObjectMeta.Name)
	}
	return ret, nil
}

func List(c *cli.Clients, opts metav1.ListOptions, ns string) (*v1beta1.EventListenerList, error) {
	unstructuredEL, err := actions.List(eventlistenerGroupResource, c.Dynamic, c.Triggers.Discovery(), ns, opts)
	if err != nil {
		return nil, err
	}

	var eventlisteners *v1beta1.EventListenerList
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredEL.UnstructuredContent(), &eventlisteners); err != nil {
		return nil, err
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list eventlisteners from %s namespace \n", ns)
		return nil, err
	}

	return eventlisteners, nil
}

func Get(c *cli.Clients, elName string, opts metav1.GetOptions, ns string) (*v1beta1.EventListener, error) {
	unstructuredEl, err := actions.Get(eventlistenerGroupResource, c.Dynamic, c.Triggers.Discovery(), elName, ns, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get EventListener %s: %v", elName, err)
	}

	var el *v1beta1.EventListener
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredEl.UnstructuredContent(), &el); err != nil {
		fmt.Fprintf(os.Stderr, "failed to get eventlistener from %s namespace \n", ns)
		return nil, err
	}
	return el, nil
}
