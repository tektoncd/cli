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

package clustertask

import (
	"fmt"
	"os"
	"regexp"
	"strings"
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
	// Set environment variable TEST_CLUSTERTASK_LIST_EMPTY to any value to skip "No ClusterTasks found" test
	if os.Getenv("TEST_CLUSTERTASK_LIST_EMPTY") == "" {
		t.Run("Get list of ClusterTasks when none present", func(t *testing.T) {
			res := tkn.Run("clustertask", "list")
			expected := "No ClusterTasks found\n"
			res.Assert(t, icmd.Expected{
				ExitCode: 0,
				Err:      icmd.None,
				Out:      expected,
			})
		})
	}
	t.Logf("Creating clustertask read-clustertask")
	res := kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("read-file-clustertask.yaml"))
	regex := regexp.MustCompile(`read-clustertask-[a-z0-9]+`)
	clusterTaskName := regex.FindString(res.Stdout())

	t.Logf("Creating git pipeline resource in namespace: %s", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("git-resource.yaml"))

	t.Run("Get list of ClusterTasks", func(t *testing.T) {
		res := tkn.Run("clustertask", "list")
		if os.Getenv("TEST_CLUSTERTASK_LIST_EMPTY") == "" {
			expected := builder.ListAllClusterTasksOutput(t, c, map[int]interface{}{
				0: &builder.TaskData{
					Name: clusterTaskName,
				},
			})
			res.Assert(t, icmd.Expected{
				ExitCode: 0,
				Err:      icmd.None,
				Out:      expected,
			})
		} else {
			assert.Assert(t, strings.Contains(res.Stdout(), clusterTaskName))
		}
	})

	t.Run("Start ClusterTask with flags", func(t *testing.T) {
		res := tkn.MustSucceed(t, "clustertask", "start", clusterTaskName,
			"-i=source="+tePipelineGitResourceName,
			"-p=FILEPATH=docs",
			"-p=FILENAME=README.md",
			"-w=name=shared-workspace,emptyDir=",
			"--showlog")

		vars := make(map[string]interface{})
		taskRunGeneratedName := builder.GetTaskRunListWithClusterTaskName(c, clusterTaskName, true).Items[0].Name
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
			CmdArgs: []string{"clustertask", "start", clusterTaskName, "-w=name=shared-workspace,emptyDir="},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Choose the git resource to use for source:"); err != nil {
					return err
				}
				if _, err := c.ExpectString("skaffold-git (https://github.com/GoogleContainerTools/skaffold#main)"); err != nil {
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
			},
		})
		taskRunGeneratedName := builder.GetTaskRunListWithClusterTaskName(c, clusterTaskName, true).Items[0].Name
		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceed"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})

	t.Run("Start ClusterTask with --use-param-defaults and all the params having default", func(t *testing.T) {
		tkn.MustSucceed(t, "clustertask", "start", clusterTaskName,
			"-i=source="+tePipelineGitResourceName,
			"-p=FILENAME=README.md",
			"-w=name=shared-workspace,emptyDir=",
			"--use-param-defaults",
			"--showlog")
		taskRunGeneratedName := builder.GetTaskRunListWithClusterTaskName(c, clusterTaskName, true).Items[0].Name
		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceed"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})

	t.Run("Start ClusterTask passing --param for some params and some params are not passed", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{
				"clustertask", "start", clusterTaskName,
				"-i=source=" + tePipelineGitResourceName,
				"--param=FILEPATH=docs",
				"-w=name=shared-workspace,emptyDir=",
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
		taskRunGeneratedName := builder.GetTaskRunListWithClusterTaskName(c, clusterTaskName, true).Items[0].Name
		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceed"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})

	t.Run("Start ClusterTask with --use-param-defaults and some of the params not having default", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{
				"clustertask", "start", clusterTaskName,
				"-i=source=" + tePipelineGitResourceName,
				"--use-param-defaults",
				"-w=name=shared-workspace,emptyDir=",
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
		taskRunGeneratedName := builder.GetTaskRunListWithClusterTaskName(c, clusterTaskName, true).Items[0].Name
		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceed"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})

	t.Run("Start ClusterTask with --pod-template", func(t *testing.T) {
		if tkn.CheckVersion("Pipeline", "v0.10.2") {
			t.Skip("Skip test as pipeline v0.10 doesn't support certain PodTemplate properties")
		}

		tkn.MustSucceed(t, "clustertask", "start", clusterTaskName,
			"-i=source="+tePipelineGitResourceName,
			"-p=FILEPATH=docs",
			"-p=FILENAME=README.md",
			"-w=name=shared-workspace,emptyDir=",
			"--showlog",
			"--pod-template="+helper.GetResourcePath("/podtemplate/podtemplate.yaml"))

		taskRunGeneratedName := builder.GetTaskRunListWithClusterTaskName(c, clusterTaskName, true).Items[0].Name
		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceeded"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})

	t.Run("Start TaskRun using tkn ct start with --last option", func(t *testing.T) {
		// Get last TaskRun for read-clustertask
		lastTaskRun := builder.GetTaskRunListWithClusterTaskName(c, clusterTaskName, true).Items[0]

		// Start TaskRun using --last
		tkn.MustSucceed(t, "ct", "start", clusterTaskName,
			"--last",
			"--showlog")

		// Sleep to make make sure TaskRun is created/running
		time.Sleep(1 * time.Second)

		// Get name of most recent TaskRun and wait for it to succeed
		taskRunUsingLast := builder.GetTaskRunListWithClusterTaskName(c, clusterTaskName, true).Items[0]
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

	t.Run("Start TaskRun using tkn ct start with --use-taskrun option", func(t *testing.T) {
		// Get last TaskRun for read-clustertask
		lastTaskRun := builder.GetTaskRunListWithClusterTaskName(c, clusterTaskName, true).Items[0]

		// Start TaskRun using --use-taskrun
		tkn.MustSucceed(t, "ct", "start", clusterTaskName,
			"--use-taskrun",
			lastTaskRun.Name,
			"--showlog")

		// Sleep to make make sure TaskRun is created/running
		time.Sleep(1 * time.Second)

		// Get name of most recent TaskRun and wait for it to succeed
		taskRunUsingParticularTaskRun := builder.GetTaskRunListWithClusterTaskName(c, clusterTaskName, true).Items[0]
		if err := wait.ForTaskRunState(c, taskRunUsingParticularTaskRun.Name, wait.TaskRunSucceed(taskRunUsingParticularTaskRun.Name), "TaskRunSucceeded"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}

		// Expect that selected TaskRun spec will match most recent TaskRun spec
		expected := lastTaskRun.Spec
		got := taskRunUsingParticularTaskRun.Spec
		if d := cmp.Diff(got, expected); d != "" {
			t.Fatalf("-got, +want: %v", d)
		}
	})

	t.Logf("Creating clustertask clustertask-optional-ws")
	res = kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("clustertask-with-optional-workspace.yaml"))
	regex = regexp.MustCompile(`clustertask-optional-ws-[a-z0-9]+`)
	clusterTaskName2 := regex.FindString(res.Stdout())

	t.Run("Start ClusterTask interactively with optional workspace (yes)", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"clustertask", "start", clusterTaskName2},
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

				if _, err := c.ExpectString("Name for the workspace: "); err != nil {
					return err
				}

				if _, err := c.SendLine("read-allowed"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Value of the Sub Path: "); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Type of the Workspace:"); err != nil {
					return err
				}

				if _, err := c.SendLine("emptyDir"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Type of EmptyDir: "); err != nil {
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
		taskRunGeneratedName := builder.GetTaskRunListWithClusterTaskName(c, clusterTaskName2, true).Items[0].Name
		if err := wait.ForTaskRunState(c, taskRunGeneratedName, wait.TaskRunSucceed(taskRunGeneratedName), "TaskRunSucceed"); err != nil {
			t.Errorf("Error waiting for TaskRun to Succeed: %s", err)
		}
	})

	t.Logf("Deleting clustertask %s", clusterTaskName)
	t.Run(fmt.Sprintf("Delete clustertask %s", clusterTaskName), func(t *testing.T) {
		res := tkn.MustSucceed(t, "clustertask", "delete", clusterTaskName, "-f")
		expected := fmt.Sprintf("ClusterTasks deleted: \"%s\"", clusterTaskName)
		res.Assert(t, icmd.Expected{
			Err: icmd.None,
			Out: expected,
		})

		// Check if clustertask %s got deleted
		res = tkn.Run("clustertask", "list")
		assert.Assert(t, !strings.Contains(res.Stdout(), clusterTaskName))
	})

	t.Logf("Deleting clustertask %s", clusterTaskName2)
	t.Run(fmt.Sprintf("Delete clustertask %s", clusterTaskName2), func(t *testing.T) {
		res := tkn.MustSucceed(t, "clustertask", "delete", clusterTaskName2, "-f")
		expected := fmt.Sprintf("ClusterTasks deleted: \"%s\"", clusterTaskName2)
		res.Assert(t, icmd.Expected{
			Err: icmd.None,
			Out: expected,
		})

		// Check if clustertask %s got deleted
		res = tkn.Run("clustertask", "list")
		assert.Assert(t, !strings.Contains(res.Stdout(), clusterTaskName2))
	})
}
