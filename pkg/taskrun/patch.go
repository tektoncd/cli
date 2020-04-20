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
	"encoding/json"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func Patch(c *cli.Clients, trname string, opts metav1.PatchOptions, ns string) (*v1beta1.TaskRun, error) {
	payload := []patchStringValue{{
		Op:    "replace",
		Path:  "/spec/status",
		Value: v1beta1.TaskRunSpecStatusCancelled,
	}}

	data, _ := json.Marshal(payload)
	unstructuredTR, err := actions.Patch(trGroupResource, c, trname, data, opts, ns)
	if err != nil {
		return nil, err
	}

	var taskrun *v1beta1.TaskRun
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredTR.UnstructuredContent(), &taskrun); err != nil {
		return nil, err
	}

	return taskrun, nil
}
