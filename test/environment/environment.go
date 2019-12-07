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
