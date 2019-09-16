// +build e2e

package e2e

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tb "github.com/tektoncd/pipeline/test/builder"
	"gotest.tools/icmd"
	knativetest "knative.dev/pkg/test"
)

func TestPipelineResourceDelete(t *testing.T) {
	PipelineResourceName := []string{"go-example-git", "go-example-git-1", "go-example-git-2"}
	c, namespace := Setup(t)
	knativetest.CleanupOnInterrupt(func() { TearDown(t, c, namespace) }, t.Logf)
	defer TearDown(t, c, namespace)

	for _, prname := range PipelineResourceName {
		t.Logf("Creating Git PipelineResource %s", prname)
		if _, err := c.PipelineResourceClient.Create(getGitResource(prname, namespace)); err != nil {
			t.Fatalf("Failed to create Pipeline Resource `%s`: %s", prname, err)
		}
	}

	t.Run("Remove pipeline resources With force delete flag (shorthand)", func(t *testing.T) {
		run := Prepare(t)
		res := icmd.RunCmd(run("resource", "rm", PipelineResourceName[0], "-n", namespace, "-f"))
		fmt.Printf("Output : %+v", res.Stdout())
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      "PipelineResource deleted: " + PipelineResourceName[0] + "\n",
		})

	})

	t.Run("Remove pipeline resources With force delete flag", func(t *testing.T) {
		run := Prepare(t)
		res := icmd.RunCmd(run("resource", "rm", PipelineResourceName[1], "-n", namespace, "--force"))
		fmt.Printf("Output : %+v", res.Stdout())
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      "PipelineResource deleted: " + PipelineResourceName[1] + "\n",
		})

	})

	t.Run("Remove pipeline resources Without force delete flag, reply no", func(t *testing.T) {
		run := Prepare(t)
		res := icmd.RunCmd(run("resource", "rm", PipelineResourceName[2], "-n", namespace), icmd.WithStdin(strings.NewReader("n")))
		fmt.Printf("Output : %+v", res.Stdout())
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "Error: Canceled deleting pipelineresource \"" + PipelineResourceName[2] + "\"\n",
		})

	})

	t.Run("Remove pipeline resources Without force delete flag, reply yes", func(t *testing.T) {
		run := Prepare(t)
		res := icmd.RunCmd(run("resource", "rm", PipelineResourceName[2], "-n", namespace), icmd.WithStdin(strings.NewReader("y")))
		fmt.Printf("Output : %+v", res.Stdout())
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      "Are you sure you want to delete pipelineresource \"" + PipelineResourceName[2] + "\" (y/n): PipelineResource deleted: " + PipelineResourceName[2] + "\n",
		})

	})
}

func getGitResource(prname string, namespace string) *v1alpha1.PipelineResource {
	return tb.PipelineResource(prname, namespace, tb.PipelineResourceSpec(
		v1alpha1.PipelineResourceTypeGit,
		tb.PipelineResourceSpecParam("Url", "https://github.com/GoogleContainerTools/kaniko"),
	))
}
