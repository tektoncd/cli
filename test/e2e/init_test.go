// +build e2e

package e2e

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"testing"
)

var originalDir string
var context1 string

const (
	tknContext        = "github.com/tektoncd/cli/cmd/tkn"
	teTaskName        = "te-task"
	teTaskRunName     = "te-task-run"
	tePipelineName    = "te-pipeline"
	tePipelineRunName = "te-pipelinerun"
	PipelineName      = "output-pipeline"
	PipelineRunName   = "output-pipeline-run"
)

func TestMain(m *testing.M) {
	log.Println("Running main e2e Test suite ")
	originalDir, context1 = SetupProcess()
	v := m.Run()
	TearDownProcess()
	os.Exit(v)
}

// Verify the other tests didn't leave
// any goroutines running.
func goroutineLeaked() bool {
	//... need to implement
	fmt.Printf("#goroutines: %d\n", runtime.NumGoroutine())
	return true
}

func SetupProcess() (string, string) {

	tkncli := NewTknRunner(tknContext)
	context1 := CreateNewContext()
	originalDir = Getwd()
	Chdir(context1)
	tkncli.BuildTknClient()
	return originalDir, context1
}

func TearDownProcess() {

	DeleteDir(context1)
	Chdir(originalDir)
}
