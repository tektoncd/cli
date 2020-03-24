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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

type Run struct {
	Name string
	Task string
}

func IsFiltered(tr Run, allowed []string) bool {
	trs := []Run{tr}
	return len(Filter(trs, allowed)) == 0
}

func HasScheduled(status interface{}) bool {
	switch s := status.(type) {
	case *v1alpha1.PipelineRunTaskRunStatus:
		return s.Status != nil && s.Status.PodName != ""
	case *v1alpha1.PipelineRunConditionCheckStatus:
		return s.Status != nil && s.Status.PodName != ""
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

type taskRunMap map[string]*v1alpha1.PipelineRunTaskRunStatus

func SortTasksBySpecOrder(pipelineTasks []v1alpha1.PipelineTask, pipelinesTaskRuns taskRunMap) []Run {
	trNames := map[string]string{}

	for name, t := range pipelinesTaskRuns {
		// making mapping of (condtion name : pod name) if task has conditions.
		for podName, cond := range t.ConditionChecks {
			trNames[cond.ConditionName] = podName
		}
		trNames[t.PipelineTaskName] = name
	}

	trs := []Run{}
	for _, ts := range pipelineTasks {
		// checking conditions before appending task, pod names.
		for _, cond := range ts.Conditions {
			if podName, ok := trNames[cond.ConditionRef]; ok {
				trs = append(trs, Run{Task: cond.ConditionRef, Name: podName})
			}
		}
		if n, ok := trNames[ts.Name]; ok {
			trs = append(trs, Run{
				Task: ts.Name,
				Name: n,
			})
		}
	}

	return trs
}
