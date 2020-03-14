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
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	prsort "github.com/tektoncd/cli/pkg/pipelinerun/sort"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	runs, err := cs.Tekton.TektonV1alpha1().PipelineRuns(p.Namespace()).List(opts)
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
