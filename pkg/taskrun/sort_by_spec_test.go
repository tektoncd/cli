// Copyright © 2026 The Tekton Authors.
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

	"github.com/google/go-cmp/cmp"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func trStatus(task string, start, end time.Time) *v1.PipelineRunTaskRunStatus {
	return &v1.PipelineRunTaskRunStatus{
		PipelineTaskName: task,
		Status: &v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{
				StartTime:      &metav1.Time{Time: start},
				CompletionTime: &metav1.Time{Time: end},
			},
		},
	}
}

func names(runs []Run) []string {
	var n []string
	for _, r := range runs {
		n = append(n, r.Name)
	}
	return n
}

func TestSortTasksBySpecOrderKeepsMatrixTaskRuns(t *testing.T) {
	now := time.Now()
	tasks := []v1.PipelineTask{{Name: "build"}, {Name: "test", Retries: 2}}
	trs := map[string]*v1.PipelineRunTaskRunStatus{
		"pr-build":     trStatus("build", now, now.Add(time.Minute)),
		"pr-test-2":    trStatus("test", now.Add(time.Minute), now.Add(4*time.Minute)),
		"pr-test-0":    trStatus("test", now.Add(time.Minute), now.Add(2*time.Minute)),
		"pr-test-1":    trStatus("test", now.Add(time.Minute), now.Add(3*time.Minute)),
		"pr-unrelated": trStatus("not-in-spec", now, now.Add(time.Minute)),
	}

	got := SortTasksBySpecOrder(tasks, trs)

	want := []string{"pr-build", "pr-test-0", "pr-test-1", "pr-test-2"}
	if d := cmp.Diff(want, names(got)); d != "" {
		t.Errorf("unexpected TaskRuns (-want +got):\n%s", d)
	}
	for _, r := range got[1:] {
		if r.Task != "test" || r.Retries != 2 {
			t.Errorf("run %s has task %q and retries %d, want test and 2", r.Name, r.Task, r.Retries)
		}
	}
}

func TestSortTasksBySpecOrderStableOnTies(t *testing.T) {
	now := time.Now()
	tasks := []v1.PipelineTask{{Name: "matrix"}}
	trs := map[string]*v1.PipelineRunTaskRunStatus{}
	for _, n := range []string{"pr-matrix-3", "pr-matrix-1", "pr-matrix-0", "pr-matrix-2"} {
		trs[n] = trStatus("matrix", now, now.Add(time.Minute))
	}

	// same times for every combination, so the order comes from the names
	want := []string{"pr-matrix-0", "pr-matrix-1", "pr-matrix-2", "pr-matrix-3"}
	for i := 0; i < 5; i++ {
		if d := cmp.Diff(want, names(SortTasksBySpecOrder(tasks, trs))); d != "" {
			t.Fatalf("unexpected order (-want +got):\n%s", d)
		}
	}
}

func TestSortTasksBySpecOrderWithoutStatus(t *testing.T) {
	tasks := []v1.PipelineTask{{Name: "build"}}
	trs := map[string]*v1.PipelineRunTaskRunStatus{
		"pr-build": {PipelineTaskName: "build"},
	}
	got := SortTasksBySpecOrder(tasks, trs)
	if len(got) != 1 || got[0].Name != "pr-build" || got[0].StartTime != nil {
		t.Errorf("unexpected runs %+v", got)
	}
}
