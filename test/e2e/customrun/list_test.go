//go:build e2e
// +build e2e

// Copyright © 2024 The Tekton Authors.
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

package customrun

import (
	"testing"

	"github.com/tektoncd/cli/test/cli"

	"github.com/tektoncd/cli/test/framework"
	"github.com/tektoncd/cli/test/helper"
	"gotest.tools/assert"
	knativetest "knative.dev/pkg/test"
)

func TestCustomRunListE2E(t *testing.T) {
	t.Parallel()
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer framework.TearDown(t, c, namespace)

	kubectl := cli.NewKubectl(namespace)
	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)
	crName := "customrun-foo"

	t.Logf("Creating CustomRun %s in namespace: %s ", crName, namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("customrun/customrun.yaml"))

	t.Run("List customrun in namespace "+namespace, func(t *testing.T) {
		res := tkn.MustSucceed(t, "customrun", "list")
		stdout := res.Stdout()
		assert.Check(t, helper.ContainsAll(stdout, crName))
	})
}
