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

package resource

import (
	"fmt"
	"strings"
	"testing"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	"github.com/tektoncd/cli/test/cli"
	"github.com/tektoncd/cli/test/framework"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
	knativetest "knative.dev/pkg/test"
)

func TestCreateGitResourceInteractively(t *testing.T) {
	t.Parallel()
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer framework.TearDown(t, c, namespace)

	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)

	t.Run("Create pipeline resource of git type, interactively in namespace "+namespace, func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"resource", "create"},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.Send("skaffold-git"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("git"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for url :"); err != nil {
					return err
				}

				if _, err := c.Send("https://github.com/tektoncd/cli"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for revision :"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
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
			},
		})
	})

	t.Run("list single pipeline resource of git type", func(t *testing.T) {
		res := tkn.MustSucceed(t, "resource", "list")
		golden.Assert(t, res.Stdout(), strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
	})
}

func TestCreateImageResourceInteractively(t *testing.T) {
	t.Parallel()
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer framework.TearDown(t, c, namespace)

	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)

	t.Run("Create pipeline resource of image type, interactively in namespace "+namespace, func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"resource", "create"},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.Send("my-image"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("git"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("image"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for url :"); err != nil {
					return err
				}

				if _, err := c.Send("quay.io/tekton/controller"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for digest :"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("New image resource \"my-image\" has been created"); err != nil {
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

	t.Run("list single pipeline resource of image type", func(t *testing.T) {
		res := tkn.MustSucceed(t, "resource", "list")
		golden.Assert(t, res.Stdout(), strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
	})
}

func TestCreatePullRequestResourceInteractively(t *testing.T) {
	t.Parallel()
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer framework.TearDown(t, c, namespace)

	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)

	t.Run("Create pipeline resource of Pull Request type, interactively in namespace "+namespace, func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"resource", "create"},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("my-pullrequest"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("git"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("image"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("pullRequest"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for url :"); err != nil {
					return err
				}

				if _, err := c.SendLine("https://github.com/tektoncd/pipeline/pull/100"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Do you want to set secrets ?"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Yes"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("No"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("New pullRequest resource \"my-pullrequest\" has been created"); err != nil {
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

	t.Run("list single pipeline resource of pullrequest type", func(t *testing.T) {
		res := tkn.MustSucceed(t, "resource", "list")
		golden.Assert(t, res.Stdout(), strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
	})
}

func TestCreateStorageResourceInteractively(t *testing.T) {
	t.Parallel()
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer framework.TearDown(t, c, namespace)

	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)

	t.Run("Create pipeline resource of storage type, interactively in namespace "+namespace, func(t *testing.T) {
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"resource", "create"},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.Send("my-storage"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("git"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowUp)); err != nil {
					return err
				}

				if _, err := c.ExpectString("storage"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a storage type"); err != nil {
					return err
				}

				if _, err := c.ExpectString("gcs"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for location :"); err != nil {
					return err
				}

				if _, err := c.SendLine("gs://build-crd-tests/archive.zip"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for dir :"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Secret Key for GOOGLE_APPLICATION_CREDENTIALS :"); err != nil {
					return err
				}

				if _, err := c.SendLine("secret-key"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Secret Name for GOOGLE_APPLICATION_CREDENTIALS :"); err != nil {
					return err
				}

				if _, err := c.SendLine("secret"); err != nil {
					return err
				}

				if _, err := c.ExpectString("New storage resource \"my-storage\" has been created"); err != nil {
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

	t.Run("list single pipeline resource of storage type", func(t *testing.T) {
		res := tkn.MustSucceed(t, "resource", "list")
		golden.Assert(t, res.Stdout(), strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
	})
}
