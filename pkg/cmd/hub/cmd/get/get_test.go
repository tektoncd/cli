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
	"github.com/tektoncd/cli/pkg/cmd/hub/app"
	res "github.com/tektoncd/cli/pkg/cmd/hub/gen/resource"
	"github.com/tektoncd/cli/pkg/cmd/hub/hub"
	"github.com/tektoncd/cli/pkg/cmd/hub/test"
	goa "goa.design/goa/v3/pkg"
	"gopkg.in/h2non/gock.v1"
	"gotest.tools/v3/golden"
)

var pipeline1 = `---
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: mango
  annotations:
    tekton.dev/pipelines.minVersion: '0.18'
    tekton.dev/tags: fruit
    tekton.dev/displayName: 'Alphanso'
spec:
  description: >-
    v0.3 of Pipeline mango
`

var pipelineWithLatestVersionYaml = &res.ResourceContent{
	Yaml: &pipeline1,
}

var pipeline2 = `---
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: mango
  annotations:
    tekton.dev/pipelines.minVersion: '0.18'
    tekton.dev/tags: fruit
    tekton.dev/displayName: 'Alphanso'
spec:
  description: >-
    v0.2 of Pipeline mango
`

var pipelineWithOldVersionYaml = &res.ResourceContent{
	Yaml: &pipeline2,
}

var want = `
Get a Abc of name 'foo':

    tkn hub get abc foo

or

Get a Abc of name 'foo' of version '0.3':

    tkn hub get abc foo --version 0.3
`

func TestValidate(t *testing.T) {
	cli := app.New()
	if err := cli.SetHub(hub.TektonHubType); err != nil {
		assert.Error(t, err)
	}

	opt := options{
		version: "0.1",
		cli:     cli,
	}
	err := opt.validate()
	assert.NoError(t, err)

	opt = options{
		version: "0.3.1",
		cli:     cli,
	}
	err = opt.validate()
	assert.NoError(t, err)
}

func TestValidate_ErrorCase(t *testing.T) {
	cli := app.New()
	if err := cli.SetHub(hub.TektonHubType); err != nil {
		assert.Error(t, err)
	}

	opt := options{
		version: "abc",
		cli:     cli,
	}
	err := opt.validate()
	assert.EqualError(t, err, "invalid value \"abc\" set for option version. valid eg. 0.1, 1.2.1")
}

func TestGetResource_WithNewVersion(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	rVer := &res.ResourceVersionYaml{Data: pipelineWithLatestVersionYaml}
	resWithVersion := res.NewViewedResourceVersionYaml(rVer, "default")

	resInfo := fmt.Sprintf("%s/%s/%s/%s", "fruit", "pipeline", "mango", "0.3")

	gock.New(test.API).
		Get("/resource/" + resInfo + "/yaml").
		Reply(200).
		JSON(&resWithVersion.Projected)
	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	opt := &options{
		cli:     cli,
		kind:    "pipeline",
		args:    []string{"mango"},
		from:    "fruit",
		version: "0.3",
	}

	err := opt.run()

	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), true)
}

func TestGetResource_WithOldVersion(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	rVer := &res.ResourceVersionYaml{Data: pipelineWithOldVersionYaml}
	resWithVersion := res.NewViewedResourceVersionYaml(rVer, "default")

	resInfo := fmt.Sprintf("%s/%s/%s/%s", "fruit", "pipeline", "mango", "0.2")

	gock.New(test.API).
		Get("/resource/" + resInfo + "/yaml").
		Reply(200).
		JSON(&resWithVersion.Projected)
	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	opt := &options{
		cli:     cli,
		kind:    "pipeline",
		args:    []string{"mango"},
		from:    "fruit",
		version: "0.2",
	}

	err := opt.run()
	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), true)
}

func TestGet_ResourceNotFound(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	gock.New(test.API).
		Get("/resource/tekton/pipeline/xyz").
		Reply(404).
		JSON(&goa.ServiceError{
			ID:      "123456",
			Name:    "not-found",
			Message: "resource not found",
		})

	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	opt := &options{
		cli:  cli,
		kind: "pipeline",
		args: []string{"xyz"},
		from: "tekton",
	}

	err := opt.run()
	assert.Error(t, err)
	assert.EqualError(t, err, "No Resource Found")
	assert.Equal(t, gock.IsDone(), true)
}

func Test_examples(t *testing.T) {
	got := examples("abc")
	assert.Equal(t, want, got)
}
