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

package pipelineresource

import (
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	tu "github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
)

func TestPipelineResourceDelete_Empty(t *testing.T) {
	cs, _ := pipelinetest.SeedTestData(t, pipelinetest.Data{})
	p := &test.Params{Tekton: cs.Pipeline}

	res := Command(p)
	_, err := test.ExecuteCommand(res, "rm", "bar")
	if err == nil {
		t.Errorf("Error expected here")
	}
	expected := "Failed to delete pipelineresource \"bar\": pipelineresources.tekton.dev \"bar\" not found"
	tu.AssertOutput(t, expected, err.Error())
}

func TestPipelineResourceDelete_WithParams(t *testing.T) {
	pres := []*v1alpha1.PipelineResource{
		tb.PipelineResource("test-1", "test-ns-1",
			tb.PipelineResourceSpec("image",
				tb.PipelineResourceSpecParam("URL", "quay.io/tekton/controller"),
			),
		),
	}

	cs, _ := pipelinetest.SeedTestData(t, pipelinetest.Data{PipelineResources: pres})
	p := &test.Params{Tekton: cs.Pipeline}
	pipelineresource := Command(p)
	out, _ := test.ExecuteCommand(pipelineresource, "rm", "test-1", "-n", "test-ns-1")
	expected := "PipelineResource deleted: test-1\n"
	tu.AssertOutput(t, expected, out)
}
