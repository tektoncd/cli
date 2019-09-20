// +build e2e

package e2e

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tb "github.com/tektoncd/pipeline/test/builder"
	"gotest.tools/icmd"
	knativetest "knative.dev/pkg/test"
)

const (
	TaskName1                      = "create-file"
	TaskName2                      = "check-stuff-file-exists"
	tePipelineName                 = "output-pipeline"
	tePipelineRunName              = "output-pipeline-run"
	tePipelineGitResourceName      = "skaffold-git"
	tePipelineFaultGitResourceName = "skaffold-git-1"
)

func TestPipelinesE2EUsingCli(t *testing.T) {
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

	t.Logf("Create Pipeline %s", tePipelineName)
	if _, err := c.PipelineClient.Create(getPipeline(tePipelineName, namespace, TaskName1, TaskName2)); err != nil {
		t.Fatalf("Failed to create pipeline `%s`: %s", tePipelineName, err)
	}

	time.Sleep(1 * time.Second)

	run := Prepare(t)

	t.Run("Get list of Pipelines from namespace  "+namespace, func(t *testing.T) {

		res := icmd.RunCmd(run("pipelines", "list", "-n", namespace))

		expected := CreateTemplateForPipelineListWithTestData(t, c, map[int]interface{}{
			0: &PipelinesListData{
				Name:   tePipelineName,
				Status: "---",
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
	// Bug to fix
	t.Run("Get list of pipelines from other namespace [default] should throw Error", func(t *testing.T) {

		res := icmd.RunCmd(run("pipelines", "list", "-n", "default"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "No pipelines\n",
			Err:      icmd.None,
		})
	})

	t.Run("Validate pipelines format for -o (output) flag, as Json Path", func(t *testing.T) {

		res := icmd.RunCmd(run("pipelines", "list", "-n", namespace,
			`-o=jsonpath={range.items[*]}{.metadata.name}{"\n"}{end}`))

		expected := CreateTemplateResourcesForOutputpath(
			GetPipelineListWithTestData(t, c,
				map[int]interface{}{
					0: &PipelinesListData{
						Name:   tePipelineName,
						Status: "---",
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

	t.Run("Pipeline json Schema validation with -o (output) flag, as Json ", func(t *testing.T) {
		res := icmd.RunCmd(run("pipelines", "list", "-n", namespace, "-o", "json"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		err := json.Unmarshal([]byte(res.Stdout()), &v1alpha1.PipelineList{})
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	})

	t.Run("Validate Pipeline describe command in namespace "+namespace, func(t *testing.T) {
		res := icmd.RunCmd(run("pipeline", "describe", tePipelineName, "-n", namespace))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})

		expected := CreateTemplateForPipelinesDescribeWithTestData(t, c, tePipelineName,
			map[int]interface{}{
				0: &PipelineDescribeData{
					Name: tePipelineName,
					Resources: map[string]string{
						"source-repo": "git",
					},
					Task: map[int]interface{}{
						0: &TaskRefData{
							TaskName: "first-create-file",
							TaskRef:  TaskName1,
							RunAfter: nil,
						},
						1: &TaskRefData{
							TaskName: "then-check",
							TaskRef:  TaskName2,
							RunAfter: nil,
						},
					},
					Runs: map[string]string{},
				},
			})
		if d := cmp.Diff(expected, res.Stdout()); d != "" {
			t.Errorf("Unexpected output mismatch: \n%s\n", d)
		}
	})

	vars := make(map[string]interface{})
	var pipelineGeneratedName []string
	var prnamegenerated string
	var pipelineRunStatus []string

	t.Run("Start Pipeline Run using pipeline start command with SA as default ", func(t *testing.T) {

		res := icmd.RunCmd(run("pipeline", "start", tePipelineName,
			"-r=source-repo=skaffold-git",
			"-s=default",
			"-n", namespace))

		time.Sleep(1 * time.Second)

		prnamegenerated = GetPipelineRunListWithName(c, tePipelineName).Items[0].Name
		vars["Element"] = prnamegenerated
		expected := ProcessString(`Pipelinerun started: {{.Element}}

In order to track the pipelinerun progress run:
tkn pipelinerun logs -n `+namespace+` {{.Element}} -f
`, vars)

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      expected,
			Err:      icmd.None,
		})
	})

	pipelineGeneratedName = append(pipelineGeneratedName, prnamegenerated)
	pipelineRunStatus = append(pipelineRunStatus, "Succeeded")

	t.Run("Start Pipeline Run using pipeline start with falut gitResource, pipeline run status should be failed  ", func(t *testing.T) {

		res := icmd.RunCmd(run("pipeline", "start", tePipelineName,
			"-r=source-repo=skaffold-git-1",
			"-s=default",
			"-n", namespace))

		time.Sleep(1 * time.Second)

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
	})

	WaitForPipelineRunToComplete(c, prnamegenerated, namespace)

	time.Sleep(1 * time.Second)

	for _, pr := range GetPipelineRunList(c).Items {

		if pipelineGeneratedName[0] != pr.Name {
			pipelineGeneratedName = append(pipelineGeneratedName, pr.Name)
			pipelineRunStatus = append(pipelineRunStatus, pr.Status.Conditions[0].Reason)
		}

	}

	// t.Run("Validate Pipeline describe command in namespace "+namespace+" after PipelineRun completed successfully", func(t *testing.T) {
	// 	res := icmd.RunCmd(run("pipeline", "describe", tePipelineName, "-n", namespace))

	// 	res.Assert(t, icmd.Expected{
	// 		ExitCode: 0,
	// 		Err:      icmd.None,
	// 	})

	// 	expected := CreateTemplateForPipelinesDescribeWithTestData(t, c, tePipelineName,
	// 		map[int]interface{}{
	// 			0: &PipelineDescribeData{
	// 				Name: tePipelineName,
	// 				Resources: map[string]string{
	// 					"source-repo": "git",
	// 				},
	// 				Task: map[int]interface{}{
	// 					0: &TaskRefData{
	// 						TaskName: "first-create-file",
	// 						TaskRef:  TaskName1,
	// 						RunAfter: nil,
	// 					},
	// 					1: &TaskRefData{
	// 						TaskName: "then-check",
	// 						TaskRef:  TaskName2,
	// 						RunAfter: nil,
	// 					},
	// 				},
	// 				Runs: map[string]string{
	// 					pipelineGeneratedName[0]: pipelineRunStatus[0],
	// 					pipelineGeneratedName[1]: pipelineRunStatus[1],
	// 				},
	// 			},
	// 		})
	// 	if d := cmp.Diff(expected, res.Stdout()); d != "" {
	// 		expected := CreateTemplateForPipelinesDescribeWithTestData(t, c, tePipelineName,
	// 			map[int]interface{}{
	// 				0: &PipelineDescribeData{
	// 					Name: tePipelineName,
	// 					Resources: map[string]string{
	// 						"source-repo": "git",
	// 					},
	// 					Task: map[int]interface{}{
	// 						0: &TaskRefData{
	// 							TaskName: "first-create-file",
	// 							TaskRef:  TaskName1,
	// 							RunAfter: nil,
	// 						},
	// 						1: &TaskRefData{
	// 							TaskName: "then-check",
	// 							TaskRef:  TaskName2,
	// 							RunAfter: nil,
	// 						},
	// 					},
	// 					Runs: map[string]string{
	// 						pipelineGeneratedName[1]: pipelineRunStatus[1],
	// 						pipelineGeneratedName[0]: pipelineRunStatus[0],
	// 					},
	// 				},
	// 			})

	// 		if d := cmp.Diff(expected, res.Stdout()); d != "" {
	// 			t.Errorf("Unexpected output mismatch: \n%s\n", d)
	// 		}

	// 	}
	// })

	t.Run("Validate interactive pipeline logs, with  follow mode (-f) ", func(t *testing.T) {

		td := &InteractiveTestData{
			Cmd: []string{"pipeline", "logs", "-f", "-n", namespace},
			OpsVsExpected: []interface{}{
				&Operation{
					Ops:      string(terminal.KeyEnter),
					Expected: tePipelineName,
				},
				&Operation{
					Ops:      string(terminal.KeyArrowDown),
					Expected: pipelineGeneratedName[1],
				},
				&Operation{
					Ops:      string(terminal.KeyEnter),
					Expected: pipelineGeneratedName[0],
				},
			},

			ExpectedLogs: []string{`.*(\[first-create-file : read-docs-old\].*/workspace/damnworkspace/docs/README.md).*`, `.*(\[then-check : read\] some stuff).*`},
		}
		RunInteractivePipelineLogs(t, namespace, td)
	})
}

func TestDeletePipelinesE2EUsingCli(t *testing.T) {
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
	for i := 1; i <= 3; i++ {
		t.Logf("Create Pipeline %s", tePipelineName+"-"+strconv.Itoa(i))
		if _, err := c.PipelineClient.Create(getPipeline(tePipelineName+"-"+strconv.Itoa(i), namespace, TaskName1, TaskName2)); err != nil {
			t.Fatalf("Failed to create pipeline `%s`: %s", tePipelineName+"-"+strconv.Itoa(i), err)
		}
	}

	time.Sleep(1 * time.Second)

	run := Prepare(t)

	t.Run("Delete pipeline "+tePipelineName+"-1"+" from namespace "+namespace+" With force delete flag (shorthand)", func(t *testing.T) {
		res := icmd.RunCmd(run("pipeline", "rm", tePipelineName+"-1", "-n", namespace, "-f"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "Pipeline deleted: " + tePipelineName + "-1" + "\n",
		})
	})

	t.Run("Delete pipeline "+tePipelineName+"-2"+" from namespace "+namespace+" With force delete flag", func(t *testing.T) {
		res := icmd.RunCmd(run("pipeline", "rm", tePipelineName+"-2", "-n", namespace, "--force"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "Pipeline deleted: " + tePipelineName + "-2" + "\n",
		})
	})

	t.Run("Delete pipeline "+tePipelineName+"-3"+" from namespace "+namespace+" without force flag, reply no", func(t *testing.T) {
		res := icmd.RunCmd(run("pipeline", "rm", tePipelineName+"-3", "-n", namespace),
			icmd.WithStdin(strings.NewReader("n")))

		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "Error: canceled deleting pipeline \"" + tePipelineName + "-3" + "\"\n",
		})
	})

	t.Run("Delete pipeline "+tePipelineName+"-3"+" from namespace "+namespace+" without force flag, reply yes", func(t *testing.T) {
		res := icmd.RunCmd(run("pipeline", "rm", tePipelineName+"-3", "-n", namespace),
			icmd.WithStdin(strings.NewReader("y")))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      "Are you sure you want to delete pipeline \"" + tePipelineName + "-3" + "\" (y/n): Pipeline deleted: " + tePipelineName + "-3" + "\n",
		})

	})

	t.Run("Check for list of pipelines, After Successfull Deletion of pipeline in namespace "+namespace+" should throw an error", func(t *testing.T) {
		res := icmd.RunCmd(run("pipelines", "list", "-n", namespace))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "No pipelines\n",
			Err:      icmd.None,
		})
	})

}

func getGitResourceForOutPutPipeline(rname string, namespace string) *v1alpha1.PipelineResource {
	return tb.PipelineResource(rname, namespace, tb.PipelineResourceSpec(
		v1alpha1.PipelineResourceTypeGit,
		tb.PipelineResourceSpecParam("url", "https://github.com/GoogleContainerTools/skaffold"),
		tb.PipelineResourceSpecParam("revision", "master"),
	))
}

func getFaultGitResource(rname string, namespace string) *v1alpha1.PipelineResource {
	return tb.PipelineResource(rname, namespace, tb.PipelineResourceSpec(
		v1alpha1.PipelineResourceTypeGit,
		tb.PipelineResourceSpecParam("url", "https://github.com/GoogleContainerTools/skaffold-1"),
		tb.PipelineResourceSpecParam("revision", "master"),
	))
}

func getCreateFileTask(taskname string, namespace string) *v1alpha1.Task {

	taskSpecOps := []tb.TaskSpecOp{
		tb.TaskInputs(tb.InputsResource("workspace", v1alpha1.PipelineResourceTypeGit, tb.ResourceTargetPath("damnworkspace"))),
		tb.TaskOutputs(tb.OutputsResource("workspace", v1alpha1.PipelineResourceTypeGit)),
		tb.Step("read-docs-old", "ubuntu", tb.StepCommand("/bin/bash"), tb.StepArgs("-c", "ls -la /workspace/damnworkspace/docs/README.md")),
		tb.Step("write-new-stuff", "ubuntu", tb.StepCommand("bash"), tb.StepArgs("-c", "echo some stuff > /workspace/damnworkspace/stuff")),
	}

	return tb.Task(taskname, namespace, tb.TaskSpec(taskSpecOps...))
}

func getReadFileTask(taskname string, namespace string) *v1alpha1.Task {

	taskSpecOps := []tb.TaskSpecOp{
		tb.TaskInputs(tb.InputsResource("workspace", v1alpha1.PipelineResourceTypeGit, tb.ResourceTargetPath("newworkspace"))),
		tb.Step("read", "ubuntu", tb.StepCommand("/bin/bash"), tb.StepArgs("-c", "cat /workspace/newworkspace/stuff")),
	}

	return tb.Task(taskname, namespace, tb.TaskSpec(taskSpecOps...))
}

func getPipeline(PipelineName string, namespace string, createFiletaskName string, readFileTaskName string) *v1alpha1.Pipeline {

	pipelineSpec := []tb.PipelineSpecOp{
		tb.PipelineDeclaredResource("source-repo", "git"),
		tb.PipelineTask("first-create-file", createFiletaskName,
			tb.PipelineTaskInputResource("workspace", "source-repo"),
			tb.PipelineTaskOutputResource("workspace", "source-repo"),
		),
		tb.PipelineTask("then-check", readFileTaskName,
			tb.PipelineTaskInputResource("workspace", "source-repo", tb.From("first-create-file")),
		),
	}

	return tb.Pipeline(PipelineName, namespace, tb.PipelineSpec(pipelineSpec...))
}

func getPipelineRun(PipelineRunName string, namespace string, serviceAccount string, pipelineName string, pipelineResourceName string) *v1alpha1.PipelineRun {
	return tb.PipelineRun(PipelineRunName, namespace,
		tb.PipelineRunSpec(pipelineName,
			tb.PipelineRunServiceAccount(serviceAccount),
			tb.PipelineRunResourceBinding("source-repo", tb.PipelineResourceBindingRef(pipelineResourceName)),
		))
}
