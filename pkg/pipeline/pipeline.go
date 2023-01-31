// Copyright © 2019 The Tekton Authors.
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

package pipeline

import (
	"fmt"
	"os"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var pipelineGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "pipelines"}

func GetAllPipelineNames(gr schema.GroupVersionResource, c *cli.Clients, ns string) ([]string, error) {
	var pipelines *v1.PipelineList
	if err := actions.ListV1(gr, c, metav1.ListOptions{}, ns, &pipelines); err != nil {
		return nil, fmt.Errorf("failed to list Tasks from namespace %s: %v", ns, err)
	}

	ret := []string{}
	for _, item := range pipelines.Items {
		ret = append(ret, item.ObjectMeta.Name)
	}
	return ret, nil
}

// Get will fetch the pipeline resource based on pipeline name
func Get(c *cli.Clients, pipelinename string, opts metav1.GetOptions, ns string) (*v1beta1.Pipeline, error) {
	unstructuredP, err := actions.Get(pipelineGroupResource, c.Dynamic, c.Tekton.Discovery(), pipelinename, ns, opts)
	if err != nil {
		return nil, err
	}

	var pipeline *v1beta1.Pipeline
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredP.UnstructuredContent(), &pipeline); err != nil {
		fmt.Fprintf(os.Stderr, "failed to get pipeline from %s namespace \n", ns)
		return nil, err
	}
	return pipeline, nil
}
