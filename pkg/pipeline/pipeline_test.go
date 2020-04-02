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

package pipeline

import (
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
)

func TestPipelinesList_with_single_run(t *testing.T) {
	clock := clockwork.NewFakeClock()

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline("pipeline", "ns",
				// created  5 minutes back
				cb.PipelineCreationTimestamp(clock.Now().Add(-5*time.Minute)),
			),
		},
	})

	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline("pipeline", "ns",
				// created  5 minutes back
				cb.PipelineCreationTimestamp(clock.Now().Add(-5*time.Minute)),
			),
			tb.Pipeline("pipeline2", "ns",
				// created  5 minutes back
				cb.PipelineCreationTimestamp(clock.Now().Add(-5*time.Minute)),
			),
		},
	})

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube}
	p2 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube}
	p3 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube}
	p3.SetNamespace("unknown")

	testParams := []struct {
		name   string
		params *test.Params
		want   []string
	}{
		{
			name:   "Single Pipeline",
			params: p,
			want:   []string{"pipeline"},
		},
		{
			name:   "Multi Pipelines",
			params: p2,
			want:   []string{"pipeline", "pipeline2"},
		},
		{
			name:   "Unknown namespace",
			params: p3,
			want:   []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllPipelineNames(tp.params)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}
