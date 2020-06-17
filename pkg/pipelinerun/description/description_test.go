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

package description

import (
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func TestPipelineRefExists_Present(t *testing.T) {
	spec := v1beta1.PipelineRunSpec{
		PipelineRef: &v1beta1.PipelineRef{
			Name: "Pipeline",
		},
	}

	output := pipelineRefExists(spec)
	test.AssertOutput(t, "Pipeline", output)
}

func TestPipelineRefExists_Not_Present(t *testing.T) {
	spec := v1beta1.PipelineRunSpec{
		PipelineRef: nil,
	}

	output := pipelineRefExists(spec)
	test.AssertOutput(t, "", output)
}

func TestPipelineResourceRefExists_Present(t *testing.T) {
	spec := v1beta1.PipelineResourceBinding{
		ResourceRef: &v1beta1.PipelineResourceRef{
			Name: "Pipeline",
		},
	}

	output := pipelineResourceRefExists(spec)
	test.AssertOutput(t, "Pipeline", output)
}

func TestPipelineResourceRefExists_Not_Present(t *testing.T) {
	spec := v1beta1.PipelineResourceBinding{
		ResourceRef: nil,
	}

	output := pipelineResourceRefExists(spec)
	test.AssertOutput(t, "", output)
}
