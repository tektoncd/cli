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

package builder

import (
	"sort"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

func SortByStartTime(trs []v1alpha1.TaskRun) {
	sort.Sort(byStartTime(trs))
}

type byStartTime []v1alpha1.TaskRun

func (trs byStartTime) Len() int      { return len(trs) }
func (trs byStartTime) Swap(i, j int) { trs[i], trs[j] = trs[j], trs[i] }
func (trs byStartTime) Less(i, j int) bool {
	if trs[j].Status.StartTime == nil {
		return false
	}
	if trs[i].Status.StartTime == nil {
		return true
	}
	return trs[j].Status.StartTime.Before(trs[i].Status.StartTime)
}
