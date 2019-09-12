// +build e2e

package e2e

import (
	"encoding/json"
	"log"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/icmd"
	knativetest "knative.dev/pkg/test"
)

func TestTaskRunListFunctionalitiesUsingCli(t *testing.T) {
	t.Helper()
	t.Parallel()

	namespace, c := CreateTaskRun(t, teTaskName, teTaskRunName)

	knativetest.CleanupOnInterrupt(func() { TearDown(t, c, namespace) }, t.Logf)
	defer TearDown(t, c, namespace)

	run := Prepare(t)

	t.Run("Get all Taskrun List from namespace  "+namespace, func(t *testing.T) {

		res := icmd.RunCmd(run("taskrun", "list", "-n", namespace))

		expected := CreateTemplateForTaskRunListWithMockData(t, c, map[int]interface{}{
			0: &TaskRunData{
				Name:   teTaskRunName,
				Status: "Succeeded",
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

	t.Run("Get Taskrun list from namespace [default] should throw Error", func(t *testing.T) {
		res := icmd.RunCmd(run("taskrun", "list", "-n", "default"))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      "No taskruns found\n",
		})

		if d := cmp.Diff("No taskruns found\n", res.Stderr()); d != "" {
			t.Errorf("Unexpected output mismatch: \n%s\n", d)
		}
	})

	t.Run("Validate Taskrun List format based on Json Output Path ", func(t *testing.T) {
		res := icmd.RunCmd(run("taskrun", "list", "-n", namespace,
			`-o=jsonpath={range.items[*]}{.metadata.name}{"\n"}{end}`))

		expected := CreateTemplateResourcesForOutputpath(
			GetTaskRunListWithMockData(t, c,
				map[int]interface{}{
					0: &TaskRunData{
						Name:   teTaskRunName,
						Status: "Succeeded",
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

	t.Run("Validate Taskrun List format based on Json Output Path ", func(t *testing.T) {
		res := icmd.RunCmd(run("taskrun", "list", "-n", namespace, "-o", "json"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		err := json.Unmarshal([]byte(res.Stdout()), &v1alpha1.TaskRunList{})
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	})

	t.Run("Validate Taskrun List logs using follow mode ", func(t *testing.T) {

		expected := map[int]interface{}{
			0: string(`.*(\[amazing-busybox\] Hello|\[amazing-busybox-1\] Welcome To Tekton!!).*`),
		}

		for i, tr := range GetTaskRunList(c).Items {
			res := icmd.RunCmd(run("taskrun", "logs", "-f", tr.Name, "-n", namespace))
			res.Assert(t, icmd.Expected{
				ExitCode: 0,
				Err:      icmd.None,
			})

			assert.Assert(t, is.Regexp((expected[i]).(string), res.Stdout()))
		}

	})

	t.Run("Validate Taskrun List logs using all containers mode ", func(t *testing.T) {

		expected := map[int]interface{}{
			0: string(`.*(\[amazing-busybox\] Hello|\[amazing-busybox-1\] Welcome To Tekton!!).*`),
		}

		for i, tr := range GetTaskRunList(c).Items {
			res := icmd.RunCmd(run("taskrun", "logs", "-a", tr.Name, "-n", namespace))

			res.Assert(t, icmd.Expected{
				ExitCode: 0,
				Err:      icmd.None,
			})

			assert.Assert(t, is.Regexp((expected[i]).(string), res.Stdout()))
		}

	})

}
