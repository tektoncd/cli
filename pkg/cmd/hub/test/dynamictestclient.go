// Copyright © 2022 The Tekton Authors.
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

package test

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

func DynamicClient(objects ...runtime.Object) *fake.FakeDynamicClient {
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			{Group: "tekton.dev", Version: "v1", Resource: "tasks"}:       "TaskList",
			{Group: "tekton.dev", Version: "v1alpha1", Resource: "tasks"}: "TaskList",
			{Group: "tekton.dev", Version: "v1beta1", Resource: "tasks"}:  "TaskList",
			{Group: "apps", Version: "v1", Resource: "deployments"}:       "DeploymentList",
		},
		objects...,
	)

	return dynamicClient
}
