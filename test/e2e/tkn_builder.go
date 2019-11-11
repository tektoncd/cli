package e2e

import (
	"log"
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

	res := icmd.RunCmd(icmd.Command("go", "build", tkn.path))

	if res.ExitCode != 0 {
		log.Fatalf("Go Build Failed....")
	}

	return " tekton client build successfully!"
}

func Prepare(t *testing.T) func(args ...string) icmd.Cmd {

	run := func(args ...string) icmd.Cmd {
		return icmd.Command("./tkn", append([]string{}, args...)...)
	}
	return run
}
