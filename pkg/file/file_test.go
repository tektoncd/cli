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

package file

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"gotest.tools/v3/assert"
)

func TestLoadLocalFile(t *testing.T) {
	localTarget := "./testdata/task.yaml"
	localContent, err := ioutil.ReadFile(localTarget)
	if err != nil {
		t.Errorf("file path %s does not exist", localTarget)
	}

	var taskExpected v1alpha1.Task
	err = yaml.Unmarshal(localContent, &taskExpected)
	if err != nil {
		t.Errorf("Error from unmarshal of local file")
	}

	p := &test.Params{}
	localContentLoad, err := LoadFileContent(p, localTarget, IsYamlFile(), fmt.Errorf("invalid file format for %s: .yaml or .yml file extension and format required", localTarget))
	if err != nil {
		t.Errorf("Error from running localContentLoad")
	}

	var taskResult v1alpha1.Task
	err = yaml.Unmarshal(localContentLoad, &taskResult)
	if err != nil {
		t.Errorf("Error from unmarshal after running LoadFileContent")
	}

	if d := cmp.Diff(taskExpected, taskResult); d != "" {
		t.Errorf("Unexpected output mismatch: %s", d)
	}
}

func TestGetError(t *testing.T) {
	p := &test.Params{}
	target := "httpz://foo.com/task.yaml"

	_, err := LoadFileContent(p, target, IsYamlFile(), fmt.Errorf("invalid file format for %s: .yaml or .yml file extension and format required", target))
	assert.Error(t, err, `Get httpz://foo.com/task.yaml: unsupported protocol scheme "httpz"`)
}
