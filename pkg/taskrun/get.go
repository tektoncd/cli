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
	"context"
	"fmt"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Get will fetch the taskrun resource based on the taskrun name
func Get(c *cli.Clients, trname string, opts metav1.GetOptions, ns string) (*v1beta1.TaskRun, error) {
	unstructuredTR, err := actions.Get(trGroupResource, c.Dynamic, c.Tekton.Discovery(), trname, ns, opts)
	if err != nil {
		return nil, err
	}

	var taskrun *v1beta1.TaskRun
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredTR.UnstructuredContent(), &taskrun); err != nil {
		return nil, fmt.Errorf("failed to get TaskRun from namespace %s", ns)
	}
	return taskrun, nil
}

// GetV1 will fetch the taskrun resource based on the taskrun name
// TODO: remove after get action is moved to v1
func GetV1(c *cli.Clients, trname string, opts metav1.GetOptions, ns string) (*v1.TaskRun, error) {

	gvr, err := actions.GetGroupVersionResource(schema.GroupVersionResource{Group: "tekton.dev", Resource: "tasks"}, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if gvr.Version == "v1beta1" {
		taskRunV1Beta1, err := Get(c, trname, opts, ns)
		if err != nil {
			return nil, err
		}
		var taskRunConverted v1.TaskRun
		err = taskRunV1Beta1.ConvertTo(context.Background(), &taskRunConverted)
		if err != nil {
			return nil, err
		}
		return &taskRunConverted, nil
	}

	unstructuredTR, err := actions.Get(trGroupResource, c.Dynamic, c.Tekton.Discovery(), trname, ns, opts)
	if err != nil {
		return nil, err
	}

	var taskrun *v1.TaskRun
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredTR.UnstructuredContent(), &taskrun); err != nil {
		return nil, fmt.Errorf("failed to get TaskRun from namespace %s", ns)
	}
	return taskrun, nil
}
