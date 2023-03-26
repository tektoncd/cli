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

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var pipelineGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "pipelines"}
var pipelineRunGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "pipelineruns"}

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

func GetPipeline(gr schema.GroupVersionResource, c *cli.Clients, pName, ns string) (*v1.Pipeline, error) {
	var pipeline v1.Pipeline
	gvr, err := actions.GetGroupVersionResource(gr, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if gvr.Version == "v1" {
		err := actions.GetV1(gr, c, pName, ns, metav1.GetOptions{}, &pipeline)
		if err != nil {
			return nil, err
		}
		return &pipeline, nil

	}

	var pipelineV1beta1 v1beta1.Pipeline
	err = actions.GetV1(gr, c, pName, ns, metav1.GetOptions{}, &pipelineV1beta1)
	if err != nil {
		return nil, err
	}
	err = pipelineV1beta1.ConvertTo(context.Background(), &pipeline)
	if err != nil {
		return nil, err
	}
	return &pipeline, nil
}
