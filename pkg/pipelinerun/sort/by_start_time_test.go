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

func Test_PipelineRunsByStartTime(t *testing.T) {
	clock := test.FakeClock()

	pr0Started := clock.Now().Add(10 * time.Second)
	pr1Started := clock.Now().Add(-2 * time.Hour)
	pr2Started := clock.Now().Add(-1 * time.Hour)

	pr1 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "pr0-1",
		},
	}
	pr1.Status.StartTime = &metav1.Time{Time: pr0Started}

	pr2 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "pr1-1",
		},
	}
	pr2.Status.StartTime = &metav1.Time{Time: pr1Started}

	pr3 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "pr2-1",
		},
	}
	pr3.Status.StartTime = &metav1.Time{Time: pr2Started}

	prs := []v1.PipelineRun{
		pr2,
		pr3,
		pr1,
	}

	SortByStartTime(prs)

	element1 := prs[0].Name
	if element1 != "pr0-1" {
		t.Errorf("SortPipelineRunsByStartTime should be pr0-1 but returned: %s", element1)
	}

	element2 := prs[1].Name
	if element2 != "pr2-1" {
		t.Errorf("SortPipelineRunsByStartTime should be pr2-1 but returned: %s", element2)
	}

	element3 := prs[2].Name
	if element3 != "pr1-1" {
		t.Errorf("SortPipelineRunsByStartTime should be pr1-1 but returned: %s", element3)
	}
}

func Test_PipelineRunsByStartTime_NilStartTime(t *testing.T) {

	pr1 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "pr0-1",
		},
	}

	pr2 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "pr1-1",
		},
	}

	pr3 := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "pr2-1",
		},
	}

	prs := []v1.PipelineRun{
		pr2,
		pr3,
		pr1,
	}

	SortByStartTime(prs)

	element1 := prs[0].Name
	if element1 != "pr1-1" {
		t.Errorf("SortPipelineRunsByStartTime should be pr1-1 but returned: %s", element1)
	}

	element2 := prs[1].Name
	if element2 != "pr2-1" {
		t.Errorf("SortPipelineRunsByStartTime should be pr2-1 but returned: %s", element2)
	}

	element3 := prs[2].Name
	if element3 != "pr0-1" {
		t.Errorf("SortPipelineRunsByStartTime should be pr0-1 but returned: %s", element3)
	}
}
