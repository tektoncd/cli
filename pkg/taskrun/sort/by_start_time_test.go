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
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_TaskRunsByStartTime(t *testing.T) {
	clock := test.FakeClock()

	tr0Started := clock.Now().Add(10 * time.Second)
	tr1Started := clock.Now().Add(-2 * time.Hour)
	tr2Started := clock.Now().Add(-1 * time.Hour)

	tr1 := v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "tr0-1",
		},
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{
				StartTime: &metav1.Time{Time: tr0Started},
			},
		},
	}

	tr2 := v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "tr1-1",
		},
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{
				StartTime: &metav1.Time{Time: tr1Started},
			},
		},
	}

	tr3 := v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "tr2-1",
		},
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{
				StartTime: &metav1.Time{Time: tr2Started},
			},
		},
	}

	trs := []v1.TaskRun{
		tr2,
		tr3,
		tr1,
	}

	SortByStartTime(trs)

	element1 := trs[0].Name
	if element1 != "tr0-1" {
		t.Errorf("SortTaskRunsByStartTime should be tr0-1 but returned: %s", element1)
	}

	element2 := trs[1].Name
	if element2 != "tr2-1" {
		t.Errorf("SortTaskRunsByStartTime should be tr2-1 but returned: %s", element2)
	}

	element3 := trs[2].Name
	if element3 != "tr1-1" {
		t.Errorf("SortTaskRunsByStartTime should be tr1-1 but returned: %s", element3)
	}
}

func Test_TaskRunsByStartTime_NilStartTime(t *testing.T) {

	tr1 := v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "tr0-1",
		},
	}

	tr2 := v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "tr1-1",
		},
	}

	tr3 := v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "tr2-1",
		},
	}

	trs := []v1.TaskRun{
		tr2,
		tr3,
		tr1,
	}

	SortByStartTime(trs)

	element1 := trs[0].Name
	if element1 != "tr1-1" {
		t.Errorf("SortTaskRunsByStartTime should be tr1-1 but returned: %s", element1)
	}

	element2 := trs[1].Name
	if element2 != "tr2-1" {
		t.Errorf("SortTaskRunsByStartTime should be tr2-1 but returned: %s", element2)
	}

	element3 := trs[2].Name
	if element3 != "tr0-1" {
		t.Errorf("SortTaskRunsByStartTime should be tr0-1 but returned: %s", element3)
	}
}
