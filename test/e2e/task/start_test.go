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

package task

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

func TestTaskStartE2E(t *testing.T) {
	t.Parallel()
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer framework.TearDown(t, c, namespace)

	kubectl := cli.NewKubectl(namespace)
	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)

	t.Logf("Creating Task read-task in namespace: %s ", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("read-file.yaml"))

	t.Logf("Creating git pipeline resource in namespace: %s", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("git-resource.yaml"))

	t.Run("Get list of Tasks from namespace  "+namespace, func(t *testing.T) {
		res := tkn.Run("task", "list")
		expected := builder.ListAllTasksOutput(t, c, map[int]interface{}{
			0: &builder.TaskData{
				Name: "read-task",
			},
		})
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      expected,
		})
	})

	t.Run("Start TaskRun using tkn start command with SA as 'pipeline' ", func(t *testing.T) {
		res := tkn.MustSucceed(t, "task", "start", "read-task",
			"-i=source="+tePipelineGitResourceName,
			"-p=FILEPATH=docs",
			"-p=FILENAME=README.md",
			"--showlog",
			"true")

		vars := make(map[string]interface{})
		taskRunGeneratedName := builder.GetTaskRunListWithName(c, "read-task").Items[0].Name
		vars["Taskrun"] = taskRunGeneratedName
		expected := helper.ProcessString(`(TaskRun started: {{.Taskrun}}
Waiting for logs to be available...
.*)`, vars)
		assert.Assert(t, is.Regexp(expected, res.Stdout()))
	})

	t.Run("Start TaskRun using tkn task start command with --use-param-defaults and all the params having default", func(t *testing.T) {
		res := tkn.MustSucceed(t, "task", "start", "read-task",
			"-i=source="+tePipelineGitResourceName,
			"--use-param-defaults",
			"-p=FILENAME=README.md",
			"--showlog",
			"true")
		assert.Assert(t, is.Regexp("TaskRun started:.*", res.Stdout()))
	})

	t.Run("Start TaskRun using tkn task start command with --use-param-defaults and some of the params not having default", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"task", "start", "read-task",
				"-i=source=" + tePipelineGitResourceName,
				"--use-param-defaults",
				"--showlog",
				"true"},
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
	})

	t.Run("Get list of TaskRuns from namespace  "+namespace, func(t *testing.T) {

		taskRun1GeneratedName := builder.GetTaskRunListWithName(c, "read-task").Items[0].Name
		taskRun2GeneratedName := builder.GetTaskRunListWithName(c, "read-task").Items[1].Name
		taskRun3GeneratedName := builder.GetTaskRunListWithName(c, "read-task").Items[2].Name

		if err := wait.ForTaskRunState(c, taskRun1GeneratedName, wait.TaskRunSucceed(taskRun1GeneratedName), "TaskRunSucceed"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}

		if err := wait.ForTaskRunState(c, taskRun2GeneratedName, wait.TaskRunSucceed(taskRun2GeneratedName), "TaskRunSucceed"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}

		if err := wait.ForTaskRunState(c, taskRun3GeneratedName, wait.TaskRunSucceed(taskRun3GeneratedName), "TaskRunSucceed"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}

		res := tkn.Run("taskrun", "list")
		expected := builder.ListAllTaskRunsOutput(t, c, false, map[int]interface{}{
			0: &builder.TaskRunData{
				Name:   "read-task-run-",
				Status: "Succeeded",
			},
			1: &builder.TaskRunData{
				Name:   "read-task-run-",
				Status: "Succeeded",
			},
			2: &builder.TaskRunData{
				Name:   "read-task-run-",
				Status: "Succeeded",
			},
		})
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      expected,
		})
	})

	t.Run("Validate interactive task logs, with  follow mode (-f) ", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"task", "logs", "-f"},
			Procedure: func(c *expect.Console) error {

				if _, err := c.ExpectString("Select taskrun:"); err != nil {
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
	})

	t.Logf("Creating Task task-with-workspace in namespace: %s ", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("/workspace/task-with-workspace.yaml"))

	t.Run("Start TaskRun with --workspace and volumeClaimTemplate", func(t *testing.T) {
		if tkn.CheckVersion("Pipeline", "v0.10.2") {
			t.Skip("Skip test as pipeline v0.10.2 doesn't support volumeClaimTemplates")
		}

		tkn.MustSucceed(t, "task", "start", "task-with-workspace",
			"--showlog",
			"--workspace=name=read-allowed,volumeClaimTemplateFile="+helper.GetResourcePath("/workspace/pvc.yaml"))

		taskRunGeneratedName := builder.GetTaskRunListWithName(c, "task-with-workspace").Items[0].Name
		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceeded"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})
}
