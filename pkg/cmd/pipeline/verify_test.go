// Copyright © 2022 The Tekton Authors.
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
	"os"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
)

func TestVerify(t *testing.T) {
	p := &test.Params{}

	pipeline := Command(p)

	os.Setenv("PRIVATE_PASSWORD", "1234")

	out, err := test.ExecuteCommand(pipeline, "verify", "testdata/signed.yaml", "-K", "testdata/cosign.pub")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := "*Warning*: This is an experimental command, its usage and behavior can change in the next release(s)\nPipeline testdata/signed.yaml passes verification \n"
	test.AssertOutput(t, expected, out)
}
