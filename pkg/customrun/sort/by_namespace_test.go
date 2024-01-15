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

func Test_TaskRunsByNamespace(t *testing.T) {
	crs := []v1beta1.CustomRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "def",
				Name:      "cr1-1",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ghi",
				Name:      "cr2-1",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "abc",
				Name:      "cr0-1",
			},
		},
	}

	SortByNamespace(crs)

	element1 := crs[0].Name
	if element1 != "cr0-1" {
		t.Errorf("SortCustomRunsByNamespace should be cr0-1 but returned: %s", element1)
	}

	element2 := crs[1].Name
	if element2 != "cr1-1" {
		t.Errorf("SortCustomRunsByNamespace should be cr1-1 but returned: %s", element2)
	}

	element3 := crs[2].Name
	if element3 != "cr2-1" {
		t.Errorf("SortCustomRunsByNamespace should be cr2-1 but returned: %s", element3)
	}
}

func Test_TaskRunsByNamespaceWithStartTime(t *testing.T) {

	clock := test.FakeClock()

	cr00Started := clock.Now().Add(10 * time.Second)
	cr01Started := clock.Now().Add(-1 * time.Hour)
	cr10Started := clock.Now().Add(10 * time.Second)
	cr11Started := clock.Now().Add(-1 * time.Hour)
	cr20Started := clock.Now().Add(10 * time.Second)
	cr21Started := clock.Now().Add(-1 * time.Hour)

	cr00 := v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "abc",
			Name:      "cr0-0",
		},
		Status: v1beta1.CustomRunStatus{
			CustomRunStatusFields: v1beta1.CustomRunStatusFields{
				StartTime: &metav1.Time{Time: cr00Started},
			},
		},
	}

	cr01 := v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "abc",
			Name:      "cr0-1",
		},
		Status: v1beta1.CustomRunStatus{
			CustomRunStatusFields: v1beta1.CustomRunStatusFields{
				StartTime: &metav1.Time{Time: cr01Started},
			},
		},
	}

	cr10 := v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "def",
			Name:      "cr1-0",
		},
		Status: v1beta1.CustomRunStatus{
			CustomRunStatusFields: v1beta1.CustomRunStatusFields{
				StartTime: &metav1.Time{Time: cr10Started},
			},
		},
	}

	cr11 := v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "def",
			Name:      "cr1-1",
		},
		Status: v1beta1.CustomRunStatus{
			CustomRunStatusFields: v1beta1.CustomRunStatusFields{
				StartTime: &metav1.Time{Time: cr11Started},
			},
		},
	}

	cr20 := v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ghi",
			Name:      "cr2-0",
		},
		Status: v1beta1.CustomRunStatus{
			CustomRunStatusFields: v1beta1.CustomRunStatusFields{
				StartTime: &metav1.Time{Time: cr20Started},
			},
		},
	}

	cr21 := v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ghi",
			Name:      "cr2-1",
		},
		Status: v1beta1.CustomRunStatus{
			CustomRunStatusFields: v1beta1.CustomRunStatusFields{
				StartTime: &metav1.Time{Time: cr21Started},
			},
		},
	}

	crs := []v1beta1.CustomRun{cr00, cr01, cr10, cr11, cr20, cr21}

	SortByNamespace(crs)

	element1 := crs[0].Name
	if element1 != "cr0-0" {
		t.Errorf("SortCustomRunsByNamespaceWithStartTime should be cr0-0 but returned: %s", element1)
	}

	element2 := crs[1].Name
	if element2 != "cr0-1" {
		t.Errorf("SortCustomRunsByNamespaceWithStartTime should be cr0-1 but returned: %s", element2)
	}

	element3 := crs[2].Name
	if element3 != "cr1-0" {
		t.Errorf("SortCustomRunsByNamespaceWithStartTime should be cr1-0 but returned: %s", element3)
	}

	element4 := crs[3].Name
	if element4 != "cr1-1" {
		t.Errorf("SortCustomRunsByNamespaceWithStartTime should be cr1-1 but returned: %s", element4)
	}

	element5 := crs[4].Name
	if element5 != "cr2-0" {
		t.Errorf("SortCustomRunsByNamespaceWithStartTime should be cr2-0 but returned: %s", element5)
	}

	element6 := crs[5].Name
	if element6 != "cr2-1" {
		t.Errorf("SortCustomRunsByNamespaceWithStartTime should be cr2-1 but returned: %s", element6)
	}
}
