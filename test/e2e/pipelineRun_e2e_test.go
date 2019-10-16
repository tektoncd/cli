// +build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/icmd"
	knativetest "knative.dev/pkg/test"
)



func TestPipelineRunE2EUsingCli(t *testing.T) {

	t.Parallel()
	c, namespace := Setup(t)
	knativetest.CleanupOnInterrupt(func() { TearDown(t, c, namespace) }, t.Logf)
	defer TearDown(t, c, namespace)

	t.Logf("Creating Git PipelineResource %s", tePipelineGitResourceName)
	if _, err := c.PipelineResourceClient.Create(getGitResourceForOutPutPipeline(tePipelineGitResourceName, namespace)); err != nil {
		t.Fatalf("Failed to create Pipeline Resource `%s`: %s", tePipelineGitResourceName, err)
	}

	t.Logf("Creating (Fault) Git PipelineResource %s", tePipelineFaultGitResourceName)
	if _, err := c.PipelineResourceClient.Create(getFaultGitResource(tePipelineFaultGitResourceName, namespace)); err != nil {
		t.Fatalf("Failed to create fault Pipeline Resource `%s`: %s", tePipelineFaultGitResourceName, err)
	}

	t.Logf("Creating Task  %s", TaskName1)
	if _, err := c.TaskClient.Create(getCreateFileTask(TaskName1, namespace)); err != nil {
		t.Fatalf("Failed to create Task Resource `%s`: %s", TaskName1, err)
	}

	t.Logf("Creating Task  %s", TaskName2)
	if _, err := c.TaskClient.Create(getReadFileTask(TaskName2, namespace)); err != nil {
		t.Fatalf("Failed to create Task Resource `%s`: %s", TaskName2, err)
	}

	t.Logf("Create Pipeline %s", tePipelineName+"-"+strconv.Itoa(4))
	if _, err := c.PipelineClient.Create(getPipeline(tePipelineName+"-"+strconv.Itoa(4), namespace, TaskName1, TaskName2)); err != nil {
		t.Fatalf("Failed to create pipeline `%s`: %s", tePipelineName+"-"+strconv.Itoa(4), err)
	}

	t.Logf("Create Pipeline run %s", tePipelineRunName+"-"+strconv.Itoa(4))
	if _, err := c.PipelineRunClient.Create(getPipelineRun(tePipelineRunName+"-"+strconv.Itoa(4), namespace, "default", tePipelineName+"-"+strconv.Itoa(4), tePipelineGitResourceName)); err != nil {
		t.Fatalf("Failed to create pipeline `%s`: %s", tePipelineRunName+"-"+strconv.Itoa(4), err)
	}

	t.Logf("Create Failure Pipeline run %s", tePipelineRunName+"-"+strconv.Itoa(5))
	if _, err := c.PipelineRunClient.Create(getPipelineRun(tePipelineRunName+"-"+strconv.Itoa(5), namespace, "default", tePipelineName+"-"+strconv.Itoa(4), tePipelineFaultGitResourceName)); err != nil {
		t.Fatalf("Failed to create pipeline `%s`: %s", tePipelineRunName+"-"+strconv.Itoa(5), err)
	}
	time.Sleep(1 * time.Second)

	run := Prepare(t)

	WaitForPipelineRunToComplete(c, tePipelineRunName+"-"+strconv.Itoa(4), namespace)

	t.Run("Get list of Pipeline Runs from namespace  "+namespace, func(t *testing.T) {

		res := icmd.RunCmd(run("pr", "list", "-n", namespace))

		expected := CreateTemplateForPipelineRunListWithTestData(t, c, tePipelineName, map[int]interface{}{
			0: &PipelineRunListData{
				Name:   tePipelineRunName+"-"+strconv.Itoa(4),
				Status: "Succeeded",
			},

			1: &PipelineRunListData{
				Name:   tePipelineRunName + "-" + strconv.Itoa(5),
				Status: "Failed",
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

	t.Run("Get list of Pipelineruns from other namespace [default] should throw Error", func(t *testing.T) {
		res := icmd.RunCmd(run("pr", "list", "-n", "default"))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      "No pipelineruns found\n",
		})

		if d := cmp.Diff("No pipelineruns found\n", res.Stderr()); d != "" {
			t.Errorf("Unexpected output mismatch: \n%s\n", d)
		}
	})

	t.Run("Validate PipelineRun list format for -o (output) flag, as Json Path ", func(t *testing.T) {
		res := icmd.RunCmd(run("pr", "list", "-n", namespace,
			`-o=jsonpath={range.items[*]}{.metadata.name}{"\n"}{end}`))

		expected := CreateTemplateResourcesForOutputpath(
			GetSortedPipelineRunListWithTestData(t, c, tePipelineName, map[int]interface{}{
				0: &PipelineRunListData{
					Name:   tePipelineRunName+"-"+strconv.Itoa(4),
					Status: "Succeeded",
				},

				1: &PipelineRunListData{
					Name:   tePipelineRunName + "-" + strconv.Itoa(5),
					Status: "Failed",
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

	t.Run("Validate PipelineRun Schema for -o (output) flag as Json ", func(t *testing.T) {
		res := icmd.RunCmd(run("pr", "list", "-n", namespace, "-o", "json"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		err := json.Unmarshal([]byte(res.Stdout()), &v1alpha1.PipelineRunList{})
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	})

	t.Run("Remove pipeline Run With force delete flag (shorthand)", func(t *testing.T) {

		res := icmd.RunCmd(run("pr", "rm", tePipelineRunName+"-"+strconv.Itoa(5), "-n", namespace, "-f"))
		fmt.Printf("Output : %+v", res.Stdout())
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      "PipelineRun deleted: " + tePipelineRunName + "-" + strconv.Itoa(5) + "\n",
		})

	})

	t.Run("Check for Pipeline Runs "+tePipelineRunName+"-"+strconv.Itoa(5)+" from namespace  "+namespace+" shouldn't exist", func(t *testing.T) {

		res := icmd.RunCmd(run("pr", "list", "-n", namespace))

		expected := CreateTemplateForPipelineRunListWithTestData(t, c, tePipelineName, map[int]interface{}{
			0: &PipelineRunListData{
				Name:   tePipelineRunName+"-"+strconv.Itoa(4),
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

// 	t.Run("Validate PipelineRun describe command", func(t *testing.T) {
// 		res := icmd.RunCmd(run("pr", "describe", tePipelineRunName, "-n", namespace))

// 		res.Assert(t, icmd.Expected{
// 			ExitCode: 0,
// 			Err:      icmd.None,
// 		})

// 		taskRunlist := GetTaskRunList(c)

// 		sort.Sort(byStartTime(taskRunlist.Items))

// 		expected := CreateDescribeTemplateForPipelineRunWithTestData(t, c, tePipelineRunName,
// 			map[int]interface{}{
// 				0: &PipelineRunDescribeData{
// 					Name:            tePipelineRunName,
// 					Namespace:       namespace,
// 					Pipeline_Ref:    tePipelineName,
// 					Service_Account: "default",
// 					Status:          "Succeeded",
// 					FailureMessage:  "",
// 					Resources: map[int]interface{}{
// 						0: &ResourceRefData{
// 							ResourceName: "source-repo",
// 							ResourceRef:  tePipelineGitResourceName,
// 						},
// 					},
// 					Params: map[string]interface{}{},
// 					TaskRuns: map[int]interface{}{
// 						0: &TaskRunRefData{
// 							TaskRunName: taskRunlist.Items[1].Name,
// 							TaskRef:     "first-create-file",
// 							Status:      "Succeeded",
// 						},
// 						1: &TaskRunRefData{
// 							TaskRunName: taskRunlist.Items[0].Name,
// 							TaskRef:     "then-check",
// 							Status:      "Succeeded",
// 						},
// 					}},
// 			})
// 		if d := cmp.Diff(expected, res.Stdout()); d != "" {
// 			t.Errorf("Unexpected output mismatch: \n%s\n", d)
// 		}

// 	})

	t.Run("Validate PipelineRun logs using follow flag (-f), which streams logs to console ", func(t *testing.T) {

		expected := []string{`.*(\[first-create-file : read-docs-old\].*/workspace/damnworkspace/docs/README.md)`, `.*(\[then-check : read\].*some stuff).*?`}

		res := icmd.RunCmd(run("pr", "logs", "-f", tePipelineRunName+"-"+strconv.Itoa(4), "-n", namespace))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		for _, log := range expected {
			assert.Assert(t, is.Regexp(log, res.Stdout()))
		}

	})

	t.Run("Validate PipelineRun logs using  flag (-a), gets logs from all containers eg., like nop ", func(t *testing.T) {

		expected := []string{`.*(\[first-create-file : read-docs-old\].*/workspace/damnworkspace/docs/README.md)`, `.*(\[then-check : read\].*some stuff).*?`}

		res := icmd.RunCmd(run("pr", "logs", "-a", tePipelineRunName+"-"+strconv.Itoa(4), "-n", namespace))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		for _, log := range expected {
			assert.Assert(t, is.Regexp(log, res.Stdout()))
		}

	})

	t.Run("Validate PipelineRun logs using  flag (-a), show all logs including init steps injected by tekton", func(t *testing.T) {

		expected := []string{`.*(\[first-create-file : read-docs-old\].*/workspace/damnworkspace/docs/README.md)`, `.*(\[then-check : read\].*some stuff).*?`}

		res := icmd.RunCmd(run("pr", "logs", "-a", tePipelineRunName+"-"+strconv.Itoa(4), "-n", namespace))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		for _, log := range expected {
			assert.Assert(t, is.Regexp(log, res.Stdout()))
		}

	})

	t.Run("Validate PipelineRun logs for specified task only using (-t) flag ", func(t *testing.T) {

		expected := []string{`.*(\[then-check : read\].*some stuff).*?`}

		res := icmd.RunCmd(run("pr", "logs", tePipelineRunName+"-"+strconv.Itoa(4), "-t", "then-check", "-n", namespace))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		for _, log := range expected {
			assert.Assert(t, is.Regexp(log, res.Stdout()))
		}

	})

	t.Run("Validate PipelineRun logs for specified task only using (-t) flag ", func(t *testing.T) {

		expected := []string{`.*(\[first-create-file : read-docs-old\].*/workspace/damnworkspace/docs/README.md).*`}

		res := icmd.RunCmd(run("pr", "logs", tePipelineRunName+"-"+strconv.Itoa(4), "-t", "first-create-file", "-n", namespace))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		for _, log := range expected {
			assert.Assert(t, is.Regexp(log, res.Stdout()))
		}

	})

}

func TestPipelineRunCancelAndDeleteUsingCli(t *testing.T) {

	t.Parallel()
	c, namespace := Setup(t)
	knativetest.CleanupOnInterrupt(func() { TearDown(t, c, namespace) }, t.Logf)
	defer TearDown(t, c, namespace)

	t.Logf("Creating Git PipelineResource %s", tePipelineGitResourceName)
	if _, err := c.PipelineResourceClient.Create(getGitResourceForOutPutPipeline(tePipelineGitResourceName, namespace)); err != nil {
		t.Fatalf("Failed to create Pipeline Resource `%s`: %s", tePipelineGitResourceName, err)
	}

	t.Logf("Creating (Fault) Git PipelineResource %s", tePipelineFaultGitResourceName)
	if _, err := c.PipelineResourceClient.Create(getFaultGitResource(tePipelineFaultGitResourceName, namespace)); err != nil {
		t.Fatalf("Failed to create fault Pipeline Resource `%s`: %s", tePipelineFaultGitResourceName, err)
	}

	t.Logf("Creating Task  %s", TaskName1)
	if _, err := c.TaskClient.Create(getCreateFileTask(TaskName1, namespace)); err != nil {
		t.Fatalf("Failed to create Task Resource `%s`: %s", TaskName1, err)
	}

	t.Logf("Creating Task  %s", TaskName2)
	if _, err := c.TaskClient.Create(getReadFileTask(TaskName2, namespace)); err != nil {
		t.Fatalf("Failed to create Task Resource `%s`: %s", TaskName2, err)
	}

	t.Logf("Create Pipeline %s", tePipelineName+"-"+strconv.Itoa(6))
	if _, err := c.PipelineClient.Create(getPipeline(tePipelineName+"-"+strconv.Itoa(6), namespace, TaskName1, TaskName2)); err != nil {
		t.Fatalf("Failed to create pipeline `%s`: %s", tePipelineName+"-"+strconv.Itoa(6), err)
	}

	t.Logf("Create Pipeline run %s", tePipelineRunName+"-"+strconv.Itoa(6))
	if _, err := c.PipelineRunClient.Create(getPipelineRun(tePipelineRunName+"-"+strconv.Itoa(6), namespace, "default", tePipelineName+"-"+strconv.Itoa(6), tePipelineGitResourceName)); err != nil {
		t.Fatalf("Failed to create pipeline `%s`: %s", tePipelineRunName+"-"+strconv.Itoa(6), err)
	}

	run := Prepare(t)

	WaitForPipelineRunToStart(c, tePipelineRunName+"-"+strconv.Itoa(6), namespace)

	t.Run("Cancel Running Pipeline Run "+tePipelineRunName+"-"+strconv.Itoa(6)+" in namespace "+namespace, func(t *testing.T) {

		res := icmd.RunCmd(run("pr", "cancel", tePipelineRunName+"-"+strconv.Itoa(6), "-n", namespace))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      "Pipelinerun cancelled: " + tePipelineRunName+"-"+strconv.Itoa(6) + "\n",
		})

	})

	time.Sleep(2 * time.Second)

	t.Run("Cancel Running Pipeline Run "+tePipelineRunName+"-"+strconv.Itoa(6)+" in another namespace default", func(t *testing.T) {

		res := icmd.RunCmd(run("pr", "cancel", tePipelineRunName+"-"+strconv.Itoa(6), "-n", "default"))

		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "Error: failed to find pipelinerun: " + tePipelineRunName+"-"+strconv.Itoa(6) + "\n",
		})

	})

	t.Run("Validate PipelineRun describe command", func(t *testing.T) {
		res := icmd.RunCmd(run("pr", "describe", tePipelineRunName+"-"+strconv.Itoa(6), "-n", namespace))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})

		taskRunlist := GetTaskRunList(c)

		sort.Sort(byStartTime(taskRunlist.Items))

		expected := CreateDescribeTemplateForPipelineRunWithTestData(t, c, tePipelineRunName+"-"+strconv.Itoa(6),
			map[int]interface{}{
				0: &PipelineRunDescribeData{
					Name:            tePipelineRunName+"-"+strconv.Itoa(6),
					Namespace:       namespace,
					Pipeline_Ref:    tePipelineName+"-"+strconv.Itoa(6),
					Service_Account: "default",
					Status:          "PipelineRunCancelled",
					FailureMessage:  "PipelineRun \"" + tePipelineRunName+"-"+strconv.Itoa(6) + "\" was cancelled (TaskRun \"" + taskRunlist.Items[0].Name + "\" was cancelled)",
					Resources: map[int]interface{}{
						0: &ResourceRefData{
							ResourceName: "source-repo",
							ResourceRef:  tePipelineGitResourceName,
						},
					},
					Params: map[string]interface{}{},
					TaskRuns: map[int]interface{}{
						0: &TaskRunRefData{
							TaskRunName: taskRunlist.Items[0].Name,
							TaskRef:     "first-create-file",
							Status:      "TaskRunCancelled",
						},
					}},
			})
		if d := cmp.Diff(expected, res.Stdout()); d != "" {
			t.Errorf("Unexpected output mismatch: \n%s\n", d)
		}
	})

	t.Run("Remove pipeline Run Without force delete flag, reply no", func(t *testing.T) {

		res := icmd.RunCmd(run("pr", "rm", tePipelineRunName+"-"+strconv.Itoa(6), "-n", namespace),
			icmd.WithStdin(strings.NewReader("n")))
		fmt.Printf("Output : %+v", res.Stdout())
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "Error: canceled deleting pipelinerun \"" + tePipelineRunName+"-"+strconv.Itoa(6) + "\"\n",
		})

	})

	t.Run("Remove pipeline Run Without force delete flag, reply yes", func(t *testing.T) {

		res := icmd.RunCmd(run("pr", "rm", tePipelineRunName+"-"+strconv.Itoa(6), "-n", namespace),
			icmd.WithStdin(strings.NewReader("y")))
		fmt.Printf("Output : %+v", res.Stdout())
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      "PipelineRun deleted: " + tePipelineRunName+"-"+strconv.Itoa(6) + "\n",
		})

	})

	t.Run("Check for deleted Pipelineruns in  namespace "+namespace+" should throw error", func(t *testing.T) {
		res := icmd.RunCmd(run("pr", "list", "-n", namespace))
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      "No pipelineruns found\n",
		})

		if d := cmp.Diff("No pipelineruns found\n", res.Stderr()); d != "" {
			t.Errorf("Unexpected output mismatch: \n%s\n", d)
		}
	})

}
