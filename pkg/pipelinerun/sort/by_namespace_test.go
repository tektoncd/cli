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

package pipelinerun

import (
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_PipelineRunsByNamespace(t *testing.T) {
	pr1 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "abc",
			Name:      "pr0-1",
		},
	}

	pr2 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "def",
			Name:      "pr1-1",
		},
	}

	pr3 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ghi",
			Name:      "pr2-1",
		},
	}

	prs := []v1.PipelineRun{
		pr2,
		pr3,
		pr1,
	}

	SortByNamespace(prs)

	element1 := prs[0].Name
	if element1 != "pr0-1" {
		t.Errorf("SortPipelineRunsByNamespace should be pr0-1 but returned: %s", element1)
	}

	element2 := prs[1].Name
	if element2 != "pr1-1" {
		t.Errorf("SortPipelineRunsByNamespace should be pr1-1 but returned: %s", element2)
	}

	element3 := prs[2].Name
	if element3 != "pr2-1" {
		t.Errorf("SortPipelineRunsByNamespace should be pr2-1 but returned: %s", element3)
	}
}

func Test_PipelineRunsByNamespaceWithStartTime(t *testing.T) {

	clock := test.FakeClock()

	pr00Started := clock.Now().Add(10 * time.Second)
	pr01Started := clock.Now().Add(-1 * time.Hour)
	pr10Started := clock.Now().Add(10 * time.Second)
	pr11Started := clock.Now().Add(-1 * time.Hour)
	pr20Started := clock.Now().Add(10 * time.Second)
	pr21Started := clock.Now().Add(-1 * time.Hour)

	pr00 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "abc",
			Name:      "pr0-0",
		},
	}

	pr00.Status.StartTime = &metav1.Time{Time: pr00Started}

	pr01 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "abc",
			Name:      "pr0-1",
		},
	}

	pr01.Status.StartTime = &metav1.Time{Time: pr01Started}

	pr10 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "def",
			Name:      "pr1-0",
		},
	}

	pr10.Status.StartTime = &metav1.Time{Time: pr10Started}

	pr11 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "def",
			Name:      "pr1-1",
		},
	}

	pr11.Status.StartTime = &metav1.Time{Time: pr11Started}

	pr20 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ghi",
			Name:      "pr2-0",
		},
	}

	pr20.Status.StartTime = &metav1.Time{Time: pr20Started}

	pr21 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ghi",
			Name:      "pr2-1",
		},
	}

	pr21.Status.StartTime = &metav1.Time{Time: pr21Started}

	prs := []v1.PipelineRun{
		pr11,
		pr21,
		pr01,
		pr10,
		pr20,
		pr00,
	}

	SortByNamespace(prs)

	element1 := prs[0].Name
	if element1 != "pr0-0" {
		t.Errorf("SortPipelineRunsByNamespaceWithStartTime should be pr0-0 but returned: %s", element1)
	}

	element2 := prs[1].Name
	if element2 != "pr0-1" {
		t.Errorf("SortPipelineRunsByNamespaceWithStartTime should be pr0-1 but returned: %s", element2)
	}

	element3 := prs[2].Name
	if element3 != "pr1-0" {
		t.Errorf("SortPipelineRunsByNamespaceWithStartTime should be pr1-0 but returned: %s", element3)
	}

	element4 := prs[3].Name
	if element4 != "pr1-1" {
		t.Errorf("SortPipelineRunsByNamespaceWithStartTime should be pr1-1 but returned: %s", element4)
	}

	element5 := prs[4].Name
	if element5 != "pr2-0" {
		t.Errorf("SortPipelineRunsByNamespaceWithStartTime should be pr2-0 but returned: %s", element5)
	}

	element6 := prs[5].Name
	if element6 != "pr2-1" {
		t.Errorf("SortPipelineRunsByNamespaceWithStartTime should be pr2-1 but returned: %s", element6)
	}
}
