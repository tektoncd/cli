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
	"github.com/tektoncd/cli/test/e2e"
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
	c, namespace := e2e.Setup(t)
	knativetest.CleanupOnInterrupt(func() { e2e.TearDown(t, c, namespace) }, t.Logf)
	defer e2e.TearDown(t, c, namespace)

	kubectl := e2e.NewKubectl(namespace)
	tkn, err := e2e.NewTknRunner(namespace)
	if err != nil {
		t.Fatalf("Error creating tknRunner %+v", err)
	}

	t.Logf("Creating task in namespace: %s ", namespace)
	e2e.Assert(t, kubectl.Create(e2e.ResourcePath("read-file.yaml")), icmd.Success)

	t.Logf("Creating git pipeline resource in namespace: %s", namespace)
	e2e.Assert(t, kubectl.Create(e2e.ResourcePath("git-resource.yaml")), icmd.Success)

	t.Run("Get list of Tasks from namespace  "+namespace, func(t *testing.T) {
		res := tkn.Run("task", "list")
		expected := e2e.ListAllTasksOutput(t, c, map[int]interface{}{
			0: &e2e.TaskData{
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
		res := tkn.Run("task", "start", "read-task",
			"-i=source="+tePipelineGitResourceName,
			"-p=FILENAME=docs/README.md",
			"--showlog",
			"true")

		vars := make(map[string]interface{})
		taskRunGeneratedName := e2e.GetTaskRunListWithName(c, "read-task").Items[0].Name
		vars["Taskrun"] = taskRunGeneratedName
		expected := e2e.ProcessString(`(TaskRun started: {{.Taskrun}}
Waiting for logs to be available...
.*)`, vars)
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		assert.Assert(t, is.Regexp(expected, res.Stdout()))
	})

	t.Run("Get list of TaskRuns from namespace  "+namespace, func(t *testing.T) {
		res := tkn.Run("taskrun", "list")
		expected := e2e.ListAllTaskRunsOutput(t, c, false, map[int]interface{}{
			0: &e2e.TaskRunData{
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

	t.Run("Get a Taskrun using field-selector from namespace  "+namespace, func(t *testing.T) {
		taskRunGeneratedName := e2e.GetTaskRunListWithName(c, "read-task").Items[0].Name
		fs := "metadata.name=" + taskRunGeneratedName
		res := tkn.Run("taskrun", "list", "--field-selector", fs)

		expected := e2e.ListAllTaskRunsOutput(t, c, false, map[int]interface{}{
			0: &e2e.TaskRunData{
				Name:   taskRunGeneratedName,
				Status: "Succeeded",
			},
		})
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      expected,
		})
	})

	t.Run("Get list of Taskruns using field-selector from namespace  "+namespace, func(t *testing.T) {
		fs := "metadata.namespace=" + namespace
		res := tkn.Run("taskrun", "list", "--field-selector", fs)

		expected := e2e.ListAllTaskRunsOutput(t, c, false, map[int]interface{}{
			0: &e2e.TaskRunData{
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
		tkn.RunInteractiveTests(t, &e2e.Prompt{
			CmdArgs: []string{"task", "logs", "-f"},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Select task:"); err != nil {
					return err
				}

				if _, err := c.ExpectString("read-task"); err != nil {
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
}
