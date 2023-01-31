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
	"fmt"
	"io"
	"os"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/printer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

// PrintObjects takes a partial resource, fetches a list of that resource's objects in the cluster using the dynamic client, and prints out the objects.
func PrintObjects(groupResource schema.GroupVersionResource, w io.Writer, dynamic dynamic.Interface, discovery discovery.DiscoveryInterface, f *cliopts.PrintFlags, ns string) error {
	allres, err := list(groupResource, dynamic, discovery, ns, metav1.ListOptions{})
	if err != nil {
		return err
	}

	return printer.PrintObject(w, allres, f)
}

// List fetches the resource and convert it to respective object
func ListV1(gr schema.GroupVersionResource, c *cli.Clients, opts metav1.ListOptions, ns string, obj interface{}) error {
	unstructuredObj, err := list(gr, c.Dynamic, c.Tekton.Discovery(), ns, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list objects from %s namespace \n", ns)
		return err
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), obj); err != nil {
		return err
	}
	return nil
}

// list takes a partial resource and fetches a list of that resource's objects in the cluster using the dynamic client.
func list(gr schema.GroupVersionResource, dynamic dynamic.Interface, discovery discovery.DiscoveryInterface, ns string, op metav1.ListOptions) (*unstructured.UnstructuredList, error) {
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

// TODO: remove as all the function uses are moved to new func
// List takes a partial resource and fetches a list of that resource's objects in the cluster using the dynamic client.
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
