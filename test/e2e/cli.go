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
	"time"

	"gotest.tools/v3/icmd"
)

const (
	tknTimeout = 10 * time.Minute
)

// TknRunner contains information about the current test execution tkn binary path
// under test
type TknRunner struct {
	path      string
	namespace string
}

// Path returns the tkn binary path
func (e TknRunner) Path() string {
	return e.path
}

// TknNamespace returns the current Namespace
func (e TknRunner) TknNamespace() string {
	return e.namespace
}

// NewTknRunner returns details about the tkn on particular namespace
func NewTknRunner(namespace string) (TknRunner, error) {
	return TknRunner{
		path:      os.Getenv("TEST_CLIENT_BINARY"),
		namespace: namespace,
	}, nil
}

// Run will help you execute tkn command on a specific namespace, with a timeout
func (e TknRunner) Run(args ...string) *icmd.Result {
	if e.namespace != "" {
		args = append(args, "--namespace", e.namespace)
	}
	cmd := append([]string{e.path}, args...)
	return icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: tknTimeout})
}

// RunNoNamespace will help you execute tkn command without namespace, with a timeout
func (e TknRunner) RunNoNamespace(args ...string) *icmd.Result {
	cmd := append([]string{e.path}, args...)
	return icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: tknTimeout})
}

// RunWithOption will help you execute tkn command with namespace, cmd option
func (e TknRunner) RunWithOption(option icmd.CmdOp, args ...string) *icmd.Result {
	if e.namespace != "" {
		args = append(args, "--namespace", e.namespace)
	}
	cmd := append([]string{e.path}, args...)
	return icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: tknTimeout}, option)
}

// Kubectl type
type Kubectl struct {
	namespace string
}

// New Kubectl object
func NewKubectl(namespace string) Kubectl {
	return Kubectl{
		namespace: namespace,
	}
}

// Run will help you execute kubectl command on a specific namespace, with a timeout
func (k Kubectl) Run(args ...string) *icmd.Result {
	if k.namespace != "" {
		args = append(args, "--namespace", k.namespace)
	}
	cmd := append([]string{"kubectl"}, args...)
	return icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: tknTimeout})
}

// RunNoNamespace will help you execute kubectl command without namespace, with a timeout
func (k Kubectl) RunNoNamespace(args ...string) *icmd.Result {
	cmd := append([]string{"kubectl"}, args...)
	return icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: tknTimeout})
}

// Assert verifies result against expected(exit code)
func Assert(t *testing.T, result *icmd.Result, expected icmd.Expected) {
	result.Assert(t, expected)
}
