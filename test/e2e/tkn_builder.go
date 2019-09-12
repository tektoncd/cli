package e2e

import (
	"testing"

	"gotest.tools/icmd"
)

type TknRunner struct {
	path string
}

func NewTknRunner(tknPath string) TknRunner {
	return TknRunner{
		path: tknPath,
	}
}

func (tkn *TknRunner) BuildTknClient() string {

	return (CmdShouldPass("go build "+tkn.path) + " tekton client build successfully!")
}

func Prepare(t *testing.T) func(args ...string) icmd.Cmd {

	run := func(args ...string) icmd.Cmd {
		return icmd.Command("./tkn", append([]string{}, args...)...)
	}
	return run
}
