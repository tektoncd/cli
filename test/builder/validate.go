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
	"fmt"
	"sort"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func Details(pre v1alpha1.PipelineResource) string {
	var key = "url"
	if pre.Spec.Type == v1beta1.PipelineResourceTypeStorage {
		key = "location"
	}

	for _, p := range pre.Spec.Params {
		if strings.ToLower(p.Name) == key {
			return p.Name + ": " + p.Value
		}
	}

	return "---"
}

func TaskRunHasFailed(tr *v1beta1.TaskRun) string {
	if len(tr.Status.Conditions) == 0 {
		return ""
	}

	if tr.Status.Conditions[0].Status == corev1.ConditionFalse {
		return tr.Status.Conditions[0].Message
	}
	return ""
}

// this will sort the Resource by Type and then by Name
func SortResourcesByTypeAndName(pres []v1beta1.PipelineDeclaredResource) []v1beta1.PipelineDeclaredResource {
	sort.Slice(pres, func(i, j int) bool {
		if pres[j].Type < pres[i].Type {
			return false
		}

		if pres[j].Type > pres[i].Type {
			return true
		}

		return pres[j].Name > pres[i].Name
	})

	return pres
}

// Pipeline Run Describe command

func PipelineRunHasFailed(pr *v1beta1.PipelineRun) string {
	if len(pr.Status.Conditions) == 0 {
		return ""
	}

	if pr.Status.Conditions[0].Status == corev1.ConditionFalse {
		for _, taskrunStatus := range pr.Status.TaskRuns {
			if len(taskrunStatus.Status.Conditions) == 0 {
				continue
			}
			if taskrunStatus.Status.Conditions[0].Status == corev1.ConditionFalse {
				return fmt.Sprintf("%s (%s)", pr.Status.Conditions[0].Message,
					taskrunStatus.Status.Conditions[0].Message)
			}
		}
		return pr.Status.Conditions[0].Message
	}
	return ""
}

type TaskrunList []tkr

type tkr struct {
	TaskrunName string
	*v1beta1.PipelineRunTaskRunStatus
}

func (s TaskrunList) Len() int      { return len(s) }
func (s TaskrunList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s TaskrunList) Less(i, j int) bool {
	return s[j].Status.StartTime.Before(s[i].Status.StartTime)
}
