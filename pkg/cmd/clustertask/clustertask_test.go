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

package clustertask

import (
	"testing"

	"github.com/tektoncd/cli/pkg/test"
)

func TestClusterTask_invalid(t *testing.T) {

	p := &test.Params{}

	clustertask := Command(p)
	out, err := test.ExecuteCommand(clustertask, "foobar")
	if err == nil {
		t.Errorf("No errors was defined. Output: %s", out)
	}
}
