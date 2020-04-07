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
	"encoding/json"
	"fmt"
	"os"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	prsort "github.com/tektoncd/cli/pkg/pipelinerun/sort"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

var prGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "pipelineruns"}

//Todo
// changing this GetPipelineRun to use dynamic client leads to multiple other change in start commands
// keeping this as it is, will change this while working on start commands
// GetPipelineRun return a pipelinerun in a namespace from its name
func GetPipelineRun(p cli.Params, opts metav1.GetOptions, prname string) (*v1alpha1.PipelineRun, error) {
	cs, err := p.Clients()
	if err != nil {
		return nil, err
	}

	prun, err := cs.Tekton.TektonV1alpha1().PipelineRuns(p.Namespace()).Get(prname, opts)
	if err != nil {
		return nil, err
	}
	return prun, nil
}

// GetAllPipelineRuns returns all pipelinesruns running in a namespace
func GetAllPipelineRuns(p cli.Params, opts metav1.ListOptions, limit int) ([]string, error) {
	cs, err := p.Clients()
	if err != nil {
		return nil, err
	}

	runs, err := List(cs, opts, p.Namespace())
	if err != nil {
		return nil, err
	}

	runslen := len(runs.Items)
	if runslen > 1 {
		prsort.SortByStartTime(runs.Items)
	}

	if limit > runslen {
		limit = runslen
	}

	ret := []string{}
	for i, run := range runs.Items {
		if i < limit {
			ret = append(ret, run.ObjectMeta.Name+" started "+formatted.Age(run.Status.StartTime, p.Time()))
		}
	}
	return ret, nil
}

func List(c *cli.Clients, opts metav1.ListOptions, ns string) (*v1beta1.PipelineRunList, error) {
	unstructuredPR, err := actions.List(prGroupResource, c, ns, opts)
	if err != nil {
		return nil, err
	}

	var runs *v1beta1.PipelineRunList
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredPR.UnstructuredContent(), &runs); err != nil {
		return nil, err
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list pipelineruns from %s namespace \n", ns)
		return nil, err
	}

	return runs, nil
}

func Get(c *cli.Clients, prname string, opts metav1.GetOptions, ns string) (*v1beta1.PipelineRun, error) {
	unstructuredPR, err := actions.Get(prGroupResource, c, prname, ns, opts)
	if err != nil {
		return nil, err
	}

	var run *v1beta1.PipelineRun
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredPR.UnstructuredContent(), &run); err != nil {
		return nil, err
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get pipelinerun from %s namespace \n", ns)
		return nil, err
	}
	return run, nil
}

func Watch(c *cli.Clients, opts metav1.ListOptions, ns string) (watch.Interface, error) {
	watch, err := actions.Watch(prGroupResource, c, ns, opts)
	if err != nil {
		return nil, err
	}
	return watch, nil
}

type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func Patch(c *cli.Clients, prname string, opts metav1.PatchOptions, ns string) (*v1beta1.PipelineRun, error) {
	payload := []patchStringValue{{
		Op:    "replace",
		Path:  "/spec/status",
		Value: v1beta1.PipelineRunSpecStatusCancelled,
	}}

	data, _ := json.Marshal(payload)
	prGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "pipelineruns"}
	unstructuredPR, err := actions.Patch(prGroupResource, c, prname, data, opts, ns)
	if err != nil {
		return nil, err
	}

	var pipelinerun *v1beta1.PipelineRun
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredPR.UnstructuredContent(), &pipelinerun); err != nil {
		return nil, err
	}

	return pipelinerun, nil
}
