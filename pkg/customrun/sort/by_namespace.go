// Copyright © 2023 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package customrun

import (
	"sort"

	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func SortByNamespace(crs []v1beta1.CustomRun) {
	sort.Sort(byNamespace(crs))
}

type byNamespace []v1beta1.CustomRun

func (crs byNamespace) compareNamespace(ins, jns string) (lt, eq bool) {
	lt, eq = ins < jns, ins == jns
	return lt, eq
}

func (crs byNamespace) Len() int      { return len(crs) }
func (crs byNamespace) Swap(i, j int) { crs[i], crs[j] = crs[j], crs[i] }
func (crs byNamespace) Less(i, j int) bool {
	var lt, eq bool
	if lt, eq = crs.compareNamespace(crs[i].Namespace, crs[j].Namespace); eq {
		if crs[j].Status.StartTime == nil {
			return false
		}
		if crs[i].Status.StartTime == nil {
			return true
		}
		return crs[j].Status.StartTime.Before(crs[i].Status.StartTime)
	}
	return lt
}
