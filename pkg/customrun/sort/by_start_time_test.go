// Copyright © 2023 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// discributed under the License is discributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package customrun

import (
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_TaskRunsByStartTime(t *testing.T) {
	clock := test.FakeClock()

	cr0Started := clock.Now().Add(10 * time.Second)
	cr1Started := clock.Now().Add(-2 * time.Hour)
	cr2Started := clock.Now().Add(-1 * time.Hour)

	cr1 := v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "cr0-1",
		},
		Status: v1beta1.CustomRunStatus{
			CustomRunStatusFields: v1beta1.CustomRunStatusFields{
				StartTime: &metav1.Time{Time: cr0Started},
			},
		},
	}

	cr2 := v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "cr1-1",
		},
		Status: v1beta1.CustomRunStatus{
			CustomRunStatusFields: v1beta1.CustomRunStatusFields{
				StartTime: &metav1.Time{Time: cr1Started},
			},
		},
	}

	cr3 := v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "cr2-1",
		},
		Status: v1beta1.CustomRunStatus{
			CustomRunStatusFields: v1beta1.CustomRunStatusFields{
				StartTime: &metav1.Time{Time: cr2Started},
			},
		},
	}

	crs := []v1beta1.CustomRun{
		cr2,
		cr3,
		cr1,
	}

	SortByStartTime(crs)

	element1 := crs[0].Name
	if element1 != "cr0-1" {
		t.Errorf("SortCustomRunsByStartTime should be cr0-1 but returned: %s", element1)
	}

	element2 := crs[1].Name
	if element2 != "cr2-1" {
		t.Errorf("SortCustomRunsByStartTime should be cr2-1 but returned: %s", element2)
	}

	element3 := crs[2].Name
	if element3 != "cr1-1" {
		t.Errorf("SortCustomRunsByStartTime should be cr1-1 but returned: %s", element3)
	}
}

func Test_TaskRunsByStartTime_NilStartTime(t *testing.T) {

	cr1 := v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "cr0-1",
		},
	}

	cr2 := v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "cr1-1",
		},
	}

	cr3 := v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "cr2-1",
		},
	}

	crs := []v1beta1.CustomRun{
		cr2,
		cr3,
		cr1,
	}

	SortByStartTime(crs)

	element1 := crs[0].Name
	if element1 != "cr1-1" {
		t.Errorf("SortCustomRunsByStartTime should be cr1-1 but returned: %s", element1)
	}

	element2 := crs[1].Name
	if element2 != "cr2-1" {
		t.Errorf("SortCustomRunsByStartTime should be cr2-1 but returned: %s", element2)
	}

	element3 := crs[2].Name
	if element3 != "cr0-1" {
		t.Errorf("SortCustomRunsByStartTime should be cr0-1 but returned: %s", element3)
	}
}
