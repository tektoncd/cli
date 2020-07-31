// +build e2e
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

package clustertask

import (
	"testing"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	"github.com/tektoncd/cli/test/builder"
	"github.com/tektoncd/cli/test/cli"
	"github.com/tektoncd/cli/test/framework"
	"github.com/tektoncd/cli/test/helper"
	"github.com/tektoncd/cli/test/wait"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/icmd"
	knativetest "knative.dev/pkg/test"
)

const (
	tePipelineGitResourceName = "skaffold-git"
)

func TestClusterTaskInteractiveStartE2E(t *testing.T) {
	t.Parallel()
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer framework.TearDown(t, c, namespace)

	kubectl := cli.NewKubectl(namespace)
	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)

	t.Run("Get list of ClusterTasks when none present", func(t *testing.T) {
		res := tkn.Run("clustertask", "list")
		expected := "No ClusterTasks found\n"
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      expected,
		})
	})

	t.Logf("Creating clustertask read-clustertask")
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("read-file-clustertask.yaml"))

	t.Logf("Creating git pipeline resource in namespace: %s", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("git-resource.yaml"))

	t.Run("Get list of ClusterTasks", func(t *testing.T) {
		res := tkn.Run("clustertask", "list")
		expected := builder.ListAllClusterTasksOutput(t, c, map[int]interface{}{
			0: &builder.TaskData{
				Name: "read-clustertask",
			},
		})
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      expected,
		})
	})

	t.Run("Start ClusterTask with flags", func(t *testing.T) {
		res := tkn.MustSucceed(t, "clustertask", "start", "read-clustertask",
			"-i=source="+tePipelineGitResourceName,
			"-p=FILEPATH=docs",
			"-p=FILENAME=README.md",
			"-w=name=shared-workspace,emptyDir=",
			"--showlog")

		vars := make(map[string]interface{})
		taskRunGeneratedName := builder.GetTaskRunListWithName(c, "read-clustertask", true).Items[0].Name
		vars["Taskrun"] = taskRunGeneratedName
		expected := helper.ProcessString(`(TaskRun started: {{.Taskrun}}
Waiting for logs to be available...
.*)`, vars)
		assert.Assert(t, is.Regexp(expected, res.Stdout()))
		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceeded"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})

	t.Run("Start ClusterTask interactively", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"clustertask", "start", "read-clustertask", "-w=name=shared-workspace,emptyDir="},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Choose the git resource to use for source:"); err != nil {
					return err
				}
				if _, err := c.ExpectString("skaffold-git (https://github.com/GoogleContainerTools/skaffold)"); err != nil {
					return err
				}
				if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
					return err
				}
				if _, err := c.ExpectString("Value for param `FILEPATH` of type `string`? (Default is `docs`)"); err != nil {
					return err
				}

				if _, err := c.ExpectString("(docs)"); err != nil {
					return err
				}
				if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
					return err
				}
				if _, err := c.ExpectString("Value for param `FILENAME` of type `string`?"); err != nil {
					return err
				}

				if _, err := c.SendLine("README.md"); err != nil {
					return err
				}
				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				c.Close()
				return nil
			}})
		taskRunGeneratedName := builder.GetTaskRunListWithName(c, "read-clustertask", true).Items[0].Name
		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceed"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})

	t.Run("Start ClusterTask with --use-param-defaults and all the params having default", func(t *testing.T) {
		tkn.MustSucceed(t, "clustertask", "start", "read-clustertask",
			"-i=source="+tePipelineGitResourceName,
			"-p=FILENAME=README.md",
			"-w=name=shared-workspace,emptyDir=",
			"--use-param-defaults",
			"--showlog")
		taskRunGeneratedName := builder.GetTaskRunListWithName(c, "read-clustertask", true).Items[0].Name
		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceed"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})

	t.Run("Start ClusterTask with --use-param-defaults and some of the params not having default", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"clustertask", "start", "read-clustertask",
				"-i=source=" + tePipelineGitResourceName,
				"--use-param-defaults",
				"-w=name=shared-workspace,emptyDir=",
				"--showlog"},
			Procedure: func(c *expect.Console) error {

				if _, err := c.ExpectString("Value for param `FILENAME` of type `string`?"); err != nil {
					return err
				}

				if _, err := c.SendLine("README.md"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				c.Close()
				return nil
			}})
		taskRunGeneratedName := builder.GetTaskRunListWithName(c, "read-clustertask", true).Items[0].Name
		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceed"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})

	t.Run("Start ClusterTask with --pod-template", func(t *testing.T) {
		if tkn.CheckVersion("Pipeline", "v0.10.2") {
			t.Skip("Skip test as pipeline v0.10 doesn't support certain PodTemplate properties")
		}

		tkn.MustSucceed(t, "clustertask", "start", "read-clustertask",
			"-i=source="+tePipelineGitResourceName,
			"-p=FILEPATH=docs",
			"-p=FILENAME=README.md",
			"-w=name=shared-workspace,emptyDir=",
			"--showlog",
			"--pod-template="+helper.GetResourcePath("/podtemplate/podtemplate.yaml"))

		taskRunGeneratedName := builder.GetTaskRunListWithName(c, "read-clustertask", true).Items[0].Name
		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceeded"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})
}
