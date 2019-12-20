// Copyright © 2019 The Tekton Authors.
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
	"bytes"
	"testing"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	goexpect "github.com/Netflix/go-expect"
	"github.com/hinshun/vt10x"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/helper/options"
	"gotest.tools/v3/assert"
)

type promptTest struct {
	name      string
	cmdArgs   []string
	procedure func(*goexpect.Console) error
}

func RunPromptTest(t *testing.T, test promptTest, opts *options.LogOptions) {

	test.runTest(t, test.procedure, func(stdio terminal.Stdio) error {
		var err error
		opts.AskOpts = WithStdio(stdio)
		err = run(opts, test.cmdArgs)
		if err != nil {
			return err
		}

		return err
	})
}

func (opts *startOptions) RunPromptTest(t *testing.T, test promptTest) {

	test.runTest(t, test.procedure, func(stdio terminal.Stdio) error {
		var err error
		opts.askOpts = WithStdio(stdio)
		opts.stream = &cli.Stream{
			Out: stdio.Out,
			Err: stdio.Err,
		}
		err = opts.run(test.cmdArgs[0])
		if err != nil {
			return err
		}

		return err
	})
}

func stdio(c *goexpect.Console) terminal.Stdio {
	return terminal.Stdio{In: c.Tty(), Out: c.Tty(), Err: c.Tty()}
}

func (pt *promptTest) runTest(t *testing.T, procedure func(*goexpect.Console) error, test func(terminal.Stdio) error) {
	t.Parallel()

	// Multiplex output to a buffer as well for the raw bytes.
	buf := new(bytes.Buffer)
	c, state, err := vt10x.NewVT10XConsole(goexpect.WithStdout(buf))
	assert.NilError(t, err)
	defer c.Close()

	donec := make(chan struct{})
	go func() {
		defer close(donec)
		if err := procedure(c); err != nil {
			t.Logf("procedure failed: %v", err)
		}
	}()

	assert.NilError(t, test(stdio(c)))

	// Close the slave end of the pty, and read the remaining bytes from the master end.
	c.Tty().Close()
	<-donec

	t.Logf("Raw output: %q", buf.String())

	// Dump the terminal's screen.
	t.Logf("\n%s", goexpect.StripTrailingEmptyLines(state.String()))
}

// WithStdio helps to test interactive command
// by setting stdio for the ask function
func WithStdio(stdio terminal.Stdio) survey.AskOpt {
	return func(options *survey.AskOptions) error {
		options.Stdio.In = stdio.In
		options.Stdio.Out = stdio.Out
		options.Stdio.Err = stdio.Err
		return nil
	}
}
