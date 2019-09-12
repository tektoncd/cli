package e2e

import (
	"context"
	"log"
	"os/exec"
	"strings"
	"time"
)

func CmdShouldPass(cmdName string) string {
	cmdArgs := strings.Fields(cmdName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:len(cmdArgs)]...)

	out, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		log.Fatalln("Command timed out")
	}

	if err != nil && cmd.ProcessState.ExitCode() != 0 {
		log.Fatalf("Non-zero exit code %+v: %+v , %+v", cmd.ProcessState.ExitCode(), err, string(out))
	}
	return string(out)

}

func CmdShouldFail(cmdName string) string {
	cmdArgs := strings.Fields(cmdName)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:len(cmdArgs)]...)

	out, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		log.Fatalln("Command timed out")
	}

	if err == nil || cmd.ProcessState.ExitCode() == 0 {
		log.Fatalf("Expected failure with Non Zero Error code but passed %s with exit code %+v", string(out), cmd.ProcessState.ExitCode())
	}
	return string(err.Error())

}
