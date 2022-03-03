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

package installer

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const tektonGroup = "tekton.dev"

func (i *Installer) create(object *unstructured.Unstructured, namespace string, op metav1.CreateOptions) (*unstructured.Unstructured, error) {

	gvrObj := schema.GroupVersionResource{Group: tektonGroup, Resource: object.GetKind()}
	gvr, err := getGroupVersionResource(gvrObj, i.cs.Tekton().Discovery())
	if err != nil {
		return nil, err
	}

	obj, err := i.cs.Dynamic().Resource(*gvr).Namespace(namespace).Create(context.Background(), object, op)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (i *Installer) get(objectName, kind, namespace string, op metav1.GetOptions) (*unstructured.Unstructured, error) {

	gvrObj := schema.GroupVersionResource{Group: tektonGroup, Resource: kind}
	gvr, err := getGroupVersionResource(gvrObj, i.cs.Tekton().Discovery())
	if err != nil {
		return nil, err
	}
	obj, err := i.cs.Dynamic().Resource(*gvr).Namespace(namespace).Get(context.Background(), objectName, op)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (i *Installer) update(object *unstructured.Unstructured, namespace string, op metav1.UpdateOptions) (*unstructured.Unstructured, error) {

	gvrObj := schema.GroupVersionResource{Group: tektonGroup, Resource: object.GetKind()}
	gvr, err := getGroupVersionResource(gvrObj, i.cs.Tekton().Discovery())
	if err != nil {
		return nil, err
	}
	obj, err := i.cs.Dynamic().Resource(*gvr).Namespace(namespace).Update(context.Background(), object, op)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (i *Installer) list(kind, namespace string, op metav1.ListOptions) (*unstructured.UnstructuredList, error) {

	gvrObj := schema.GroupVersionResource{Group: tektonGroup, Resource: strings.ToLower(kind) + "s"}
	gvr, err := getGroupVersionResource(gvrObj, i.cs.Tekton().Discovery())
	if err != nil {
		return nil, err
	}
	obj, err := i.cs.Dynamic().Resource(*gvr).Namespace(namespace).List(context.Background(), op)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
