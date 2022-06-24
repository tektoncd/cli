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

package actions

import (
	"context"

	"github.com/tektoncd/cli/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Create is used to take a partial resource and an unstructured object and create it in the cluster using the dynamic client.
func Create(gr schema.GroupVersionResource, clients *cli.Clients, object *unstructured.Unstructured, ns string, op metav1.CreateOptions) (*unstructured.Unstructured, error) {
	gvr, err := GetGroupVersionResource(gr, clients.Tekton.Discovery())
	if err != nil {
		return nil, err
	}
	obj, err := clients.Dynamic.Resource(*gvr).Namespace(ns).Create(context.Background(), object, op)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
