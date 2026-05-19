// Copyright © 2021 The Tekton Authors.
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

package checkupgrade

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	res "github.com/tektoncd/cli/pkg/cmd/hub/gen/resource"
	"github.com/tektoncd/cli/pkg/cmd/hub/hub"
	"github.com/tektoncd/cli/pkg/cmd/hub/test"
	cb "github.com/tektoncd/cli/pkg/cmd/hub/test/builder"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"gopkg.in/h2non/gock.v1"
	"gotest.tools/v3/golden"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var resVersion = &res.ResourceData{
	ID:   1,
	Name: "foo",
	Kind: "Task",
	Catalog: &res.Catalog{
		ID:   1,
		Name: "tekton",
		Type: "community",
	},
	Rating: 4.8,
	LatestVersion: &res.ResourceVersionData{
		ID:                  12,
		Version:             "0.2",
		Description:         "v0.1 Task to run foo",
		DisplayName:         "foo-bar",
		MinPipelinesVersion: "0.11",
		RawURL:              "http://raw.github.url/foo/0.1/foo.yaml",
		WebURL:              "http://web.github.com/foo/0.1/foo.yaml",
		UpdatedAt:           "2020-01-01 12:00:00 +0000 UTC",
	},
	Tags: []*res.Tag{
		{
			ID:   3,
			Name: "cli",
		},
	},
	Versions: []*res.ResourceVersionData{
		{
			ID:      12,
			Version: "0.2",
		},
	},
}

func TestUpdateAvailable(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	resource := &res.Resource{Data: resVersion}
	res := res.NewViewedResource(resource, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo").
		MatchParam("pipelinesversion", "0.14").
		Reply(200).
		JSON(&res.Projected)

	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	existingTask := &v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
			Labels: map[string]string{
				"hub.tekton.dev/catalog":    "tekton",
				"app.kubernetes.io/version": "0.1",
			}},
	}

	version := "v1beta1"
	dynamic := test.DynamicClient(cb.UnstructuredV1beta1T(existingTask, version))

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: []*v1beta1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
	}

	err := test.CreateTektonPipelineController(dynamic, "v0.14.0")
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	err = opts.run()
	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), true)
}

func TestV1UpdateAvailable(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	resource := &res.Resource{Data: resVersion}
	res := res.NewViewedResource(resource, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo").
		MatchParam("pipelinesversion", "0.14").
		Reply(200).
		JSON(&res.Projected)

	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	existingTask := &v1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
			Labels: map[string]string{
				"hub.tekton.dev/catalog":    "tekton",
				"app.kubernetes.io/version": "0.1",
			}},
	}

	version := "v1"
	dynamic := test.DynamicClient(cb.UnstructuredV1(existingTask, version))

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: []*v1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
	}

	err := test.CreateTektonPipelineController(dynamic, "v0.14.0")
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	err = opts.run()
	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), true)
}

func TestUpdateAvailable_WithSkippedTasks(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	resource := &res.Resource{Data: resVersion}
	res := res.NewViewedResource(resource, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo").
		MatchParam("pipelinesversion", "0.14").
		Reply(200).
		JSON(&res.Projected)

	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	existingTasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "hub",
				Labels: map[string]string{
					"hub.tekton.dev/catalog":    "tekton",
					"app.kubernetes.io/version": "0.1",
				}},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-bar",
				Namespace: "hub",
				Labels: map[string]string{
					"app.kubernetes.io/version": "0.1",
				}},
		},
	}

	version := "v1beta1"
	dynamic := test.DynamicClient(cb.UnstructuredV1beta1T(existingTasks[0], version), cb.UnstructuredV1beta1T(existingTasks[1], version))

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: existingTasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
	}

	err := test.CreateTektonPipelineController(dynamic, "v0.14.0")
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	err = opts.run()
	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), true)
}

func TestV1UpdateAvailable_WithSkippedTasks(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	resource := &res.Resource{Data: resVersion}
	res := res.NewViewedResource(resource, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo").
		MatchParam("pipelinesversion", "0.14").
		Reply(200).
		JSON(&res.Projected)

	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	existingTasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "hub",
				Labels: map[string]string{
					"hub.tekton.dev/catalog":    "tekton",
					"app.kubernetes.io/version": "0.1",
				}},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-bar",
				Namespace: "hub",
				Labels: map[string]string{
					"app.kubernetes.io/version": "0.1",
				}},
		},
	}

	version := "v1"
	dynamic := test.DynamicClient(cb.UnstructuredV1(existingTasks[0], version), cb.UnstructuredV1(existingTasks[1], version))

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: existingTasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
	}

	err := test.CreateTektonPipelineController(dynamic, "v0.14.0")
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	err = opts.run()
	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), true)
}

func TestNoUpdateAvailable(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	resource := &res.Resource{Data: resVersion}
	res := res.NewViewedResource(resource, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo").
		MatchParam("pipelinesversion", "0.14").
		Reply(200).
		JSON(&res.Projected)

	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	existingTask := &v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
			Labels: map[string]string{
				"hub.tekton.dev/catalog":    "tekton",
				"app.kubernetes.io/version": "0.2",
			}},
	}

	version := "v1beta1"
	dynamic := test.DynamicClient(cb.UnstructuredV1beta1T(existingTask, version))

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: []*v1beta1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
	}

	err := test.CreateTektonPipelineController(dynamic, "v0.14.0")
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	err = opts.run()
	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), true)
}

