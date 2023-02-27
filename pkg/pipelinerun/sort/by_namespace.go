// Copyright © 2020 The Tekton Authors.
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
	"sort"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func SortByNamespace(prs []v1.PipelineRun) {
	sort.Sort(byNamespace(prs))
}

type byNamespace []v1.PipelineRun

func (prs byNamespace) compareNamespace(ins, jns string) (lt, eq bool) {
	lt, eq = ins < jns, ins == jns
	return lt, eq
}

func (prs byNamespace) Len() int      { return len(prs) }
func (prs byNamespace) Swap(i, j int) { prs[i], prs[j] = prs[j], prs[i] }
func (prs byNamespace) Less(i, j int) bool {
	var lt, eq bool
	if lt, eq = prs.compareNamespace(prs[i].Namespace, prs[j].Namespace); eq {
		if prs[j].Status.StartTime == nil {
			return false
		}
		if prs[i].Status.StartTime == nil {
			return true
		}
		return prs[j].Status.StartTime.Before(prs[i].Status.StartTime)
	}
	return lt
}
