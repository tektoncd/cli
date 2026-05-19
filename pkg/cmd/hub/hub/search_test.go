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

func TestEndpoint(t *testing.T) {

	opt := SearchOption{
		Match: "contains",
		Limit: 100,
	}

	url := opt.Endpoint()
	assert.Equal(t, "/v1/query?limit=100&match=contains", url)

	opt = SearchOption{
		Name:  "res",
		Kinds: []string{"k1", "k2", "k3"},
		Tags:  []string{"t1", "t2"},
		Match: "contains",
	}
	url = opt.Endpoint()
	assert.Equal(t, "/v1/query?kinds=k1&kinds=k2&kinds=k3&match=contains&name=res&tags=t1&tags=t2", url)
}
