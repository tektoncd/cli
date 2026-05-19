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

package get

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	res "github.com/tektoncd/cli/pkg/cmd/hub/gen/resource"
	"github.com/tektoncd/cli/pkg/cmd/hub/hub"
	"github.com/tektoncd/cli/pkg/cmd/hub/test"
	"gopkg.in/h2non/gock.v1"
	"gotest.tools/v3/golden"
)

var task1 = `---
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: foo
  annotations:
    tekton.dev/pipelines.minVersion: '0.13.1'
    tekton.dev/tags: cli
    tekton.dev/displayName: 'foo-bar'
spec:
  description: >-
    v0.3 Task to run foo
`

var taskWithNewVersionYaml = &res.ResourceContent{
	Yaml: &task1,
}

func TestGetTask_WithNewVersion(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	rVer := &res.ResourceVersionYaml{Data: taskWithNewVersionYaml}
	resWithVersion := res.NewViewedResourceVersionYaml(rVer, "default")

	resInfo := fmt.Sprintf("%s/%s/%s/%s", "tekton", "task", "foo", "0.3")

	gock.New(test.API).
		Get("/resource/" + resInfo + "/yaml").
		Reply(200).
		JSON(&resWithVersion.Projected)
	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	opts := taskOptions{
		options: &options{
			cli:     cli,
			kind:    "task",
			args:    []string{"foo"},
			from:    "tekton",
			version: "0.3",
		},
	}

	err := opts.run()
	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), true)
}
