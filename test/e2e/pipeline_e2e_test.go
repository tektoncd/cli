// Copyright © 2020 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tb "github.com/tektoncd/pipeline/test/builder"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/icmd"
	knativetest "knative.dev/pkg/test"
)

const (
	TaskName1                      = "create-file"
	TaskName2                      = "check-stuff-file-exists"
	tePipelineName                 = "output-pipeline"
	tePipelineGitResourceName      = "skaffold-git"
	tePipelineFaultGitResourceName = "skaffold-git-1"
)

func TestPipelinesE2E(t *testing.T) {
	t.Parallel()
	c, namespace := Setup(t)
	knativetest.CleanupOnInterrupt(func() { TearDown(t, c, namespace) }, t.Logf)
	defer TearDown(t, c, namespace)

	t.Logf("Creating Git PipelineResource %s", tePipelineGitResourceName)
	if _, err := c.PipelineResourceClient.Create(getGitResourceForOutPutPipeline(tePipelineGitResourceName, namespace)); err != nil {
		t.Fatalf("Failed to create Pipeline Resource `%s`: %s", tePipelineGitResourceName, err)
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

	run := TestEnv.Prepare(t)

	t.Run("Get list of Tasks from namespace  "+namespace, func(t *testing.T) {
		res := icmd.RunCmd(run("task", "list", "-n", namespace))

		expected := ListAllTasksOutput(t, c, map[int]interface{}{
			0: &TaskData{
				Name: TaskName2,
			},
			1: &TaskData{
				Name: TaskName1,
			},
		})

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      expected,
		})
	})

	t.Run("Get list of Pipelines from namespace  "+namespace, func(t *testing.T) {

		res := icmd.RunCmd(run("pipelines", "list", "-n", namespace))

		expected := ListAllPipelinesOutput(t, c, map[int]interface{}{
			0: &PipelinesListData{
				Name:   tePipelineName,
				Status: "---",
			},
		})

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      expected,
		})

	})
	// Bug to fix
	t.Run("Get list of pipelines from other namespace [default] should throw Error", func(t *testing.T) {

		res := icmd.RunCmd(run("pipelines", "list", "-n", "default"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "No Pipelines found\n",
			Err:      icmd.None,
		})
	})

	t.Run("Validate pipelines format for -o (output) flag, as Json Path", func(t *testing.T) {

		res := icmd.RunCmd(run("pipelines", "list", "-n", namespace,
			`-o=jsonpath={range.items[*]}{.metadata.name}{"\n"}{end}`))

		expected := ListResourceNamesForJSONPath(
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
			Out:      expected,
		})

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

		expected := GetPipelineDescribeOutput(t, c, tePipelineName,
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

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      expected,
		})

	})

	vars := make(map[string]interface{})
	var pipelineGeneratedName string

	t.Run("Start Pipeline Run using pipeline start command with SA as 'pipeline' ", func(t *testing.T) {

		res := icmd.RunCmd(run("pipeline", "start", tePipelineName,
			"-r=source-repo="+tePipelineGitResourceName,
			"--showlog",
			"true",
			"-n", namespace))

		time.Sleep(1 * time.Second)

		pipelineGeneratedName = GetPipelineRunListWithName(c, tePipelineName).Items[0].Name
		vars["Element"] = pipelineGeneratedName
		expected := ProcessString(`(Pipelinerun started: {{.Element}}
Waiting for logs to be available...
.*)`, vars)

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		assert.Assert(t, is.Regexp(expected, res.Stdout()))

	})

	time.Sleep(1 * time.Second)

	t.Run("Get list of Taskruns from namespace  "+namespace, func(t *testing.T) {
		time.Sleep(1 * time.Second)

		res := icmd.RunCmd(run("taskrun", "list", "-n", namespace))

		expected := ListAllTaskRunsOutput(t, c, false, map[int]interface{}{
			0: &TaskRunData{
				Name:   "output-pipeline-run-",
				Status: "Succeeded",
			},
			1: &TaskRunData{
				Name:   "output-pipeline-run-",
				Status: "Succeeded",
			},
		})

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      expected,
		})

	})

	t.Run("Validate Pipeline describe command in namespace "+namespace+" after PipelineRun completed successfully", func(t *testing.T) {
		res := icmd.RunCmd(run("pipeline", "describe", tePipelineName, "-n", namespace))

		expected := GetPipelineDescribeOutput(t, c, tePipelineName,
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
					Runs: map[string]string{
						pipelineGeneratedName: "Succeeded",
					},
				},
			})

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      expected,
		})

	})

	t.Run("Validate interactive pipeline logs, with  follow mode (-f) ", func(t *testing.T) {

		RunInteractiveTests(t, namespace, TestEnv.TknBinary(), &Prompt{
			CmdArgs: []string{"pipeline", "logs", "-f", "-n", namespace},
			Procedure: func(c *expect.Console) error {
				if _, err := c.ExpectString("Select pipeline:"); err != nil {
					return err
				}

				if _, err := c.ExpectString("output-pipeline"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				c.Close()
				return nil
			}})
	})
}

