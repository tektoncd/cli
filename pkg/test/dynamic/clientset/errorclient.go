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

package clientset

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

func newErrorResource(r schema.GroupVersionResource) errorResourceInterface {
	return errorResourceInterface{resource: r}
}

type errorResourceInterface struct {
	resource schema.GroupVersionResource
}

func (i errorResourceInterface) Namespace(string) dynamic.ResourceInterface {
	return i
}

func (i errorResourceInterface) err() error {
	return fmt.Errorf("resource %+v not supported", i.resource)
}

func (i errorResourceInterface) Create(_ context.Context, _ *unstructured.Unstructured, _ metav1.CreateOptions, _ ...string) (*unstructured.Unstructured, error) {
	return nil, i.err()
}

func (i errorResourceInterface) Update(_ context.Context, _ *unstructured.Unstructured, _ metav1.UpdateOptions, _ ...string) (*unstructured.Unstructured, error) {
	return nil, i.err()
}

func (i errorResourceInterface) UpdateStatus(_ context.Context, _ *unstructured.Unstructured, _ metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	return nil, i.err()
}

func (i errorResourceInterface) Delete(_ context.Context, _ string, _ metav1.DeleteOptions, _ ...string) error {
	return i.err()
}

func (i errorResourceInterface) DeleteCollection(_ context.Context, _ metav1.DeleteOptions, _ metav1.ListOptions) error {
	return i.err()
}

func (i errorResourceInterface) Get(_ context.Context, _ string, _ metav1.GetOptions, _ ...string) (*unstructured.Unstructured, error) {
	return nil, i.err()
}

func (i errorResourceInterface) List(_ context.Context, _ metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return nil, i.err()
}

func (i errorResourceInterface) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	return nil, i.err()
}

func (i errorResourceInterface) Patch(_ context.Context, _ string, _ types.PatchType, _ []byte, _ metav1.PatchOptions, _ ...string) (*unstructured.Unstructured, error) {
	return nil, i.err()
}

func (i errorResourceInterface) Apply(_ context.Context, _ string, _ *unstructured.Unstructured, _ metav1.ApplyOptions, _ ...string) (*unstructured.Unstructured, error) {
	return nil, i.err()
}

func (i errorResourceInterface) ApplyStatus(_ context.Context, _ string, _ *unstructured.Unstructured, _ metav1.ApplyOptions) (*unstructured.Unstructured, error) {
	return nil, i.err()
}
