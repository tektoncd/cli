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

package actions

import (
	"context"
	"io"

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

// TODO: remove as all the function uses are moved to new func
// PrintObject is used to take a partial resource and the name of an object in the cluster, fetch it using the dynamic client, and print out the object.
func PrintObject(groupResource schema.GroupVersionResource, obj string, w io.Writer, dynamic dynamic.Interface, discovery discovery.DiscoveryInterface, f *cliopts.PrintFlags, ns string) error {
	res, err := Get(groupResource, dynamic, discovery, obj, ns, metav1.GetOptions{})
	if err != nil {
		return err
	}

	return printer.PrintObject(w, res, f)
}

// PrintObject is used to take a partial resource and the name of an object in the cluster, fetch it using the dynamic client, and print out the object.
func PrintObjectV1(groupResource schema.GroupVersionResource, obj string, w io.Writer, client *cli.Clients, f *cliopts.PrintFlags, ns string) error {
	res, err := GetUnstructured(groupResource, client, obj, ns, metav1.GetOptions{})
	if err != nil {
		return err
	}

	return printer.PrintObject(w, res, f)
}

// GetV1 is used to take a partial resource and the name of an object in the cluster and fetch it from the cluster using the dynamic client.
func GetV1(gr schema.GroupVersionResource, c *cli.Clients, objname, ns string, op metav1.GetOptions, obj interface{}) error {
	unstructuredObj, err := GetUnstructured(gr, c, objname, ns, op)
	if err != nil {
		return err
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), obj); err != nil {
		return err
	}

	return nil
}

func GetUnstructured(gr schema.GroupVersionResource, c *cli.Clients, objname, ns string, op metav1.GetOptions) (*unstructured.Unstructured, error) {
	gvr, err := GetGroupVersionResource(gr, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	unstructuredObj, err := c.Dynamic.Resource(*gvr).Namespace(ns).Get(context.Background(), objname, op)
	if err != nil {
		return nil, err
	}
	return unstructuredObj, nil
}

// TODO: remove as all the function uses are moved to new func
// Get is used to take a partial resource and the name of an object in the cluster and fetch it from the cluster using the dynamic client.
func Get(gr schema.GroupVersionResource, dynamic dynamic.Interface, discovery discovery.DiscoveryInterface, objname, ns string, op metav1.GetOptions) (*unstructured.Unstructured, error) {
	gvr, err := GetGroupVersionResource(gr, discovery)
	if err != nil {
		return nil, err
	}

	obj, err := dynamic.Resource(*gvr).Namespace(ns).Get(context.Background(), objname, op)
	if err != nil {
		return nil, err
	}

	return obj, nil
}
