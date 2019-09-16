// +build smoke

package e2e

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/icmd"
)

var originalDir string
var context1 string

const (
	tknContext = "github.com/tektoncd/cli/cmd/tkn"
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

func TestSmoke(t *testing.T) {
	t.Helper()
	t.Parallel()

	run := Prepare(t)

	t.Run("Checking tkn binary", func(t *testing.T) {
		res := icmd.RunCmd(run())
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
		})
		expected := `CLI for tekton pipelines

Usage:
  tkn [command]

Available Commands:
  clustertask Manage clustertasks
  pipeline    Manage pipelines
  pipelinerun Manage pipelineruns
  resource    Manage pipeline resources
  task        Manage tasks
  taskrun     Manage taskruns

Other Commands:
  completion  Prints shell completion scripts
  version     Prints version information

Flags:
  -h, --help   help for tkn

Use "tkn [command] --help" for more information about a command.
`
		assert.Assert(t, is.Equal(expected, res.Stderr()))
	})

	t.Run("Checking tkn client version", func(t *testing.T) {
		res := icmd.RunCmd(run("version"))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		assert.Assert(t, is.Equal("Client version: dev\n", res.Stdout()))
	})

	t.Run("Checking for tkn help option", func(t *testing.T) {

		res := icmd.RunCmd(run("help", "-h"))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
		})
		assert.Assert(t, is.Regexp("(Help provides help for any command in the application.).*", res.Stderr()))

	})

	t.Run("Checking for Task list in (default) namespace", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list"))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      "No tasks found\n",
		})
	})

	t.Run("Checking for Task list in (foo) namespace ", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", "foo"))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      "No tasks found\n",
		})
	})

	t.Run("Listing down task from configured cluster", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-k", os.Getenv("HOME")+"/.kube/config"))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      "No tasks found\n",
		})
	})

	t.Run("Listing down  taskrun list under default namespace (Default)", func(t *testing.T) {
		res := icmd.RunCmd(run("taskrun", "list", "-n", "default"))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      "No taskruns found\n",
		})
	})

	t.Run("Listing down  taskrun list under default namespace (foo)", func(t *testing.T) {
		res := icmd.RunCmd(run("taskrun", "list", "-n", "foo"))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      "No taskruns found\n",
		})
	})

	t.Run("Listing down taskRun list from configured cluster", func(t *testing.T) {
		res := icmd.RunCmd(run("taskrun", "list", "-k", os.Getenv("HOME")+"/.kube/config"))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      "No taskruns found\n",
		})
	})

	t.Run("check for taskrun logs under namespaces (foo)", func(t *testing.T) {
		res := icmd.RunCmd(run("taskrun", "logs", "-n", "foo"))
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "Error: accepts 1 arg(s), received 0\n",
		})
	})

	t.Run("Check for taskrun logs under namespaces (bar) using follow mode ", func(t *testing.T) {
		res := icmd.RunCmd(run("taskrun", "logs", "-f", "foo", "-n", "bar"))
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `Error: Unable to get Taskrun : taskruns.tekton.dev "foo" not found` + "\n",
		})
	})

	t.Run("check for taskrun logs without enough arguments", func(t *testing.T) {
		res := icmd.RunCmd(run("taskrun", "logs"))
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "accepts 1 arg(s), received 0\n",
		})
	})

	t.Run("check for taskrun Logs from configured cluster", func(t *testing.T) {
		res := icmd.RunCmd(run("taskrun", "logs", "foo", "-k", os.Getenv("HOME")+"/.kube/config"))
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `Error: Unable to get Taskrun : taskruns.tekton.dev "foo" not found` + "\n",
		})
	})

	t.Run("Get help to  delete/rm pipeline resources delete /rm command ", func(t *testing.T) {
		res := i.cmd.RunCmd(run("resource", "delete", "-h"))
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
		})
		expected := `Delete a pipeline resource in a namespace

Usage:
tkn resource delete [flags]

Aliases:
  delete, rm

Examples:

# Delete a PipelineResource of name 'foo' in namespace 'bar'
tkn resource delete foo -n bar

tkn res rm foo -n bar


Flags:
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -f, --force                         Whether to force deletion (default: false)
  -h, --help                          help for delete
  -o, --output string                 Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-file.
	  --template string               Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].
	  
Global Flags:
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
`
		assert.Assert(t, is.Equal(expected, res.Stderr()))
	})

}
