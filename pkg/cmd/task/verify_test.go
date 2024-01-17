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

package task

import (
	"fmt"
	"os"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
)

func TestVerify(t *testing.T) {
	p := &test.Params{}
	task := Command(p)
	os.Setenv("PRIVATE_PASSWORD", "1234")

	testcases := []struct {
		name       string
		taskFile   string
		apiVersion string
	}{{
		name:       "verify v1beta1 Task",
		taskFile:   "testdata/signed.yaml",
		apiVersion: "v1beta1",
	}, {
		name:       "verify v1 Task",
		taskFile:   "testdata/signed-v1.yaml",
		apiVersion: "v1",
	}}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := test.ExecuteCommand(task, "verify", tc.taskFile, "-K", "testdata/cosign.pub", "--api-version", tc.apiVersion)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			expected := fmt.Sprintf("*Warning*: This is an experimental command, it's usage and behavior can change in the next release(s)\nTask %s passes verification \n", tc.taskFile)
			test.AssertOutput(t, expected, out)
		})
	}
}
