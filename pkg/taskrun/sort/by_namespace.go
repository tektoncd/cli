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

package taskrun

import (
	"sort"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func SortByNamespace(trs []v1.TaskRun) {
	sort.Sort(byNamespace(trs))
}

type byNamespace []v1.TaskRun

func (trs byNamespace) compareNamespace(ins, jns string) (lt, eq bool) {
	lt, eq = ins < jns, ins == jns
	return lt, eq
}

func (trs byNamespace) Len() int      { return len(trs) }
func (trs byNamespace) Swap(i, j int) { trs[i], trs[j] = trs[j], trs[i] }
func (trs byNamespace) Less(i, j int) bool {
	var lt, eq bool
	if lt, eq = trs.compareNamespace(trs[i].Namespace, trs[j].Namespace); eq {
		if trs[j].Status.StartTime == nil {
			return false
		}
		if trs[i].Status.StartTime == nil {
			return true
		}
		return trs[j].Status.StartTime.Before(trs[i].Status.StartTime)
	}
	return lt
}
