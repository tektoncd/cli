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

package upgrade

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var resVersion = &res.ResourceVersionData{
	ID:                  11,
	Version:             "0.3",
	DisplayName:         "foo-bar",
	Description:         "v0.3 Task to run foo",
	MinPipelinesVersion: "0.12",
	RawURL:              "http://raw.github.url/foo/0.3/foo.yaml",
	WebURL:              "http://web.github.com/foo/0.3/foo.yaml",
	UpdatedAt:           "2020-01-01 12:00:00 +0000 UTC",
	Resource: &res.ResourceData{
		ID:   1,
		Name: "foo",
		Kind: "Task",
		Catalog: &res.Catalog{
			ID:   1,
			Name: "tekton",
			Type: "community",
		},
		Rating: 4.8,
		Tags: []*res.Tag{
			{
				ID:   3,
				Name: "cli",
			},
		},
	},
}

var task1 = `---
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: foo
  labels:
    app.kubernetes.io/version: '0.3'
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

var task2 = `---
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: foo
  labels:
    app.kubernetes.io/version: '0.3'
  annotations:
    tekton.dev/pipelines.minVersion: '0.13.1'
    tekton.dev/tags: cli
    tekton.dev/displayName: 'foo-bar'
spec:
  description: >-
    v0.3 Task to run foo
