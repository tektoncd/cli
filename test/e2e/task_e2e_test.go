// +build e2e

package e2e

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tb "github.com/tektoncd/pipeline/test/builder"
	"gotest.tools/icmd"
	knativetest "knative.dev/pkg/test"
)

func TestTaskListe2eUsingCli(t *testing.T) {
	t.Helper()
	t.Parallel()

	c, namespace := Setup(t)
	knativetest.CleanupOnInterrupt(func() { TearDown(t, c, namespace) }, t.Logf)
	defer TearDown(t, c, namespace)

	t.Logf("Creating Task Resource %s", teTaskName)
	if _, err := c.TaskClient.Create(getTask(teTaskName, namespace)); err != nil {
		t.Fatalf("Failed to create Task Resource `%s`: %s", teTaskName, err)
	}
	time.Sleep(1 * time.Second)

	run := Prepare(t)

	t.Run("Get list of Tasks from namespace  "+namespace, func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", namespace))

		expected := CreateTemplateForTaskListWithTestData(t, c, map[int]interface{}{
			0: &TaskData{
				Name: teTaskName,
			},
		})

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		if d := cmp.Diff(expected, res.Stdout()); d != "" {
			t.Errorf("Unexpected output myismatch: \n%s\n", d)
		}
	})

	t.Run("Get list of Task from other namespace [default] should throw Error", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", "default"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      "No tasks found\n",
		})

		if d := cmp.Diff("No tasks found\n", res.Stderr()); d != "" {
			t.Errorf("Unexpected output mismatch: \n%s\n", d)
		}
	})

	t.Run("Validate Tasks format for -o (output) flag, as Json Path ", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", namespace,
			`-o=jsonpath={range.items[*]}{.metadata.name}{"\n"}{end}`))

		expected := CreateTemplateResourcesForOutputpath(GetTaskListWithTestData(t, c, map[int]interface{}{
			0: &TaskData{
				Name: teTaskName,
			},
		}))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		if d := cmp.Diff(expected, res.Stdout()); d != "" {
			t.Errorf("Unexpected output mismatch: \n%s\n", d)
		}
	})

	t.Run("Validate Taskruns Schema for -o (output) flag as Json ", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", namespace, "-o", "json"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		err := json.Unmarshal([]byte(res.Stdout()), &v1alpha1.TaskList{})
		if err != nil {
			t.Errorf("error: %v", err)
		}
	})

	t.Run("Delete Task "+teTaskName+" from namespace "+namespace+" without force flag, reply no", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "rm", teTaskName, "-n", namespace),
			icmd.WithStdin(strings.NewReader("n")))

		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "Error: canceled deleting task \"" + teTaskName + "\"\n",
		})

	})

	t.Run("Delete Task "+teTaskName+" from namespace "+namespace+" without force flag, reply yes", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "rm", teTaskName, "-n", namespace),
			icmd.WithStdin(strings.NewReader("y")))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      "Are you sure you want to delete task \"" + teTaskName + "\" (y/n): Task deleted: " + teTaskName + "\n",
		})

	})
}
func TestTaskListe2eUsingCli_1(t *testing.T) {
	t.Helper()
	t.Parallel()

	c, namespace := Setup(t)
	knativetest.CleanupOnInterrupt(func() { TearDown(t, c, namespace) }, t.Logf)
	defer TearDown(t, c, namespace)

	t.Logf("Creating Task Resource %s", "te-busybox")
	if _, err := c.TaskClient.Create(getTask("te-busybox", namespace)); err != nil {
		t.Fatalf("Failed to create Task Resource `%s`: %s", "te-busybox", err)
	}
	time.Sleep(1 * time.Second)

	run := Prepare(t)

	t.Run("Get list of Tasks from namespace  "+namespace, func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", namespace))

		expected := CreateTemplateForTaskListWithTestData(t, c, map[int]interface{}{
			0: &TaskData{
				Name: "te-busybox",
			},
		})

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		if d := cmp.Diff(expected, res.Stdout()); d != "" {
			t.Errorf("Unexpected output myismatch: \n%s\n", d)
		}
	})

	t.Run("Get list of Task from other namespace [default] should throw Error", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", "default"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      "No tasks found\n",
		})

		if d := cmp.Diff("No tasks found\n", res.Stderr()); d != "" {
			t.Errorf("Unexpected output mismatch: \n%s\n", d)
		}
	})

	t.Run("Validate Tasks format for -o (output) flag, as Json Path ", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", namespace,
			`-o=jsonpath={range.items[*]}{.metadata.name}{"\n"}{end}`))

		expected := CreateTemplateResourcesForOutputpath(GetTaskListWithTestData(t, c, map[int]interface{}{
			0: &TaskData{
				Name: "te-busybox",
			},
		}))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		if d := cmp.Diff(expected, res.Stdout()); d != "" {
			t.Errorf("Unexpected output mismatch: \n%s\n", d)
		}
	})

	t.Run("Validate Taskruns Schema for -o (output) flag as Json ", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", namespace, "-o", "json"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		err := json.Unmarshal([]byte(res.Stdout()), &v1alpha1.TaskList{})
		if err != nil {
			t.Errorf("error: %v", err)
		}
	})

	t.Run("Delete Task "+"te-busybox"+" from namespace "+namespace+" without force flag, reply no", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "rm", "te-busybox", "-n", namespace),
			icmd.WithStdin(strings.NewReader("n")))

		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "Error: canceled deleting task \"" + "te-busybox" + "\"\n",
		})

	})

	t.Run("Delete Task "+"te-busybox"+" from namespace "+namespace+" without force flag, reply yes", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "rm", "te-busybox", "-n", namespace),
			icmd.WithStdin(strings.NewReader("y")))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      "Are you sure you want to delete task \"" + "te-busybox" + "\" (y/n): Task deleted: " + "te-busybox" + "\n",
		})

	})

}

func getTask(taskname string, namespace string) *v1alpha1.Task {
	return tb.Task(taskname, namespace,
		tb.TaskSpec(tb.Step("amazing-busybox", "busybox", tb.StepCommand("/bin/sh"), tb.StepArgs("-c", "echo Hello")),
			tb.Step("amazing-busybox-1", "busybox", tb.StepCommand("/bin/sh"), tb.StepArgs("-c", "echo Welcome To Tekton!!")),
		))
}
