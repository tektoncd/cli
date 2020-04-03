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

	trget "github.com/tektoncd/cli/pkg/actions/get"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Get(c *cli.Clients, trname string, opts metav1.GetOptions, ns string) (*v1beta1.TaskRun, error) {
	trGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}
	unstructuredTR, err := trget.Get(trGroupResource, c, trname, ns, opts)
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