`

var taskV1WithNewVersionYaml = &res.ResourceContent{
	Yaml: &task2,
}

func TestUpgrade_ResourceNotExist(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	version := "v1beta1"
	dynamic := test.DynamicClient()

	cs, _ := test.SeedV1beta1TestData(t, test.Data{})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
		args: []string{"foo"},
	}

	err := opts.run()
	assert.Error(t, err)
	assert.EqualError(t, err, "Task foo doesn't exist in hub namespace. Use install command to install the task")
}

func TestV1Upgrade_ResourceNotExist(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	version := "v1"
	dynamic := test.DynamicClient()

	cs, _ := test.SeedTestData(t, pipelinetest.Data{})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
		args: []string{"foo"},
	}

	err := opts.run()
	assert.Error(t, err)
	assert.EqualError(t, err, "Task foo doesn't exist in hub namespace. Use install command to install the task")
}

func TestUpgrade_VersionCatalogMissing(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	existingTask := &v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
		},
	}

	version := "v1beta1"
	dynamic := test.DynamicClient(cb.UnstructuredV1beta1T(existingTask, version))

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: []*v1beta1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
		args: []string{"foo"},
	}

	err := opts.run()
	assert.Error(t, err)
	assert.EqualError(t, err, "Task foo seems to be missing version and catalog label. Use reinstall command to overwrite existing task")
}

func TestV1Upgrade_VersionCatalogMissing(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	existingTask := &v1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
		},
	}

	version := "v1"
	dynamic := test.DynamicClient(cb.UnstructuredV1(existingTask, version))

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: []*v1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
		args: []string{"foo"},
	}

	err := opts.run()
	assert.Error(t, err)
	assert.EqualError(t, err, "Task foo seems to be missing version and catalog label. Use reinstall command to overwrite existing task")
}

func TestUpgrade_VersionMissing(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	existingTask := &v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
			Labels:    map[string]string{"hub.tekton.dev/catalog": "abc"},
		},
	}

	version := "v1beta1"
	dynamic := test.DynamicClient(cb.UnstructuredV1beta1T(existingTask, version))

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: []*v1beta1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
		args: []string{"foo"},
	}

	err := opts.run()
	assert.Error(t, err)
	assert.EqualError(t, err, "Task foo seems to be missing version label. Use reinstall command to overwrite existing task")
}

func TestV1Upgrade_VersionMissing(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	existingTask := &v1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
			Labels:    map[string]string{"hub.tekton.dev/catalog": "abc"},
		},
	}

	version := "v1"
	dynamic := test.DynamicClient(cb.UnstructuredV1(existingTask, version))

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: []*v1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:   test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:  cli,
		kind: "task",
		args: []string{"foo"},
	}

	err := opts.run()
	assert.Error(t, err)
	assert.EqualError(t, err, "Task foo seems to be missing version label. Use reinstall command to overwrite existing task")
}

func TestUpgrade_ToSpecificVersion(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	rVer := &res.ResourceVersionYaml{Data: taskWithNewVersionYaml}
	resWithVersion := res.NewViewedResourceVersionYaml(rVer, "default")

	resInfo := fmt.Sprintf("%s/%s/%s/%s", "tekton", "task", "foo", "0.3")

	gock.New(test.API).
		Get("/resource/" + resInfo + "/yaml").
		Reply(200).
		JSON(&resWithVersion.Projected)

	resVersion := &res.ResourceVersion{Data: resVersion}
	resource := res.NewViewedResourceVersion(resVersion, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo/0.3").
		Reply(200).
		JSON(&resource.Projected)

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
		cs:      test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:     cli,
		kind:    "task",
		args:    []string{"foo"},
		version: "0.3",
	}

	err := opts.run()
	assert.NoError(t, err)
	assert.Equal(t, "WARN: tekton pipelines version unknown, this resource is compatible with pipelines min version v0.12\nTask foo upgraded to v0.3 in hub namespace\n", buf.String())
	assert.Equal(t, gock.IsDone(), true)
}

func TestV1Upgrade_ToSpecificVersion(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	rVer := &res.ResourceVersionYaml{Data: taskV1WithNewVersionYaml}
	resWithVersion := res.NewViewedResourceVersionYaml(rVer, "default")

	resInfo := fmt.Sprintf("%s/%s/%s/%s", "tekton", "task", "foo", "0.3")

	gock.New(test.API).
		Get("/resource/" + resInfo + "/yaml").
		Reply(200).
		JSON(&resWithVersion.Projected)

	resVersion := &res.ResourceVersion{Data: resVersion}
	resource := res.NewViewedResourceVersion(resVersion, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo/0.3").
		Reply(200).
		JSON(&resource.Projected)

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
		cs:      test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:     cli,
		kind:    "task",
		args:    []string{"foo"},
		version: "0.3",
	}

	err := opts.run()
	assert.NoError(t, err)
	assert.Equal(t, "WARN: tekton pipelines version unknown, this resource is compatible with pipelines min version v0.12\nTask foo upgraded to v0.3 in hub namespace\n", buf.String())
	assert.Equal(t, gock.IsDone(), true)
}

func TestUpgrade_SameVersionError(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	rVer := &res.ResourceVersionYaml{Data: taskWithNewVersionYaml}
	resWithVersion := res.NewViewedResourceVersionYaml(rVer, "default")

	resInfo := fmt.Sprintf("%s/%s/%s/%s", "tekton", "task", "foo", "0.3")

	gock.New(test.API).
		Get("/resource/" + resInfo + "/yaml").
		Reply(200).
		JSON(&resWithVersion.Projected)

	resVersion := &res.ResourceVersion{Data: resVersion}
	resource := res.NewViewedResourceVersion(resVersion, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo/0.3").
		Reply(200).
		JSON(&resource.Projected)

	existingTask := &v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
			Labels: map[string]string{
				"hub.tekton.dev/catalog":    "tekton",
				"app.kubernetes.io/version": "0.3",
			}},
	}

	version := "v1beta1"
	dynamic := test.DynamicClient(cb.UnstructuredV1beta1T(existingTask, version))

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: []*v1beta1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:      test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:     cli,
		kind:    "task",
		args:    []string{"foo"},
		version: "0.3",
	}

	err := opts.run()
	assert.Error(t, err)
	assert.EqualError(t, err, "cannot upgrade task foo to v0.3. existing resource seems to be of same version. Use reinstall command to overwrite existing task")
	assert.Equal(t, gock.IsDone(), true)
}

func TestV1Upgrade_SameVersionError(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	rVer := &res.ResourceVersionYaml{Data: taskV1WithNewVersionYaml}
	resWithVersion := res.NewViewedResourceVersionYaml(rVer, "default")

	resInfo := fmt.Sprintf("%s/%s/%s/%s", "tekton", "task", "foo", "0.3")

	gock.New(test.API).
		Get("/resource/" + resInfo + "/yaml").
		Reply(200).
		JSON(&resWithVersion.Projected)

	resVersion := &res.ResourceVersion{Data: resVersion}
	resource := res.NewViewedResourceVersion(resVersion, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo/0.3").
		Reply(200).
		JSON(&resource.Projected)

	existingTask := &v1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
			Labels: map[string]string{
				"hub.tekton.dev/catalog":    "tekton",
				"app.kubernetes.io/version": "0.3",
			}},
	}

	version := "v1"
	dynamic := test.DynamicClient(cb.UnstructuredV1(existingTask, version))

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: []*v1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:      test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:     cli,
		kind:    "task",
		args:    []string{"foo"},
		version: "0.3",
	}

	err := opts.run()
	assert.Error(t, err)
	assert.EqualError(t, err, "cannot upgrade task foo to v0.3. existing resource seems to be of same version. Use reinstall command to overwrite existing task")
	assert.Equal(t, gock.IsDone(), true)
}

func TestUpgrade_LowerVersionError(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	rVer := &res.ResourceVersionYaml{Data: taskWithNewVersionYaml}
	resWithVersion := res.NewViewedResourceVersionYaml(rVer, "default")

	resInfo := fmt.Sprintf("%s/%s/%s/%s", "tekton", "task", "foo", "0.3")

	gock.New(test.API).
		Get("/resource/" + resInfo + "/yaml").
		Reply(200).
		JSON(&resWithVersion.Projected)

	resVersion := &res.ResourceVersion{Data: resVersion}
	resource := res.NewViewedResourceVersion(resVersion, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo/0.3").
		Reply(200).
		JSON(&resource.Projected)

	existingTask := &v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
			Labels: map[string]string{
				"hub.tekton.dev/catalog":    "tekton",
				"app.kubernetes.io/version": "0.7",
			}},
	}

	version := "v1beta1"
	dynamic := test.DynamicClient(cb.UnstructuredV1beta1T(existingTask, version))

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: []*v1beta1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:      test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:     cli,
		kind:    "task",
		args:    []string{"foo"},
		version: "0.3",
	}

	err := opts.run()
	assert.Error(t, err)
	assert.EqualError(t, err, "cannot upgrade task foo to v0.3. existing resource seems to be of higher version(v0.7). Use downgrade command")
	assert.Equal(t, gock.IsDone(), true)
}

func TestV1Upgrade_LowerVersionError(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	rVer := &res.ResourceVersionYaml{Data: taskV1WithNewVersionYaml}
	resWithVersion := res.NewViewedResourceVersionYaml(rVer, "default")

	resInfo := fmt.Sprintf("%s/%s/%s/%s", "tekton", "task", "foo", "0.3")

	gock.New(test.API).
		Get("/resource/" + resInfo + "/yaml").
		Reply(200).
		JSON(&resWithVersion.Projected)

	resVersion := &res.ResourceVersion{Data: resVersion}
	resource := res.NewViewedResourceVersion(resVersion, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo/0.3").
		Reply(200).
		JSON(&resource.Projected)

	existingTask := &v1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "hub",
			Labels: map[string]string{
				"hub.tekton.dev/catalog":    "tekton",
				"app.kubernetes.io/version": "0.7",
			}},
	}

	version := "v1"
	dynamic := test.DynamicClient(cb.UnstructuredV1(existingTask, version))

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: []*v1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	opts := &options{
		cs:      test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:     cli,
		kind:    "task",
		args:    []string{"foo"},
		version: "0.3",
	}

	err := opts.run()
	assert.Error(t, err)
	assert.EqualError(t, err, "cannot upgrade task foo to v0.3. existing resource seems to be of higher version(v0.7). Use downgrade command")
	assert.Equal(t, gock.IsDone(), true)
}

func TestUpgrade_ToSpecificVersion_RespectingPipelineSuccess(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	rVer := &res.ResourceVersionYaml{Data: taskWithNewVersionYaml}
	resWithVersion := res.NewViewedResourceVersionYaml(rVer, "default")

	resInfo := fmt.Sprintf("%s/%s/%s/%s", "tekton", "task", "foo", "0.3")

	gock.New(test.API).
		Get("/resource/" + resInfo + "/yaml").
		Reply(200).
		JSON(&resWithVersion.Projected)

	resVersion := &res.ResourceVersion{Data: resVersion}
	resource := res.NewViewedResourceVersion(resVersion, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo/0.3").
		Reply(200).
		JSON(&resource.Projected)

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
		cs:      test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:     cli,
		kind:    "task",
		args:    []string{"foo"},
		version: "0.3",
	}

	err := test.CreateTektonPipelineController(dynamic, "v0.14.0")
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	err = opts.run()
	assert.NoError(t, err)
	assert.Equal(t, "Task foo upgraded to v0.3 in hub namespace\n", buf.String())
	assert.Equal(t, gock.IsDone(), true)
}

func TestV1Upgrade_ToSpecificVersion_RespectingPipelineSuccess(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	rVer := &res.ResourceVersionYaml{Data: taskV1WithNewVersionYaml}
	resWithVersion := res.NewViewedResourceVersionYaml(rVer, "default")

	resInfo := fmt.Sprintf("%s/%s/%s/%s", "tekton", "task", "foo", "0.3")

	gock.New(test.API).
		Get("/resource/" + resInfo + "/yaml").
		Reply(200).
		JSON(&resWithVersion.Projected)

	resVersion := &res.ResourceVersion{Data: resVersion}
	resource := res.NewViewedResourceVersion(resVersion, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo/0.3").
		Reply(200).
		JSON(&resource.Projected)

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
		cs:      test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:     cli,
		kind:    "task",
		args:    []string{"foo"},
		version: "0.3",
	}

	err := test.CreateTektonPipelineController(dynamic, "v0.14.0")
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	err = opts.run()
	assert.NoError(t, err)
	assert.Equal(t, "Task foo upgraded to v0.3 in hub namespace\n", buf.String())
	assert.Equal(t, gock.IsDone(), true)
}

func TestUpgrade_ToSpecificVersion_RespectingPipelineFailure(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	rVer := &res.ResourceVersionYaml{Data: taskWithNewVersionYaml}
	resWithVersion := res.NewViewedResourceVersionYaml(rVer, "default")

	resInfo := fmt.Sprintf("%s/%s/%s/%s", "tekton", "task", "foo", "0.3")

	gock.New(test.API).
		Get("/resource/" + resInfo + "/yaml").
		Reply(200).
		JSON(&resWithVersion.Projected)

	resVersion := &res.ResourceVersion{Data: resVersion}
	resource := res.NewViewedResourceVersion(resVersion, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo/0.3").
		Reply(200).
		JSON(&resource.Projected)

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
		cs:      test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:     cli,
		kind:    "task",
		args:    []string{"foo"},
		version: "0.3",
	}

	err := test.CreateTektonPipelineController(dynamic, "v0.11.0")
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	err = opts.run()
	assert.Error(t, err)
	assert.EqualError(t, err, "cannot upgrade Task foo(0.3) as it requires Tekton Pipelines min version v0.12 but found v0.11.0")
	assert.Equal(t, gock.IsDone(), true)
}

func TestV1Upgrade_ToSpecificVersion_RespectingPipelineFailure(t *testing.T) {
	cli := test.NewCLI(hub.TektonHubType)

	defer gock.Off()

	rVer := &res.ResourceVersionYaml{Data: taskV1WithNewVersionYaml}
	resWithVersion := res.NewViewedResourceVersionYaml(rVer, "default")

	resInfo := fmt.Sprintf("%s/%s/%s/%s", "tekton", "task", "foo", "0.3")

	gock.New(test.API).
		Get("/resource/" + resInfo + "/yaml").
		Reply(200).
		JSON(&resWithVersion.Projected)

	resVersion := &res.ResourceVersion{Data: resVersion}
	resource := res.NewViewedResourceVersion(resVersion, "default")
	gock.New(test.API).
		Get("/resource/tekton/task/foo/0.3").
		Reply(200).
		JSON(&resource.Projected)

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
		cs:      test.FakeClientSet(cs.Pipeline, dynamic, "hub"),
		cli:     cli,
		kind:    "task",
		args:    []string{"foo"},
		version: "0.3",
	}

	err := test.CreateTektonPipelineController(dynamic, "v0.11.0")
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	err = opts.run()
	assert.Error(t, err)
	assert.EqualError(t, err, "cannot upgrade Task foo(0.3) as it requires Tekton Pipelines min version v0.12 but found v0.11.0")
	assert.Equal(t, gock.IsDone(), true)
}
