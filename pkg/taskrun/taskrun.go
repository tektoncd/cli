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

package taskrun

import (
	"sort"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Run struct {
	Name           string
	Task           string
	Retries        int
	StartTime      *metav1.Time
	CompletionTime *metav1.Time
}

type Runs []Run

func (r Runs) Len() int {
	return len(r)
}

func (r Runs) Less(i, j int) bool {
	if r[i].CompletionTime != nil && r[j].CompletionTime != nil {
		return r[i].CompletionTime.Before(r[j].CompletionTime)
	}
	return r[i].StartTime.Before(r[j].StartTime)
}

func (r Runs) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func IsFiltered(tr Run, allowed []string) bool {
	trs := []Run{tr}
	return len(Filter(trs, allowed)) == 0
}

func HasScheduled(trs *v1.PipelineRunTaskRunStatus) bool {
	if trs.Status != nil {
		return trs.Status.PodName != ""
	}
	return false
}

func Filter(trs []Run, ts []string) []Run {
	if len(ts) == 0 {
		return trs
	}

	filter := map[string]bool{}
	for _, t := range ts {
		filter[t] = true
	}

	filtered := []Run{}
	for _, tr := range trs {
		if filter[tr.Task] {
			filtered = append(filtered, tr)
		}
	}

	return filtered
}

func SortTasksBySpecOrder(pipelineTasks []v1.PipelineTask, pipelinesTaskRuns map[string]*v1.PipelineRunTaskRunStatus) []Run {
	trNames := map[string]string{}

	for name, t := range pipelinesTaskRuns {
		trNames[t.PipelineTaskName] = name
	}
	trs := Runs{}

	for _, ts := range pipelineTasks {
		if n, ok := trNames[ts.Name]; ok {
			trStatusFields := pipelinesTaskRuns[n].Status.TaskRunStatusFields
			trs = append(trs, Run{
				Task:           ts.Name,
				Name:           n,
				Retries:        ts.Retries,
				StartTime:      trStatusFields.StartTime,
				CompletionTime: trStatusFields.CompletionTime,
			})
		}
	}
	sort.Sort(trs)
	return trs
}
