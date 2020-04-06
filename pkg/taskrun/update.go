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

package taskrun

import (
	"fmt"
	"os"

	traction "github.com/tektoncd/cli/pkg/actions/update"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Update(c *cli.Clients, trobj *v1beta1.TaskRun, opts metav1.UpdateOptions, ns string) (*v1beta1.TaskRun, error) {
	trGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}
	unsMapTRObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(trobj)
	if err != nil {
		return nil, err
	}
	unsTRObj := &unstructured.Unstructured{
		Object: unsMapTRObj,
	}
	unstructuredTR, err := traction.Update(trGroupResource, c, unsTRObj, opts, ns)
	if err != nil {
		return nil, err
	}

	var taskrun *v1beta1.TaskRun
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredTR.UnstructuredContent(), &taskrun); err != nil {
		return nil, err
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get taskrun from %s namespace \n", ns)
		return nil, err
	}
	return taskrun, nil
}
