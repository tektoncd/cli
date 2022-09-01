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

	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
)

func SortByNamespace(pres []v1alpha1.PipelineResource) {
	sort.Sort(byNamespace(pres))
}

type byNamespace []v1alpha1.PipelineResource

func (pres byNamespace) compareNamespace(ins, jns string) (lt, eq bool) {
	lt, eq = ins < jns, ins == jns
	return lt, eq
}

func (pres byNamespace) Len() int      { return len(pres) }
func (pres byNamespace) Swap(i, j int) { pres[i], pres[j] = pres[j], pres[i] }
func (pres byNamespace) Less(i, j int) bool {
	var lt, eq bool
	if lt, eq = pres.compareNamespace(pres[i].Namespace, pres[j].Namespace); eq {
		if pres[j].Spec.Type < pres[i].Spec.Type {
			return false
		}
		if pres[j].Spec.Type > pres[i].Spec.Type {
			return true
		}
		return pres[j].Name > pres[i].Name
	}
	return lt
}
