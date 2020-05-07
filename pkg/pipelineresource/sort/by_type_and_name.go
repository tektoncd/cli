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

package pipelineresource

import (
	"sort"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

func SortByTypeAndName(pres []v1alpha1.PipelineResource) {
	sort.Sort(byTypeAndName(pres))
}

type byTypeAndName []v1alpha1.PipelineResource

func (pres byTypeAndName) Len() int      { return len(pres) }
func (pres byTypeAndName) Swap(i, j int) { pres[i], pres[j] = pres[j], pres[i] }
func (pres byTypeAndName) Less(i, j int) bool {
	if pres[j].Spec.Type < pres[i].Spec.Type {
		return false
	}
	if pres[j].Spec.Type > pres[i].Spec.Type {
		return true
	}
	return pres[j].Name > pres[i].Name
}