func TestPipelinesNegativeE2E(t *testing.T) {
	t.Parallel()
	c, namespace := Setup(t)
	knativetest.CleanupOnInterrupt(func() { TearDown(t, c, namespace) }, t.Logf)
	defer TearDown(t, c, namespace)

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

	run := TestEnv.Prepare(t)

	t.Run("Get list of Pipelines from namespace  "+namespace, func(t *testing.T) {

		res := icmd.RunCmd(run("pipelines", "list", "-n", namespace))

		expected := ListAllPipelinesOutput(t, c, map[int]interface{}{
			0: &PipelinesListData{
				Name:   tePipelineName,
				Status: "---",
			},
		})

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      expected,
		})

	})
	// Bug to fix
	t.Run("Get list of pipelines from other namespace [default] should throw Error", func(t *testing.T) {

		res := icmd.RunCmd(run("pipelines", "list", "-n", "default"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "No Pipelines found\n",
			Err:      icmd.None,
		})
	})

	t.Run("Validate pipelines format for -o (output) flag, as Json Path", func(t *testing.T) {

		res := icmd.RunCmd(run("pipelines", "list", "-n", namespace,
			`-o=jsonpath={range.items[*]}{.metadata.name}{"\n"}{end}`))

		expected := ListResourceNamesForJSONPath(
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
			Out:      expected,
		})
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

		expected := GetPipelineDescribeOutput(t, c, tePipelineName,
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

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      expected,
		})
	})

	vars := make(map[string]interface{})
	var pipelineGeneratedName string

	t.Run("Start Pipeline Run using pipeline start command with SA as 'pipelines' ", func(t *testing.T) {

		res := icmd.RunCmd(run("pipeline", "start", tePipelineName,
			"-r=source-repo="+tePipelineFaultGitResourceName,
			"--showlog",
			"true",
			"-n", namespace))

		pipelineGeneratedName = GetPipelineRunListWithName(c, tePipelineName).Items[0].Name
		vars["Element"] = pipelineGeneratedName
		expected := ProcessString(`(Pipelinerun started: {{.Element}}
Waiting for logs to be available...
.*)`, vars)

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
		})
		assert.Assert(t, is.Regexp(expected, res.Stdout()))
	})

	time.Sleep(1 * time.Second)

	t.Run("Validate Pipeline describe command in namespace "+namespace+" after PipelineRun completed successfully", func(t *testing.T) {
		res := icmd.RunCmd(run("pipeline", "describe", tePipelineName, "-n", namespace))

		expected := GetPipelineDescribeOutput(t, c, tePipelineName,
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
					Runs: map[string]string{
						pipelineGeneratedName: "Failed",
					},
				},
			})

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      expected,
		})

	})

}

func TestDeletePipelinesE2E(t *testing.T) {
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

	run := TestEnv.Prepare(t)

	t.Run("Delete pipeline "+tePipelineName+"-1"+" from namespace "+namespace+" With force delete flag (shorthand)", func(t *testing.T) {
		res := icmd.RunCmd(run("pipeline", "rm", tePipelineName+"-1", "-n", namespace, "-f"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "Pipelines deleted: \"" + tePipelineName + "-1" + "\"\n",
		})
	})

	t.Run("Delete pipeline "+tePipelineName+"-2"+" from namespace "+namespace+" With force delete flag", func(t *testing.T) {
		res := icmd.RunCmd(run("pipeline", "rm", tePipelineName+"-2", "-n", namespace, "--force"))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "Pipelines deleted: \"" + tePipelineName + "-2" + "\"\n",
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
			Out:      "Are you sure you want to delete pipeline \"" + tePipelineName + "-3" + "\" (y/n): Pipelines deleted: \"" + tePipelineName + "-3" + "\"\n",
		})

	})

	t.Run("Check for list of pipelines, After Successfull Deletion of pipeline in namespace "+namespace+" should throw an error", func(t *testing.T) {
		res := icmd.RunCmd(run("pipelines", "list", "-n", namespace))

		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "No Pipelines found\n",
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
		tb.Step("ubuntu", tb.StepName("read-docs-old"), tb.StepCommand("/bin/bash"), tb.StepArgs("-c", "ls -la /workspace/damnworkspace/docs/README.md")),
		tb.Step("ubuntu", tb.StepName("write-new-stuff"), tb.StepCommand("bash"), tb.StepArgs("-c", "ln -s /workspace/damnworkspace /workspace/output/workspace && echo some stuff > /workspace/output/workspace/stuff")),
	}

	return tb.Task(taskname, namespace, tb.TaskSpec(taskSpecOps...))
}

func getReadFileTask(taskname string, namespace string) *v1alpha1.Task {

	taskSpecOps := []tb.TaskSpecOp{
		tb.TaskInputs(tb.InputsResource("workspace", v1alpha1.PipelineResourceTypeGit, tb.ResourceTargetPath("newworkspace"))),
		tb.Step("ubuntu", tb.StepName("read"), tb.StepCommand("/bin/bash"), tb.StepArgs("-c", "cat /workspace/newworkspace/stuff")),
	}

	return tb.Task(taskname, namespace, tb.TaskSpec(taskSpecOps...))
}

func getPipeline(pipelineName string, namespace string, createFiletaskName string, readFileTaskName string) *v1alpha1.Pipeline {

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

	return tb.Pipeline(pipelineName, namespace, tb.PipelineSpec(pipelineSpec...))
}
