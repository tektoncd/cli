// Copyright © 2023 The Tekton Authors.
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

package customrun

import (
	"sort"

	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func SortByStartTime(crs []v1beta1.CustomRun) {
	sort.Sort(byStartTime(crs))
}

type byStartTime []v1beta1.CustomRun

func (crs byStartTime) Len() int      { return len(crs) }
func (crs byStartTime) Swap(i, j int) { crs[i], crs[j] = crs[j], crs[i] }
func (crs byStartTime) Less(i, j int) bool {
	if crs[j].Status.StartTime == nil {
		return false
	}
	if crs[i].Status.StartTime == nil {
		return true
	}
	return crs[j].Status.StartTime.Before(crs[i].Status.StartTime)
}
