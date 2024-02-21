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

package pipeline

import (
	"testing"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	"github.com/tektoncd/cli/test/cli"
	"github.com/tektoncd/cli/test/framework"
	"github.com/tektoncd/cli/test/helper"
	"gotest.tools/v3/assert"
	knativetest "knative.dev/pkg/test"
)

func TestPipelineInteractiveStartE2E(t *testing.T) {
	t.Parallel()
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer framework.TearDown(t, c, namespace)

	kubectl := cli.NewKubectl(namespace)
	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)

	t.Logf("Creating pipeline in namespace: %s", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("pipeline.yaml"))

	t.Run("Start PipelineRun using pipeline start interactively with SA as 'pipeline' ", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"pipeline", "start", "output-pipeline", "-w=name=shared-data,emptyDir="},
			Procedure: func(c *expect.Console) error {

				if _, err := c.ExpectString("Value for param `message` of type `string`? (Default is `hello`)"); err != nil {
					return err
				}

				if _, err := c.ExpectString("hello"); err != nil {
					return err
				}

				if _, err := c.SendLine("test-e2e"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Value for param `filename` of type `string`?"); err != nil {
					return err
				}

				if _, err := c.SendLine("output"); err != nil {
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

	t.Run("Validate pipeline logs, with  follow mode (-f) and --last ", func(t *testing.T) {
		res := tkn.Run(t, "pipeline", "logs", "--last", "-f")
		expected := "\n\ntest-e2e\n\n"
		helper.AssertOutput(t, expected, res.Stdout())
	})

	t.Run("Start PipelineRun using pipeline start interactively using --use-param-defaults and some of the params not having default ", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{
				"pipeline", "start", "output-pipeline",
				"--use-param-defaults",
				"-w=name=shared-data,emptyDir=",
			},
			Procedure: func(c *expect.Console) error {

				if _, err := c.ExpectString("Value for param `filename` of type `string`?"); err != nil {
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
	})

	t.Run("Start PipelineRun using pipeline start interactively with --param flag and --use-param-defaults and some of the params not having default ", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{
				"pipeline", "start", "output-pipeline",
				"-p=message=test-e2e", "--use-param-defaults", "-w=name=shared-data,emptyDir=",
			},
			Procedure: func(c *expect.Console) error {

				if _, err := c.ExpectString("Value for param `filename` of type `string`?"); err != nil {
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
	})

	t.Run("Validate interactive pipeline logs, with  follow mode (-f) ", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"pipeline", "logs", "-f"},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Select pipelinerun:"); err != nil {
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
}

func TestPipelineInteractiveStartWithOptionalWorkspaceE2E(t *testing.T) {
	t.Parallel()
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer framework.TearDown(t, c, namespace)

	kubectl := cli.NewKubectl(namespace)
	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)

	t.Logf("Creating pipeline in namespace: %s", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("pipeline-with-optional-workspace.yaml"))

	t.Run("Start PipelineRun using pipeline start interactively with SA as 'pipeline' ", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"pipeline", "start", "pipeline-optional-ws"},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Do you want to give specifications for the optional workspace `ws`: (y/N)"); err != nil {
					return err
				}

				if _, err := c.SendLine("y"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Please give specifications for the workspace: ws"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Name for the workspace :"); err != nil {
					return err
				}

				if _, err := c.SendLine("ws"); err != nil {
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
	})

	t.Run("Validate interactive pipeline logs, with  follow mode (-f) ", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"pipeline", "logs", "-f"},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Select pipeline:"); err != nil {
					return err
				}

				if _, err := c.ExpectString("pipeline-optional-ws"); err != nil {
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
}
