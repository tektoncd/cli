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

package pipelinerun

import (
	"strings"
	"testing"

	"github.com/tektoncd/cli/test/e2e"
	"gotest.tools/v3/icmd"
	knativetest "knative.dev/pkg/test"
)

func TestPipelineRunLogE2E(t *testing.T) {
	t.Parallel()
	c, namespace := e2e.Setup(t)
	knativetest.CleanupOnInterrupt(func() { e2e.TearDown(t, c, namespace) }, t.Logf)
	defer e2e.TearDown(t, c, namespace)

	kubectl := e2e.NewKubectl(namespace)
	tkn, err := e2e.NewTknRunner(namespace)
	if err != nil {
		t.Fatalf("Error creating tknRunner %+v", err)
	}

	if tkn.CheckVersion("Pipeline", "v0.10.2") {
		t.Skip("Skip test as pipeline v0.10.2 doesn't support finally")
	}

	t.Logf("Creating pipelinerun in namespace: %s", namespace)
	e2e.Assert(t, kubectl.Create(e2e.ResourcePath("pipelinerun-with-finally.yaml")), icmd.Success)

	t.Run("Pipelinerun logs with finally  "+namespace, func(t *testing.T) {
		res := tkn.Run("pipelinerun", "logs", "exit-handler", "-f")
		s := []string{
			"[print-msg : main] printing a message\n",
			"[echo-on-exit : main] finally\n",
		}
		expected := strings.Join(s, "\n") + "\n"
		e2e.AssertOutput(t, expected, res.Stdout())
	})
}
