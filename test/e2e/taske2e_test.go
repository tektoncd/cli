// +build e2e

package e2e

import (
	"encoding/json"
	"log"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"gotest.tools/icmd"
	knativetest "knative.dev/pkg/test"
)

func TestTaskListe2eUsingCli(t *testing.T) {
	t.Helper()
	t.Parallel()

	namespace, c := CreateTask(t, teTaskName)

	knativetest.CleanupOnInterrupt(func() { TearDown(t, c, namespace) }, t.Logf)
	defer TearDown(t, c, namespace)
	run := Prepare(t)

	t.Run("Get all Task List from namespace  "+namespace, func(t *testing.T) {
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

	t.Run("Get Task list from namespace [default] should throw Error", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", "default"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      "No tasks found\n",
		})

		if d := cmp.Diff("No tasks found\n", res.Stderr()); d != "" {
			t.Errorf("Unexpected output mismatch: \n%s\n", d)
		}
	})

	t.Run("Validate Task List format based on Json Output Path ", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", namespace, `-o=jsonpath={range.items[*]}{.metadata.name}{"\n"}{end}`))

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

	t.Run("Validate Task List format based on Json Output Path ", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", namespace, "-o", "json"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		err := json.Unmarshal([]byte(res.Stdout()), &v1alpha1.TaskList{})
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	})
}
func TestTaskListe2eUsingCli_1(t *testing.T) {
	t.Helper()
	t.Parallel()

	namespace, c := CreateTask(t, "te-busybox")
	knativetest.CleanupOnInterrupt(func() { TearDown(t, c, namespace) }, t.Logf)
	defer TearDown(t, c, namespace)
	run := Prepare(t)

	t.Run("Get all Task List from namespace  "+namespace, func(t *testing.T) {

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
			t.Errorf("Unexpected output mismatch: \n%s\n", d)
		}
	})

	t.Run("Get Task list from namespace [default] should throw Error", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", "default"))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      "No tasks found\n",
		})

		if d := cmp.Diff("No tasks found\n", res.Stderr()); d != "" {
			t.Errorf("Unexpected output mismatch: \n%s\n", d)
		}
	})

	t.Run("Validate Task List format based on Json Output Path ", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", namespace, `-o=jsonpath={range.items[*]}{.metadata.name}{"\n"}{end}`))

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

	t.Run("Validate Task List format based on Json Output Path ", func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", namespace, "-o", "json"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		err := json.Unmarshal([]byte(res.Stdout()), &v1alpha1.TaskList{})
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	})

}
