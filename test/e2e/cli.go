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
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Netflix/go-expect"
	goexpect "github.com/Netflix/go-expect"
	"github.com/hinshun/vt10x"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"
)

const (
	timeout = 5 * time.Minute
)

// TknRunner contains information about the current test execution tkn binary path
// under test
type TknRunner struct {
	path      string
	namespace string
}

// Prompt provides test utility for prompt test.
type Prompt struct {
	CmdArgs   []string
	Procedure func(*goexpect.Console) error
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
	return icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: timeout})
}

// RunNoNamespace will help you execute tkn command without namespace, with a timeout
func (e TknRunner) RunNoNamespace(args ...string) *icmd.Result {
	cmd := append([]string{e.path}, args...)
	return icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: timeout})
}

// RunWithOption will help you execute tkn command with namespace, cmd option
func (e TknRunner) RunWithOption(option icmd.CmdOp, args ...string) *icmd.Result {
	if e.namespace != "" {
		args = append(args, "--namespace", e.namespace)
	}
	cmd := append([]string{e.path}, args...)
	return icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: timeout}, option)
}

// RunInteractiveTests helps to run interactive tests.
func (e TknRunner) RunInteractiveTests(t *testing.T, ops *Prompt) *expect.Console {
	t.Helper()

	// Multiplex output to a buffer as well for the raw bytes.
	buf := new(bytes.Buffer)
	c, state, err := vt10x.NewVT10XConsole(goexpect.WithStdout(buf))
	assert.NilError(t, err)
	defer c.Close()

	if e.namespace != "" {
		ops.CmdArgs = append(ops.CmdArgs, "--namespace", e.namespace)
	}

	cmd := exec.Command(e.path, ops.CmdArgs[0:len(ops.CmdArgs)]...) //nolint:gosec
	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()
	cmd.Stderr = c.Tty()

	assert.NilError(t, cmd.Start())

	donec := make(chan struct{})
	go func() {
		defer close(donec)
		if err := ops.Procedure(c); err != nil {
			t.Logf("procedure failed: %v", err)
		}
	}()

	// Close the slave end of the pty, and read the remaining bytes from the master end.
	c.Tty().Close()
	<-donec

	// Dump the terminal's screen.
	t.Logf("\n%s", goexpect.StripTrailingEmptyLines(state.String()))

	assert.NilError(t, cmd.Wait())

	return c
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

//Create will help you to create a resource in any namespace
func (k Kubectl) Create(args ...string) *icmd.Result {
	if k.namespace != "" {
		args = append(args, "--namespace", k.namespace)
	}
	cmd := append([]string{"kubectl", "create", "-f"}, args...)
	res := icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: timeout})
	time.Sleep(1 * time.Second)
	return res
}

// Run will help you execute kubectl command on a specific namespace, with a timeout
func (k Kubectl) Run(args ...string) *icmd.Result {
	if k.namespace != "" {
		args = append(args, "--namespace", k.namespace)
	}
	cmd := append([]string{"kubectl"}, args...)
	return icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: timeout})
}

// RunNoNamespace will help you execute kubectl command without namespace, with a timeout
func (k Kubectl) RunNoNamespace(args ...string) *icmd.Result {
	cmd := append([]string{"kubectl"}, args...)
	return icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: timeout})
}

// Assert verifies result against expected(exit code)
func Assert(t *testing.T, result *icmd.Result, expected icmd.Expected) {
	result.Assert(t, expected)
}

// RunInteractiveTests helps to run interactive tests.
func (e TknRunner) RunInteractiveTestsDummy(t *testing.T, ops *Prompt) *expect.Console {
	t.Helper()

	// Multiplex output to a buffer as well for the raw bytes.
	buf := new(bytes.Buffer)
	c, state, err := vt10x.NewVT10XConsole(goexpect.WithStdout(buf))
	assert.NilError(t, err)
	defer c.Close()

	if e.namespace != "" {
		ops.CmdArgs = append(ops.CmdArgs, "--namespace", e.namespace)
	}

	cmd := exec.Command(e.path, ops.CmdArgs[0:len(ops.CmdArgs)]...) //nolint:gosec
	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()
	cmd.Stderr = c.Tty()

	assert.NilError(t, cmd.Start())

	donec := make(chan struct{})
	go func() {
		defer close(donec)
		if err := ops.Procedure(c); err != nil {
			t.Logf("procedure failed: %v", err)
		}
	}()

	// Close the slave end of the pty, and read the remaining bytes from the master end.
	c.Tty().Close()
	<-donec

	// Dump the terminal's screen.
	t.Logf("\n%s", goexpect.StripTrailingEmptyLines(state.String()))

	assert.NilError(t, cmd.Wait())

	return c
}

// TODO: Re-write this to just get the version of Tekton components through tkn version
// as described in https://github.com/tektoncd/cli/issues/1067
func (e TknRunner) CheckVersion(component string, version string) bool {
	cmd := append([]string{e.path}, "version")
	result := icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: timeout})

	return strings.Contains(result.Stdout(), component+" version: "+version)
}
