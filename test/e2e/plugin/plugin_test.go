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

package e2e

import (
	"os"
	"testing"

	"github.com/tektoncd/cli/test/cli"
	"github.com/tektoncd/cli/test/framework"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
	"gotest.tools/v3/icmd"
	knativetest "knative.dev/pkg/test"
)

func TestTknPlugin(t *testing.T) {
	t.Parallel()
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer framework.TearDown(t, c, namespace)

	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)
	currentpath, err := os.Getwd()
	assert.NilError(t, err)
	defer env.Patch(t, "TKN_PLUGINS_DIR", currentpath)()
	t.Run("Success", func(t *testing.T) {
		tkn.MustSucceed(t, "success")
		tkn.MustSucceed(t, "success", "with", "args")
	})
	t.Run("Failure", func(t *testing.T) {
		tkn.Run("failure").Assert(t, icmd.Expected{
			ExitCode: 12,
		})
		tkn.Run("failure", "with", "args").Assert(t, icmd.Expected{
			ExitCode: 12,
		})
		tkn.Run("failure", "exit20").Assert(t, icmd.Expected{
			ExitCode: 20,
		})
	})
}
