// Copyright © 2019 The Tekton Authors.
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

package task

import (
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
)

func TestTask_GetAllTaskNames(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()
	tdata := []*v1alpha1.Task{
		tb.Task("task", "ns",
			// created  5 minutes back
			cb.TaskCreationTime(clock.Now().Add(-5*time.Minute)),
		),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks: tdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredT(tdata[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic clinet: %v", err)
	}

	tdata2 := []*v1alpha1.Task{
		tb.Task("task", "ns",
			// created  5 minutes back
			cb.TaskCreationTime(clock.Now().Add(-5*time.Minute)),
		),
		tb.Task("task2", "ns",
			// created  5 minutes back
			cb.TaskCreationTime(clock.Now().Add(-5*time.Minute)),
		),
	}
	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks: tdata2,
	})
	cs2.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredT(tdata2[0], version),
		cb.UnstructuredT(tdata2[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic clinet: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}
	p3 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}
	p3.SetNamespace("unknown")

	testParams := []struct {
		name   string
		params *test.Params
		want   []string
	}{
		{
			name:   "Single Task",
			params: p,
			want:   []string{"task"},
		},
		{
			name:   "Multi Tasks",
			params: p2,
			want:   []string{"task", "task2"},
		},
		{
			name:   "Unknown namespace",
			params: p3,
			want:   []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllTaskNames(tp.params)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}
