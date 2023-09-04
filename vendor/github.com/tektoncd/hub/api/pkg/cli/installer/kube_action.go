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
	"fmt"
	"strings"

	gvrRes "github.com/tektoncd/hub/api/pkg/cli/gvr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const tektonGroup = "tekton.dev"

func (i *Installer) create(object *unstructured.Unstructured, namespace string, op metav1.CreateOptions) (*unstructured.Unstructured, error) {

	grObj := schema.GroupVersionResource{Group: tektonGroup, Resource: object.GetKind()}
	versions, err := gvrRes.GetVersionList(grObj, i.cs.Tekton().Discovery())
	if err != nil {
		return nil, err
	}

	gvr := schema.GroupVersionResource{
		Group:    object.GroupVersionKind().Group,
		Resource: strings.ToLower(object.GetKind()) + "s",
	}

	if contains(versions, object.GroupVersionKind().Version) {
		gvr.Version = object.GroupVersionKind().Version
	} else {
		return nil, fmt.Errorf("Error: API version in the data %s does not match the expected API version", object.GroupVersionKind().Version)
	}

	obj, err := i.cs.Dynamic().Resource(gvr).Namespace(namespace).Create(context.Background(), object, op)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (i *Installer) get(objectName, kind, namespace string, op metav1.GetOptions) (*unstructured.Unstructured, error) {

	grObj := schema.GroupVersionResource{Group: tektonGroup, Resource: kind}
	versions, err := gvrRes.GetVersionList(grObj, i.cs.Tekton().Discovery())
	if err != nil {
		return nil, err
	}

	gvr := schema.GroupVersionResource{
		Group:    tektonGroup,
		Resource: strings.ToLower(kind) + "s",
		Version:  versions[0],
	}
	obj, err := i.cs.Dynamic().Resource(gvr).Namespace(namespace).Get(context.Background(), objectName, op)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (i *Installer) update(object *unstructured.Unstructured, namespace string, op metav1.UpdateOptions) (*unstructured.Unstructured, error) {

	grObj := schema.GroupVersionResource{Group: tektonGroup, Resource: object.GetKind()}
	versions, err := gvrRes.GetVersionList(grObj, i.cs.Tekton().Discovery())
	if err != nil {
		return nil, err
	}
	gvr := schema.GroupVersionResource{
		Group:    object.GroupVersionKind().Group,
		Resource: strings.ToLower(object.GetKind()) + "s",
	}
	if contains(versions, object.GroupVersionKind().Version) {
		gvr.Version = object.GroupVersionKind().Version
	} else {
		return nil, fmt.Errorf("Error: API version in the data %s does not match the expected API version", object.GroupVersionKind().Version)
	}

	obj, err := i.cs.Dynamic().Resource(gvr).Namespace(namespace).Update(context.Background(), object, op)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (i *Installer) list(kind, namespace string, op metav1.ListOptions) (*unstructured.UnstructuredList, error) {

	grObj := schema.GroupVersionResource{Group: tektonGroup, Resource: strings.ToLower(kind) + "s"}
	versions, err := gvrRes.GetVersionList(grObj, i.cs.Tekton().Discovery())
	if err != nil {
		return nil, err
	}

	gvr := schema.GroupVersionResource{
		Group:    tektonGroup,
		Resource: strings.ToLower(kind) + "s",
		Version:  versions[0],
	}
	obj, err := i.cs.Dynamic().Resource(gvr).Namespace(namespace).List(context.Background(), op)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// contains checks whether the array contains the string and return true or false
func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
