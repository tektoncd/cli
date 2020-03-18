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

package environment

import (
	"log"
	"os"
	"testing"
	"time"

	"gotest.tools/v3/icmd"
)

// TknRunner contains information about the current test execution tkn binary path
// under test
type TknRunner struct {
	Path string
}

// TknBinary returns the tkn binary for this testing environment
func (e *TknRunner) TknBinary() string {
	return e.Path
}

// NewTknRunner returns details about the testing environment
func NewTknRunner() (*TknRunner, error) {
	env := os.Getenv("TEST_CLIENT_BINARY")
	if env == "" {
		log.Println("\"TEST_CLIENT_BINARY\" env variable is required, Cannot Procced E2E Tests")
		os.Exit(0)
	}
	return &TknRunner{
		Path: env,
	}, nil
}

// Prepare will help you create tkn cmd that can be executable with a timeout
func (e *TknRunner) Prepare(t *testing.T) func(args ...string) icmd.Cmd {
	run := func(args ...string) icmd.Cmd {
		return icmd.Cmd{Command: append([]string{e.Path}, args...), Timeout: 10 * time.Minute}
	}
	return run
}
