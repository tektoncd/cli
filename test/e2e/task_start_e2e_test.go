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

package e2e

import (
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/icmd"
	knativetest "knative.dev/pkg/test"
)

func TestTaskStartE2E(t *testing.T) {
	t.Parallel()
	c, namespace := Setup(t)
	knativetest.CleanupOnInterrupt(func() { TearDown(t, c, namespace) }, t.Logf)
	defer TearDown(t, c, namespace)

	tkn, err := NewTknRunner(namespace)
	if err != nil {
		t.Fatalf("Error creating tknRunner %+v", err)
	}

	t.Logf("Creating task in namespace " + namespace)
	CreateResource(t, "read-file.yaml", namespace)

	t.Logf("Creating Git PipelineResource")
	if _, err := c.PipelineResourceClient.Create(getGitResource(tePipelineGitResourceName, namespace)); err != nil {
		t.Fatalf("Failed to create Pipeline Resource `%s`: %s", tePipelineGitResourceName, err)
	}

	t.Run("Get list of Tasks from namespace  "+namespace, func(t *testing.T) {
		res := tkn.Run("task", "list")

		expected := ListAllTasksOutput(t, c, map[int]interface{}{
			0: &TaskData{
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
		taskRunGeneratedName := GetTaskRunListWithName(c, "read-task").Items[0].Name
		vars["Taskrun"] = taskRunGeneratedName
		expected := ProcessString(`(Taskrun started: {{.Taskrun}}
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

		expected := ListAllTaskRunsOutput(t, c, false, map[int]interface{}{
			0: &TaskRunData{
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
		RunInteractiveTests(t, namespace, tkn.Path(), &Prompt{
			CmdArgs: []string{"task", "logs", "-f", "-n", namespace},
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

func CreateResource(t *testing.T, resource, namespace string) {
	t.Helper()
	kubectl := NewKubectl(namespace)
	res := kubectl.Run("apply", "-f", TestResourcePath(resource))
	time.Sleep(1 * time.Second)
	Assert(t, res, icmd.Success)
}
