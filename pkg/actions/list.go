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
	"io"

	"github.com/tektoncd/cli/pkg/printer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

func PrintObjects(groupResource schema.GroupVersionResource, w io.Writer, dynamic dynamic.Interface, discovery discovery.DiscoveryInterface, f *cliopts.PrintFlags, ns string) error {
	allres, err := List(groupResource, dynamic, discovery, ns, metav1.ListOptions{})
	if err != nil {
		return err
	}

	return printer.PrintObject(w, allres, f)
}

func List(gr schema.GroupVersionResource, dynamic dynamic.Interface, discovery discovery.DiscoveryInterface, ns string, op metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	gvr, err := GetGroupVersionResource(gr, discovery)
	if err != nil {
		return nil, err
	}

	allRes, err := dynamic.Resource(*gvr).Namespace(ns).List(context.Background(), op)
	if err != nil {
		return nil, err
	}

	return allRes, nil
}
