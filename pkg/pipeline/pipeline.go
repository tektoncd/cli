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

	"github.com/tektoncd/cli/pkg/cli"
	plist "github.com/tektoncd/cli/pkg/list"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetAllPipelineNames(p cli.Params) ([]string, error) {
	cs, err := p.Clients()
	if err != nil {
		return nil, err
	}

	tkn := cs.Tekton.TektonV1alpha1()
	ps, err := tkn.Pipelines(p.Namespace()).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := []string{}
	for _, item := range ps.Items {
		ret = append(ret, item.ObjectMeta.Name)
	}
	return ret, nil
}

func List(c *cli.Clients, opts metav1.ListOptions, ns string) (*v1beta1.PipelineList, error) {

	pipelineGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "pipelines"}
	unstructuredP, err := plist.AllObjecs(pipelineGroupResource, c, ns, opts)
	if err != nil {
		return nil, err
	}

	var pipelines *v1beta1.PipelineList
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredP.UnstructuredContent(), &pipelines); err != nil {
		return nil, err
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list pipelines from %s namespace \n", ns)
		return nil, err
	}

	return pipelines, nil
}