func TestV1NoUpdateAvailable(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	resource := &res.Resource{Data: resVersion}
	res := res.NewViewedResource(resource, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo").
		MatchParam("pipelinesversion", "0.14").
		Reply(200).
		JSON(&res.Projected)

	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	existingTask := &v1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
			Labels: map[string]string{
				"hub.tekton.dev/catalog":    "tekton",
				"app.kubernetes.io/version": "0.2",
			}},
	}

	version := "v1"
	dynamic := test.DynamicClient(cb.UnstructuredV1(existingTask, version))

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: []*v1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
	}

	err := test.CreateTektonPipelineController(dynamic, "v0.14.0")
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	err = opts.run()
	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), true)
}

func TestNoUpdateAvailable_TaskNotInstalledViaHubCLI(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	resource := &res.Resource{Data: resVersion}
	res := res.NewViewedResource(resource, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo").
		MatchParam("pipelinesversion", "0.14").
		Reply(200).
		JSON(&res.Projected)

	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	existingTask := &v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
			Labels: map[string]string{
				"app.kubernetes.io/version": "0.1",
			}},
	}

	version := "v1beta1"
	dynamic := test.DynamicClient(cb.UnstructuredV1beta1T(existingTask, version))

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: []*v1beta1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
	}

	err := test.CreateTektonPipelineController(dynamic, "v0.14.0")
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	err = opts.run()
	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), false)
}

func TestV1NoUpdateAvailable_TaskNotInstalledViaHubCLI(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	resource := &res.Resource{Data: resVersion}
	res := res.NewViewedResource(resource, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo").
		MatchParam("pipelinesversion", "0.14").
		Reply(200).
		JSON(&res.Projected)

	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	existingTask := &v1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
			Labels: map[string]string{
				"app.kubernetes.io/version": "0.1",
			}},
	}

	version := "v1"
	dynamic := test.DynamicClient(cb.UnstructuredV1(existingTask, version))

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: []*v1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
	}

	err := test.CreateTektonPipelineController(dynamic, "v0.14.0")
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	err = opts.run()
	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), false)
}

func TestUpdateAvailable_PipelinesUnknown(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	resource := &res.Resource{Data: resVersion}
	res := res.NewViewedResource(resource, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo").
		Reply(200).
		JSON(&res.Projected)

	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	existingTask := &v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
			Labels: map[string]string{
				"hub.tekton.dev/catalog":    "tekton",
				"app.kubernetes.io/version": "0.1",
			}},
	}

	version := "v1beta1"
	dynamic := test.DynamicClient(cb.UnstructuredV1beta1T(existingTask, version))

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: []*v1beta1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
	}

	err := opts.run()
	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), true)
}

func TestV1UpdateAvailable_PipelinesUnknown(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	resource := &res.Resource{Data: resVersion}
	res := res.NewViewedResource(resource, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo").
		Reply(200).
		JSON(&res.Projected)

	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	existingTask := &v1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
			Labels: map[string]string{
				"hub.tekton.dev/catalog":    "tekton",
				"app.kubernetes.io/version": "0.1",
			}},
	}

	version := "v1"
	dynamic := test.DynamicClient(cb.UnstructuredV1(existingTask, version))

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: []*v1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
	}

	err := opts.run()
	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), true)
}

func TestUpdateAvailable_WithSkippedTasks_PipelinesUnknown(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	resource := &res.Resource{Data: resVersion}
	res := res.NewViewedResource(resource, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo").
		Reply(200).
		JSON(&res.Projected)

	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	existingTasks := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "hub",
				Labels: map[string]string{
					"hub.tekton.dev/catalog":    "tekton",
					"app.kubernetes.io/version": "0.1",
				}},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-bar",
				Namespace: "hub",
				Labels: map[string]string{
					"app.kubernetes.io/version": "0.1",
				}},
		},
	}

	version := "v1beta1"
	dynamic := test.DynamicClient(cb.UnstructuredV1beta1T(existingTasks[0], version), cb.UnstructuredV1beta1T(existingTasks[1], version))

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: existingTasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
	}

	err := opts.run()
	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), true)
}

func TestV1UpdateAvailable_WithSkippedTasks_PipelinesUnknown(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	resource := &res.Resource{Data: resVersion}
	res := res.NewViewedResource(resource, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo").
		Reply(200).
		JSON(&res.Projected)

	buf := new(bytes.Buffer)
	cli.SetStream(buf, buf)

	existingTasks := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "hub",
				Labels: map[string]string{
					"hub.tekton.dev/catalog":    "tekton",
					"app.kubernetes.io/version": "0.1",
				}},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-bar",
				Namespace: "hub",
				Labels: map[string]string{
					"app.kubernetes.io/version": "0.1",
				}},
		},
	}

	version := "v1"
	dynamic := test.DynamicClient(cb.UnstructuredV1(existingTasks[0], version), cb.UnstructuredV1(existingTasks[1], version))

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: existingTasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
	}

	err := opts.run()
	assert.NoError(t, err)
	golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden", t.Name()))
	assert.Equal(t, gock.IsDone(), true)
}
