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
	"github.com/tektoncd/cli/test/e2e"
	"gotest.tools/v3/icmd"
	knativetest "knative.dev/pkg/test"
)

func TestPipelineInteractiveStartE2E(t *testing.T) {
	t.Parallel()
	c, namespace := e2e.Setup(t)
	knativetest.CleanupOnInterrupt(func() { e2e.TearDown(t, c, namespace) }, t.Logf)
	defer e2e.TearDown(t, c, namespace)

	kubectl := e2e.NewKubectl(namespace)
	tkn, err := e2e.NewTknRunner(namespace)
	if err != nil {
		t.Fatalf("Error creating tknRunner %+v", err)
	}

	t.Logf("Creating pipeline in namespace: %s", namespace)
	e2e.Assert(t, kubectl.Create(e2e.ResourcePath("pipeline.yaml")), icmd.Success)

	t.Logf("Creating git pipeline resource in namespace: %s", namespace)
	e2e.Assert(t, kubectl.Create(e2e.ResourcePath("git-resource.yaml")), icmd.Success)

	t.Run("Start PipelineRun using pipeline start interactively with SA as 'pipeline' ", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &e2e.Prompt{
			CmdArgs: []string{"pipeline", "start", "output-pipeline", "-s", "pipeline"},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Choose the git resource to use for source-repo:"); err != nil {
					return err
				}

				if _, err := c.ExpectString("skaffold-git (https://github.com/GoogleContainerTools/skaffold)"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				c.Close()
				return nil
			}})
	})

	t.Run("Validate interactive pipeline logs, with  follow mode (-f) ", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &e2e.Prompt{
			CmdArgs: []string{"pipeline", "logs", "-f"},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Select pipeline:"); err != nil {
					return err
				}

				if _, err := c.ExpectString("output-pipeline"); err != nil {
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

func TestPipelineInteractiveStartWithNewResourceE2E(t *testing.T) {
	t.Parallel()
	c, namespace := e2e.Setup(t)
	knativetest.CleanupOnInterrupt(func() { e2e.TearDown(t, c, namespace) }, t.Logf)
	defer e2e.TearDown(t, c, namespace)

	kubectl := e2e.NewKubectl(namespace)
	tkn, err := e2e.NewTknRunner(namespace)
	if err != nil {
		t.Fatalf("Error creating tknRunner %+v", err)
	}

	t.Logf("Creating pipeline in namespace: %s", namespace)
	e2e.Assert(t, kubectl.Create(e2e.ResourcePath("pipeline.yaml")), icmd.Success)

	t.Run("Start PipelineRun using pipeline start interactively with SA as 'pipeline' ", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &e2e.Prompt{
			CmdArgs: []string{"pipeline", "start", "output-pipeline", "-s", "pipeline"},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Please create a new \"git\" resource for PipelineResource \"source-repo\""); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("skaffold-git"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for url :"); err != nil {
					return err
				}

				if _, err := c.SendLine("https://github.com/GoogleContainerTools/skaffold"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for revision :"); err != nil {
					return err
				}

				if _, err := c.SendLine("master"); err != nil {
					return err
				}

				if _, err := c.ExpectString("New git resource \"skaffold-git\" has been created"); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				c.Close()
				return nil
			}})
	})

	t.Run("Validate interactive pipeline logs, with  follow mode (-f) ", func(t *testing.T) {
		tkn.RunInteractiveTests(t, &e2e.Prompt{
			CmdArgs: []string{"pipeline", "logs", "-f"},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Select pipeline:"); err != nil {
					return err
				}

				if _, err := c.ExpectString("output-pipeline"); err != nil {
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
