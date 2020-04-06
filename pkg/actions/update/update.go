// Copyright © 2019-2020 The Tekton Authors.
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

package update

import (
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Update(gr schema.GroupVersionResource, clients *cli.Clients, obj *unstructured.Unstructured, opts metav1.UpdateOptions, ns string) (*unstructured.Unstructured, error) {
	gvr, err := actions.GetGroupVersionResource(gr, clients.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	updatedObj, err := clients.Dynamic.Resource(*gvr).Namespace(ns).Update(obj, opts)
	if err != nil {
		return nil, err
	}

	return updatedObj, nil
}
