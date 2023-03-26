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

package pipelinerun

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	prsort "github.com/tektoncd/cli/pkg/pipelinerun/sort"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var pipelineRunGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "pipelineruns"}
var taskrunGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}

// GetAllPipelineRuns returns all pipelinesruns running in a namespace
func GetAllPipelineRuns(gr schema.GroupVersionResource, opts metav1.ListOptions, c *cli.Clients, ns string, limit int, time clockwork.Clock) ([]string, error) {
	var pipelineruns *v1.PipelineRunList
	if err := actions.ListV1(gr, c, opts, ns, &pipelineruns); err != nil {
		return nil, fmt.Errorf("failed to list PipelineRuns from namespace %s: %v", ns, err)
	}

	runslen := len(pipelineruns.Items)
	if limit > runslen {
		limit = runslen
	}

	if runslen > 1 {
		prsort.SortByStartTime(pipelineruns.Items)
	}
	ret := []string{}
	for i, run := range pipelineruns.Items {
		if i < limit {
			ret = append(ret, run.ObjectMeta.Name+" started "+formatted.Age(run.Status.StartTime, time))
		}
	}
	return ret, nil
}

type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func Cancel(c *cli.Clients, prname string, opts metav1.PatchOptions, cancelStatus, ns string) (*v1.PipelineRun, error) {
	payload := []patchStringValue{{
		Op:    "replace",
		Path:  "/spec/status",
		Value: cancelStatus,
	}}

	data, _ := json.Marshal(payload)
	prGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "pipelineruns"}
	var pipelinerun *v1.PipelineRun
	err := actions.Patch(prGroupResource, c, prname, data, opts, ns, &pipelinerun)
	if err != nil {
		return nil, err
	}

	return pipelinerun, nil
}

// It will create the resource based on the api available.
func Create(c *cli.Clients, pr *v1beta1.PipelineRun, opts metav1.CreateOptions, ns string) (*v1beta1.PipelineRun, error) {
	gvr, err := actions.GetGroupVersionResource(pipelineRunGroupResource, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if gvr.Version == "v1" {
		prv1 := v1.PipelineRun{}
		err = pr.ConvertTo(context.Background(), &prv1)
		if err != nil {
			return nil, err
		}
		prv1.Kind = "PipelineRun"
		prv1.APIVersion = "tekton.dev/v1"

		object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&prv1)
		unstructuredPR := &unstructured.Unstructured{
			Object: object,
		}
		newUnstructuredPR, err := actions.Create(pipelineRunGroupResource, c, unstructuredPR, ns, opts)
		if err != nil {
			return nil, err
		}
		var pipelinerun v1.PipelineRun
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(newUnstructuredPR.UnstructuredContent(), &pipelinerun); err != nil {
			return nil, err
		}

		pipelinerunv1beta1 := v1beta1.PipelineRun{}
		err = pipelinerunv1beta1.ConvertFrom(context.Background(), &pipelinerun)
		if err != nil {
			return nil, err
		}
		pipelinerunv1beta1.Kind = "PipelineRun"
		pipelinerunv1beta1.APIVersion = "tekton.dev/v1beta1"
		return &pipelinerunv1beta1, nil
	}

	object, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pr)
	unstructuredPR := &unstructured.Unstructured{
		Object: object,
	}

	newUnstructuredPR, err := actions.Create(pipelineRunGroupResource, c, unstructuredPR, ns, opts)
	if err != nil {
		return nil, err
	}

	var pipelinerun *v1beta1.PipelineRun
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(newUnstructuredPR.UnstructuredContent(), &pipelinerun); err != nil {
		return nil, err
	}

	return pipelinerun, nil
}
