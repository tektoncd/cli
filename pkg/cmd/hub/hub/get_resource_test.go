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

package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetResourceEndpoint(t *testing.T) {

	opt := ResourceOption{
		Version: "",
		Catalog: "tekton",
		Name:    "abc",
		Kind:    "task",
	}
	url := opt.Endpoint()
	assert.Equal(t, "/v1/resource/tekton/task/abc", url)

	opt.PipelineVersion = "0.17"
	url = opt.Endpoint()
	assert.Equal(t, "/v1/resource/tekton/task/abc?pipelinesversion=0.17", url)

	opt.Version = "0.1.1"
	url = opt.Endpoint()
	assert.Equal(t, "/v1/resource/tekton/task/abc/0.1.1", url)
}

func TestSortVersionsSemanticaly(t *testing.T) {
	// includes valid semver and some invalid strings (alpha/beta)
	in := []string{"0.9.0", "1.0.0", "v0.10.0", "alpha", "2.0.0", "1.2.3", "beta"}
	out := sortVersionsSemanticaly(in)

	// valid semvers should be in descending order first
	assert.Equal(t, []string{"2.0.0", "1.2.3", "1.0.0", "v0.10.0", "0.9.0", "beta", "alpha"}, out)
}
