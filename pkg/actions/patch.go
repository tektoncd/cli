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
	"fmt"
	"os"

	"github.com/tektoncd/cli/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// Patch takes a partial resource, an object name in the cluster, and patch data to be applied to that object, and patches the object using the dynamic client.
func Patch(gr schema.GroupVersionResource, clients *cli.Clients, objName string, data []byte, opt metav1.PatchOptions, ns string, obj interface{}) error {
	gvr, err := GetGroupVersionResource(gr, clients.Tekton.Discovery())
	if err != nil {
		return err
	}
	unstructuredObj, err := clients.Dynamic.Resource(*gvr).Namespace(ns).Patch(context.Background(), objName, types.JSONPatchType, data, opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to patch object from %s namespace \n", ns)
		return err
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), obj); err != nil {
		return err
	}

	return nil
}
