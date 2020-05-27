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
	"context"
	"fmt"
	"os"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var pipelineGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "pipelines"}

func GetAllPipelineNames(p cli.Params) ([]string, error) {
	cs, err := p.Clients()
	if err != nil {
		return nil, err
	}

	ps, err := List(cs, metav1.ListOptions{}, p.Namespace())
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
	unstructuredP, err := actions.List(pipelineGroupResource, c, ns, opts)
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

// It will fetch the resource based on the api available and return v1beta1 form
func Get(c *cli.Clients, pipelinename string, opts metav1.GetOptions, ns string) (*v1beta1.Pipeline, error) {
	gvr, err := actions.GetGroupVersionResource(pipelineGroupResource, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if gvr.Version == "v1alpha1" {
		pipeline, err := getV1alpha1(c, pipelinename, opts, ns)
		if err != nil {
			return nil, err
		}
		var pipelineConverted v1beta1.Pipeline
		err = pipeline.ConvertTo(context.Background(), &pipelineConverted)
		if err != nil {
			return nil, err
		}
		return &pipelineConverted, nil
	}
	return GetV1beta1(c, pipelinename, opts, ns)
}

// It will fetch the resource in v1beta1 struct format
func GetV1beta1(c *cli.Clients, pipelinename string, opts metav1.GetOptions, ns string) (*v1beta1.Pipeline, error) {
	unstructuredP, err := actions.Get(pipelineGroupResource, c, pipelinename, ns, opts)
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

// It will fetch the resource in v1alpha1 struct format
func getV1alpha1(c *cli.Clients, pipelinename string, opts metav1.GetOptions, ns string) (*v1alpha1.Pipeline, error) {
	unstructuredP, err := actions.Get(pipelineGroupResource, c, pipelinename, ns, opts)
	if err != nil {
		return nil, err
	}

	var pipeline *v1alpha1.Pipeline
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredP.UnstructuredContent(), &pipeline); err != nil {
		fmt.Fprintf(os.Stderr, "failed to get pipeline from %s namespace \n", ns)
		return nil, err
	}
	return pipeline, nil
}
