//go:build e2e
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
	"time"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/cli/test/builder"
	"github.com/tektoncd/cli/test/cli"
	"github.com/tektoncd/cli/test/framework"
	"github.com/tektoncd/cli/test/helper"
	"github.com/tektoncd/cli/test/wait"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

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
			"--showlog")

		vars := make(map[string]interface{})
		taskRunGeneratedName := builder.GetTaskRunListWithTaskName(c, "read-task", true).Items[0].Name
		vars["Taskrun"] = taskRunGeneratedName
		expected := helper.ProcessString(`(TaskRun started: {{.Taskrun}}
Waiting for logs to be available...
.*)`, vars)
		assert.Assert(t, is.Regexp(expected, res.Stdout()))

		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceed"); err != nil {
			t.Errorf("Error waiting for TaskRun %q to finished: %s", taskRunGeneratedName, err)
		}
	})

	t.Run("Start TaskRun using tkn task start command with --use-param-defaults and all the params having default", func(t *testing.T) {
		res := tkn.MustSucceed(t, "task", "start", "read-task",
			"-i=source="+tePipelineGitResourceName,
			"--use-param-defaults",
			"-p=FILENAME=README.md",
			"--showlog")
		assert.Assert(t, is.Regexp("TaskRun started:.*", res.Stdout()))
	})

	t.Run("Start TaskRun using tkn task start command with --use-param-defaults and some of the params not having default", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{
				"task", "start", "read-task",
				"-i=source=" + tePipelineGitResourceName,
				"--use-param-defaults",
				"--showlog",
			},
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
			},
		})
	})

	t.Run("Get list of TaskRuns from namespace  "+namespace, func(t *testing.T) {
		taskRuns := builder.GetTaskRunListWithTaskName(c, "read-task", false)
		taskRun1GeneratedName := taskRuns.Items[0].Name
		taskRun2GeneratedName := taskRuns.Items[1].Name
		taskRun3GeneratedName := taskRuns.Items[2].Name

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

	t.Run("Start TaskRun using tkn task start command with passing one param and tkn will ask for other", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{
				"task", "start", "read-task",
				"-i=source=" + tePipelineGitResourceName,
				"-p=FILEPATH=docs",
				"--showlog",
			},
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
			},
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
			},
		})
	})

	t.Logf("Creating Task task-with-workspace in namespace: %s ", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("task-with-workspace.yaml"))

	t.Run("Start TaskRun with --workspace and volumeClaimTemplate", func(t *testing.T) {
		if tkn.CheckVersion("Pipeline", "v0.10.2") {
			t.Skip("Skip test as pipeline v0.10 doesn't support volumeClaimTemplates")
		}

		res := tkn.MustSucceed(t, "task", "start", "task-with-workspace",
			"--showlog",
			"--workspace=name=read-allowed,volumeClaimTemplateFile="+helper.GetResourcePath("pvc.yaml"))

		vars := make(map[string]interface{})
		taskRunGeneratedName := builder.GetTaskRunListWithTaskName(c, "task-with-workspace", true).Items[0].Name
		vars["Taskrun"] = taskRunGeneratedName
		expected := helper.ProcessString(`(TaskRun started: {{.Taskrun}}
Waiting for logs to be available...
.*)`, vars)
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		assert.Assert(t, is.Regexp(expected, res.Stdout()))

		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceeded"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})

	t.Run("Start TaskRun with --pod-template", func(t *testing.T) {
		tkn.MustSucceed(t, "task", "start", "read-task",
			"-i=source="+tePipelineGitResourceName,
			"-p=FILEPATH=docs",
			"-p=FILENAME=README.md",
			"--showlog",
			"--pod-template="+helper.GetResourcePath("/podtemplate/podtemplate.yaml"))

		taskRunGeneratedName := builder.GetTaskRunListWithTaskName(c, "read-task", true).Items[0].Name
		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceeded"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})

	t.Run("Cancel finished TaskRun with tkn taskrun cancel", func(t *testing.T) {
		// Get last TaskRun for read-task
		taskRunLast := builder.GetTaskRunListWithTaskName(c, "read-task", true).Items[0]

		// Cancel TaskRun
		res := tkn.Run("taskrun", "cancel", taskRunLast.Name)

		// Expect error from TaskRun cancel for already completed TaskRun
		expected := "Error: failed to cancel TaskRun " + taskRunLast.Name + ": TaskRun has already finished execution\n"
		assert.Assert(t, res.Stderr() == expected)
	})

	t.Logf("Creating Task sleep in namespace: %s", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("cancel/task-cancel.yaml"))
	t.Run("Cancel TaskRun with tkn taskrun cancel", func(t *testing.T) {
		task := "sleep"
		// Start TaskRun
		tkn.MustSucceed(t, "task", "start", task, "-p", "seconds=30s")

		// Get name of most recent TaskRun
		taskRunName := builder.GetTaskRunListWithTaskName(c, task, true).Items[0].Name

		// Cancel TaskRun
		res := tkn.MustSucceed(t, "taskrun", "cancel", taskRunName)

		// Verify successful cancel
		if err := wait.ForTaskRunState(c, taskRunName, wait.TaskRunFailed(taskRunName), "Cancelled"); err != nil {
			t.Errorf("Error waiting for TaskRun %q to finished: %s", taskRunName, err)
		}

		// Expect successfully cancelled TaskRun
		expected := "TaskRun cancelled: " + taskRunName + "\n"
		assert.Assert(t, res.Stdout() == expected)

		cancelledRun := builder.GetTaskRun(c, taskRunName)
		assert.Assert(t, v1beta1.TaskRunReasonCancelled.String() == cancelledRun.Status.Conditions[0].Reason)
	})

	t.Run("Start TaskRun using tkn task start with --last option", func(t *testing.T) {
		// Get last TaskRun for read-task
		lastTaskRun := builder.GetTaskRunListWithTaskName(c, "read-task", true).Items[0]

		// Start TaskRun using --last
		tkn.MustSucceed(t, "task", "start", "read-task",
			"--last",
			"--showlog")

		// Sleep to make make sure TaskRun is created/running
		time.Sleep(1 * time.Second)

		// Get name of most recent TaskRun and wait for it to succeed
		taskRunUsingLast := builder.GetTaskRunListWithTaskName(c, "read-task", true).Items[0]
		if err := wait.ForTaskRunState(c, taskRunUsingLast.Name, wait.TaskRunSucceed(taskRunUsingLast.Name), "TaskRunSucceeded"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}

		// Expect that previous TaskRun spec will match most recent TaskRun spec
		expected := lastTaskRun.Spec
		got := taskRunUsingLast.Spec
		if d := cmp.Diff(got, expected); d != "" {
			t.Fatalf("-got, +want: %v", d)
		}
	})

	t.Logf("Creating Task task-optional-ws in namespace: %s ", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("task-with-optional-workspace.yaml"))

	t.Run("Start Task interactively with optional workspace (yes)", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"task", "start", "task-optional-ws"},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Do you want to give specifications for the optional workspace `read-allowed`: (y/N)"); err != nil {
					return err
				}

				if _, err := c.SendLine("y"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Please give specifications for the workspace: read-allowed"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Name for the workspace :"); err != nil {
					return err
				}

				if _, err := c.SendLine("read-allowed"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Value of the Sub Path :"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Type of the Workspace :"); err != nil {
					return err
				}

				if _, err := c.SendLine("emptyDir"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Type of EmptyDir :"); err != nil {
					return err
				}

				if _, err := c.SendLine(""); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				c.Close()
				return nil
			},
		})
		taskRunGeneratedName := builder.GetTaskRunListWithTaskName(c, "task-optional-ws", true).Items[0].Name
		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceed"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})
}
