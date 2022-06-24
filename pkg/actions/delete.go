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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

// Delete is used to take a partial resource and the name of an object in the cluster and delete it using the dynamic client.
func Delete(gr schema.GroupVersionResource, dynamic dynamic.Interface, discovery discovery.DiscoveryInterface, objname, ns string, op metav1.DeleteOptions) error {
	gvr, err := GetGroupVersionResource(gr, discovery)
	if err != nil {
		return err
	}

	err = dynamic.Resource(*gvr).Namespace(ns).Delete(context.Background(), objname, op)
	if err != nil {
		return err
	}

	return nil
}
