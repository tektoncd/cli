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

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func SortByStartTimeTaskRun(trs []v1.TaskRun) {
	sort.Sort(byStartTimeTR(trs))
}

func SortByStartTimePipelineRun(prs []v1.PipelineRun) {
	sort.Sort(byStartTimePR(prs))
}

type byStartTimeTR []v1.TaskRun

func (trs byStartTimeTR) Len() int      { return len(trs) }
func (trs byStartTimeTR) Swap(i, j int) { trs[i], trs[j] = trs[j], trs[i] }
func (trs byStartTimeTR) Less(i, j int) bool {
	if trs[j].Status.StartTime == nil {
		return false
	}
	if trs[i].Status.StartTime == nil {
		return true
	}
	return trs[j].Status.StartTime.Before(trs[i].Status.StartTime)
}

type byStartTimePR []v1.PipelineRun

func (prs byStartTimePR) Len() int      { return len(prs) }
func (prs byStartTimePR) Swap(i, j int) { prs[i], prs[j] = prs[j], prs[i] }
func (prs byStartTimePR) Less(i, j int) bool {
	if prs[j].Status.StartTime == nil {
		return false
	}
	if prs[i].Status.StartTime == nil {
		return true
	}
	return prs[j].Status.StartTime.Before(prs[i].Status.StartTime)
}
