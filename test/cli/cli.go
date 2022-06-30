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

package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Netflix/go-expect"
	goexpect "github.com/Netflix/go-expect"
	"github.com/tektoncd/cli/test/helper"
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
	if os.Getenv("TEST_CLIENT_BINARY") != "" {
		return TknRunner{
			path:      os.Getenv("TEST_CLIENT_BINARY"),
			namespace: namespace,
		}, nil
	}
	return TknRunner{
		path:      os.Getenv("TEST_CLIENT_BINARY"),
		namespace: namespace,
	}, fmt.Errorf("Error: couldn't Create tknRunner, please do check tkn binary path: (%+v)", os.Getenv("TEST_CLIENT_BINARY"))
}

// Run will help you execute tkn command on a specific namespace, with a timeout
func (e TknRunner) Run(t *testing.T, args ...string) *icmd.Result {
	t.Helper()
	if e.namespace != "" {
		args = append(args, "--namespace", e.namespace)
	}
	cmd := append([]string{e.path}, args...)
	return icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: timeout})
}

// MustSucceed asserts that the command ran with 0 exit code
func (e TknRunner) MustSucceed(t *testing.T, args ...string) *icmd.Result {
	t.Helper()
	return e.Assert(t, icmd.Success, args...)
}

// Assert runs a command and verifies exit code (0)
func (e TknRunner) Assert(t *testing.T, exp icmd.Expected, args ...string) *icmd.Result {
	t.Helper()
	res := e.Run(t, args...)
	res.Assert(t, exp)
	return res
}

// RunNoNamespace will help you execute tkn command without namespace, with a timeout
func (e TknRunner) RunNoNamespace(t *testing.T, args ...string) *icmd.Result {
	t.Helper()
	cmd := append([]string{e.path}, args...)
	return icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: timeout})
}

// RunWithOption will help you execute tkn command with namespace, cmd option
func (e TknRunner) RunWithOption(t *testing.T, option icmd.CmdOp, args ...string) *icmd.Result {
	t.Helper()
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
	c, state, err := helper.NewVT10XConsole(goexpect.WithStdout(buf))
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
	_ = c.Tty().Close()
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

// Run will help you execute kubectl command on a specific namespace, with a timeout
func (k Kubectl) Run(t *testing.T, args ...string) *icmd.Result {
	t.Helper()
	if k.namespace != "" {
		args = append(args, "--namespace", k.namespace)
	}
	cmd := append([]string{"kubectl"}, args...)
	return icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: timeout})
}

// RunNoNamespace will help you execute kubectl command without namespace, with a timeout
func (k Kubectl) RunNoNamespace(t *testing.T, args ...string) *icmd.Result {
	t.Helper()
	cmd := append([]string{"kubectl"}, args...)
	return icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: timeout})
}

// MustSucceed asserts that the command ran with 0 exit code
func (k Kubectl) MustSucceed(t *testing.T, args ...string) *icmd.Result {
	t.Helper()
	return k.Assert(t, icmd.Success, args...)
}

// Assert runs a command and verifies against expected
func (k Kubectl) Assert(t *testing.T, exp icmd.Expected, args ...string) *icmd.Result {
	t.Helper()
	res := k.Run(t, args...)
	res.Assert(t, exp)
	time.Sleep(1 * time.Second)
	return res
}

// TODO: Re-write this to just get the version of Tekton components through tkn version
// as described in https://github.com/tektoncd/cli/issues/1067
func (e TknRunner) CheckVersion(t *testing.T, component string, version string) bool {
	t.Helper()
	cmd := append([]string{e.path}, "version")
	result := icmd.RunCmd(icmd.Cmd{Command: cmd, Timeout: timeout})

	return strings.Contains(result.Stdout(), component+" version: "+version)
}
