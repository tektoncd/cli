//go:build e2e
// +build e2e

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

package pipeline

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/cli/test/builder"
	"github.com/tektoncd/cli/test/cli"
	"github.com/tektoncd/cli/test/framework"
	"github.com/tektoncd/cli/test/helper"
	"github.com/tektoncd/cli/test/wait"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/icmd"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer framework.TearDown(t, c, namespace)

	kubectl := cli.NewKubectl(namespace)
	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)

	t.Logf("Creating pipeline in namespace: %s", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("pipeline.yaml"))

	t.Logf("Creating git pipeline resource in namespace: %s", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("git-resource.yaml"))

	t.Run("Get list of Tasks from namespace  "+namespace, func(t *testing.T) {
		res := tkn.Run("task", "list")
		expected := builder.ListAllTasksOutput(t, c, map[int]interface{}{
			0: &builder.TaskData{
				Name: TaskName2,
			},
			1: &builder.TaskData{
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
		res := tkn.Run("pipelines", "list")
		expected := builder.ListAllPipelinesOutput(t, c, map[int]interface{}{
			0: &builder.PipelinesListData{
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

	t.Run("Get list of pipelines from other namespace [default] should throw Error", func(t *testing.T) {
		res := tkn.RunNoNamespace("pipelines", "list", "-n", "default")
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "No Pipelines found\n",
			Err:      icmd.None,
		})
	})

	t.Run("Validate pipelines format for -o (output) flag, as Json Path", func(t *testing.T) {
		res := tkn.Run("pipelines", "list", `-o=jsonpath={range.items[*]}{.metadata.name}{"\n"}{end}`)
		expected := builder.ListResourceNamesForJSONPath(
			builder.GetPipelineListWithTestData(t, c,
				map[int]interface{}{
					0: &builder.PipelinesListData{
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
		res := tkn.MustSucceed(t, "pipelines", "list", "-o", "json")
		assert.NilError(t, json.Unmarshal([]byte(res.Stdout()), &v1alpha1.PipelineList{}))
	})

	t.Run("Validate Pipeline describe command in namespace "+namespace, func(t *testing.T) {
		res := tkn.Run("pipeline", "describe", tePipelineName)
		expected := builder.GetPipelineDescribeOutput(t, c, tePipelineName,
			map[int]interface{}{
				0: &builder.PipelineDescribeData{
					Name: tePipelineName,
					Resources: map[string]string{
						"source-repo": "git",
					},
					Task: map[int]interface{}{
						0: &builder.TaskRefData{
							TaskName: "first-create-file",
							TaskRef:  TaskName1,
						},
						1: &builder.TaskRefData{
							TaskName: "then-check",
							TaskRef:  TaskName2,
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

	t.Run("Start PipelineRun using pipeline start command with SA as 'pipeline' ", func(t *testing.T) {
		res := tkn.MustSucceed(t, "pipeline", "start", tePipelineName,
			"-r=source-repo="+tePipelineGitResourceName,
			"-p=FILEPATH=docs",
			"-p=FILENAME=README.md",
			"--showlog")

		time.Sleep(1 * time.Second)

		pipelineGeneratedName = builder.GetPipelineRunListWithName(c, tePipelineName, true).Items[0].Name
		vars["Element"] = pipelineGeneratedName
		expected := helper.ProcessString(`(PipelineRun started: {{.Element}}
Waiting for logs to be available...
.*)`, vars)
		assert.Assert(t, is.Regexp(expected, res.Stdout()))
	})

	time.Sleep(1 * time.Second)

	t.Run("Get list of Taskruns from namespace  "+namespace, func(t *testing.T) {
		res := tkn.Run("taskrun", "list")
		expected := builder.ListAllTaskRunsOutput(t, c, false, map[int]interface{}{
			0: &builder.TaskRunData{
				Name:   "output-pipeline-run-",
				Status: "Succeeded",
			},
			1: &builder.TaskRunData{
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
		res := tkn.Run("pipeline", "describe", tePipelineName)
		expected := builder.GetPipelineDescribeOutput(t, c, tePipelineName,
			map[int]interface{}{
				0: &builder.PipelineDescribeData{
					Name: tePipelineName,
					Resources: map[string]string{
						"source-repo": "git",
					},
					Task: map[int]interface{}{
						0: &builder.TaskRefData{
							TaskName: "first-create-file",
							TaskRef:  TaskName1,
							RunAfter: nil,
						},
						1: &builder.TaskRefData{
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
		tkn.RunInteractiveTests(t, &cli.Prompt{
			CmdArgs: []string{"pipeline", "logs", "-f"},
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
			},
		})
	})

	t.Run("Start PipelineRun with tkn pipeline start --last", func(t *testing.T) {
		// Get last PipelineRun for output-pipeline
		pipelineRunLast := builder.GetPipelineRunListWithName(c, tePipelineName, true).Items[0]

		// Start PipelineRun using --last
		tkn.MustSucceed(t, "pipeline", "start", tePipelineName,
			"--last",
			"--showlog")

		// Sleep to make make sure PipelineRun is created/running
		time.Sleep(1 * time.Second)

		// Get name of most recent PipelineRun and wait for it to succeed
		pipelineRunUsingLast := builder.GetPipelineRunListWithName(c, tePipelineName, true).Items[0]
		timeout := 5 * time.Minute
		if err := wait.ForPipelineRunState(c, pipelineRunUsingLast.Name, timeout, wait.PipelineRunSucceed(pipelineRunUsingLast.Name), "PipelineRunSucceeded"); err != nil {
			t.Errorf("Error waiting for PipelineRun to fail: %s", err)
		}

		// Expect that previous PipelineRun spec will match most recent PipelineRun spec
		expected := pipelineRunLast.Spec
		got := pipelineRunUsingLast.Spec
		if d := cmp.Diff(got, expected); d != "" {
			t.Fatalf("-got, +want: %v", d)
		}
	})

	t.Logf("Creating Pipeline pipeline-with-workspace in namespace: %s", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("pipeline-with-workspace.yaml"))

	t.Run("Start PipelineRun with --workspace and volumeClaimTemplate", func(t *testing.T) {
		if tkn.CheckVersion("Pipeline", "v0.10.2") {
			t.Skip("Skip test as pipeline v0.10.2 doesn't support volumeClaimTemplates")
		}

		res := tkn.MustSucceed(t, "pipeline", "start", "pipeline-with-workspace",
			"--workspace=name=ws,volumeClaimTemplateFile="+helper.GetResourcePath("pvc.yaml"),
			"--showlog")

		time.Sleep(1 * time.Second)

		pipelineRunGeneratedName := builder.GetPipelineRunListWithName(c, "pipeline-with-workspace", true).Items[0].Name
		vars["Element"] = pipelineRunGeneratedName
		expected := helper.ProcessString(`(PipelineRun started: {{.Element}}
Waiting for logs to be available...
.*)`, vars)
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
		})
		assert.Assert(t, is.Regexp(expected, res.Stdout()))

		timeout := 5 * time.Minute
		if err := wait.ForPipelineRunState(c, pipelineRunGeneratedName, timeout, wait.PipelineRunSucceed(pipelineRunGeneratedName), "PipelineRunSucceeded"); err != nil {
			t.Errorf("Error waiting for PipelineRun to Succeed: %s", err)
		}
	})

	t.Run("Start PipelineRun with --pod-template", func(t *testing.T) {
		tkn.MustSucceed(t, "pipeline", "start", tePipelineName,
			"-r=source-repo="+tePipelineGitResourceName,
			"--pod-template="+helper.GetResourcePath("/podtemplate/podtemplate.yaml"),
			"--use-param-defaults",
			"-p=FILENAME=README.md",
			"--showlog")

		time.Sleep(1 * time.Second)

		pipelineRunGeneratedName := builder.GetPipelineRunListWithName(c, tePipelineName, true).Items[0].Name
		timeout := 5 * time.Minute
		if err := wait.ForPipelineRunState(c, pipelineRunGeneratedName, timeout, wait.PipelineRunSucceed(pipelineRunGeneratedName), "PipelineRunSucceeded"); err != nil {
			t.Errorf("Error waiting for PipelineRun to Succeed: %s", err)
		}
	})

	t.Run("Cancel finished PipelineRun with tkn pipelinerun cancel", func(t *testing.T) {
		// Get last PipelineRun for pipeline-with-workspace
		pipelineRunLast := builder.GetPipelineRunListWithName(c, tePipelineName, true).Items[0]

		// Cancel PipelineRun
		res := tkn.Run("pipelinerun", "cancel", pipelineRunLast.Name)

		// Expect error from PipelineRun cancel for already completed PipelineRun
		expected := "Error: failed to cancel PipelineRun " + pipelineRunLast.Name + ": PipelineRun has already finished execution\n"
		assert.Assert(t, res.Stderr() == expected)
	})

	t.Logf("Creating Task sleep in namespace: %s", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("cancel/task-cancel.yaml"))
	t.Logf("Creating Pipeline sleep-pipeline in namespace: %s", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("cancel/pipeline-cancel.yaml"))
	t.Run("Cancel PipelineRun with tkn pipelinerun cancel", func(t *testing.T) {
		pipeline := "sleep-pipeline"
		// Start PipelineRun
		tkn.MustSucceed(t, "pipeline", "start", pipeline, "-p", "seconds=30s")

		time.Sleep(1 * time.Second)

		// Get name of most recent PipelineRun
		pipelineRunName := builder.GetPipelineRunListWithName(c, pipeline, true).Items[0].Name

		// Cancel PipelineRun
		res := tkn.MustSucceed(t, "pipelinerun", "cancel", pipelineRunName)

		// Verify successful cancel
		timeout := 2 * time.Minute
		if err := wait.ForPipelineRunState(c, pipelineRunName, timeout, wait.PipelineRunFailed(pipelineRunName), "PipelineRunCancelled"); err != nil {
			t.Errorf("Error waiting for PipelineRun %q to finished: %s", pipelineRunName, err)
		}

		cancelledRun := builder.GetPipelineRun(c, pipelineRunName)
		// Expect successfully cancelled PipelineRun
		expected := "PipelineRun cancelled: " + pipelineRunName + "\n"
		assert.Assert(t, res.Stdout() == expected)
		assert.Assert(t, "PipelineRunCancelled" == cancelledRun.Status.Conditions[0].Reason)
	})
}

func TestPipelinesNegativeE2E(t *testing.T) {
	t.Parallel()
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer framework.TearDown(t, c, namespace)

	kubectl := cli.NewKubectl(namespace)
	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)

	t.Logf("Creating pipeline in namespace: %s", namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("pipeline.yaml"))

	t.Logf("Creating (Fault) Git PipelineResource %s", tePipelineFaultGitResourceName)
	if _, err := c.PipelineResourceClient.Create(context.Background(), getFaultGitResource(tePipelineFaultGitResourceName, namespace), metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create fault Pipeline Resource `%s`: %s", tePipelineFaultGitResourceName, err)
	}

	t.Run("Get list of Pipelines from namespace  "+namespace, func(t *testing.T) {
		res := tkn.Run("pipelines", "list")
		expected := builder.ListAllPipelinesOutput(t, c, map[int]interface{}{
			0: &builder.PipelinesListData{
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

	t.Run("Get list of pipelines from other namespace [default] should throw Error", func(t *testing.T) {
		res := tkn.RunNoNamespace("pipelines", "list", "-n", "default")
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "No Pipelines found\n",
			Err:      icmd.None,
		})
	})

	t.Run("Validate pipelines format for -o (output) flag, as Json Path", func(t *testing.T) {
		res := tkn.Run("pipelines", "list", `-o=jsonpath={range.items[*]}{.metadata.name}{"\n"}{end}`)
		expected := builder.ListResourceNamesForJSONPath(
			builder.GetPipelineListWithTestData(t, c,
				map[int]interface{}{
					0: &builder.PipelinesListData{
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
		res := tkn.MustSucceed(t, "pipelines", "list", "-o", "json")
		assert.NilError(t, json.Unmarshal([]byte(res.Stdout()), &v1alpha1.PipelineList{}))
	})

	t.Run("Validate Pipeline describe command in namespace "+namespace, func(t *testing.T) {
		res := tkn.Run("pipeline", "describe", tePipelineName)
		expected := builder.GetPipelineDescribeOutput(t, c, tePipelineName,
			map[int]interface{}{
				0: &builder.PipelineDescribeData{
					Name: tePipelineName,
					Resources: map[string]string{
						"source-repo": "git",
					},
					Task: map[int]interface{}{
						0: &builder.TaskRefData{
							TaskName: "first-create-file",
							TaskRef:  TaskName1,
							RunAfter: nil,
						},
						1: &builder.TaskRefData{
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
		res := tkn.MustSucceed(t, "pipeline", "start", tePipelineName,
			"-r=source-repo="+tePipelineFaultGitResourceName,
			"-p=FILEPATH=docs",
			"-p=FILENAME=README.md",
			"--showlog",
			"true")

		pipelineGeneratedName = builder.GetPipelineRunListWithName(c, tePipelineName, true).Items[0].Name
		vars["Element"] = pipelineGeneratedName
		expected := helper.ProcessString(`(PipelineRun started: {{.Element}}
Waiting for logs to be available...
.*)`, vars)
		assert.Assert(t, is.Regexp(expected, res.Stdout()))
	})

	time.Sleep(1 * time.Second)

	t.Run("Validate Pipeline describe command in namespace "+namespace+" after PipelineRun completed successfully", func(t *testing.T) {
		res := tkn.Run("pipeline", "describe", tePipelineName)
		expected := builder.GetPipelineDescribeOutput(t, c, tePipelineName,
			map[int]interface{}{
				0: &builder.PipelineDescribeData{
					Name: tePipelineName,
					Resources: map[string]string{
						"source-repo": "git",
					},
					Task: map[int]interface{}{
						0: &builder.TaskRefData{
							TaskName: "first-create-file",
							TaskRef:  TaskName1,
							RunAfter: nil,
						},
						1: &builder.TaskRefData{
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
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer framework.TearDown(t, c, namespace)

	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)

	t.Logf("Creating Git PipelineResource %s", tePipelineGitResourceName)
	if _, err := c.PipelineResourceClient.Create(context.Background(), getGitResource(tePipelineGitResourceName, namespace), metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create Pipeline Resource `%s`: %s", tePipelineGitResourceName, err)
	}

	t.Logf("Creating (Fault) Git PipelineResource %s", tePipelineFaultGitResourceName)
	if _, err := c.PipelineResourceClient.Create(context.Background(), getFaultGitResource(tePipelineFaultGitResourceName, namespace), metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create fault Pipeline Resource `%s`: %s", tePipelineFaultGitResourceName, err)
	}

	t.Logf("Creating Task  %s", TaskName1)
	if _, err := c.TaskClient.Create(context.Background(), getCreateFileTask(TaskName1, namespace), metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create Task Resource `%s`: %s", TaskName1, err)
	}

	t.Logf("Creating Task  %s", TaskName2)
	if _, err := c.TaskClient.Create(context.Background(), getReadFileTask(TaskName2, namespace), metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create Task Resource `%s`: %s", TaskName2, err)
	}
	for i := 1; i <= 3; i++ {
		t.Logf("Create Pipeline %s", tePipelineName+"-"+strconv.Itoa(i))
		if _, err := c.PipelineClient.Create(context.Background(), getPipeline(tePipelineName+"-"+strconv.Itoa(i), namespace, TaskName1, TaskName2), metav1.CreateOptions{}); err != nil {
			t.Fatalf("Failed to create pipeline `%s`: %s", tePipelineName+"-"+strconv.Itoa(i), err)
		}
	}

	time.Sleep(1 * time.Second)

	t.Run("Delete pipeline "+tePipelineName+"-1"+" from namespace "+namespace+" With force delete flag (shorthand)", func(t *testing.T) {
		res := tkn.Run("pipeline", "rm", tePipelineName+"-1", "-f")
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "Pipelines deleted: \"" + tePipelineName + "-1" + "\"\n",
		})
	})

	t.Run("Delete pipeline "+tePipelineName+"-2"+" from namespace "+namespace+" With force delete flag", func(t *testing.T) {
		res := tkn.Run("pipeline", "rm", tePipelineName+"-2", "--force")
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "Pipelines deleted: \"" + tePipelineName + "-2" + "\"\n",
		})
	})

	t.Run("Delete pipeline "+tePipelineName+"-3"+" from namespace "+namespace+" without force flag, reply no", func(t *testing.T) {
		res := tkn.RunWithOption(icmd.WithStdin(strings.NewReader("n")),
			"pipeline", "rm", tePipelineName+"-3")
		res.Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      "Error: canceled deleting Pipeline(s) \"" + tePipelineName + "-3" + "\"\n",
		})
	})

	t.Run("Delete pipeline "+tePipelineName+"-3"+" from namespace "+namespace+" without force flag, reply yes", func(t *testing.T) {
		res := tkn.RunWithOption(icmd.WithStdin(strings.NewReader("y")),
			"pipeline", "rm", tePipelineName+"-3")
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Err:      icmd.None,
			Out:      "Are you sure you want to delete Pipeline(s) \"" + tePipelineName + "-3" + "\" (y/n): Pipelines deleted: \"" + tePipelineName + "-3" + "\"\n",
		})
	})

	t.Run("Check for list of pipelines, After Successful Deletion of pipeline in namespace "+namespace+" should throw an error", func(t *testing.T) {
		res := tkn.Run("pipelines", "list")
		res.Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "No Pipelines found\n",
			Err:      icmd.None,
		})
	})
}

func getGitResource(rname string, namespace string) *v1alpha1.PipelineResource {
	return &v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rname,
			Namespace: namespace,
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: v1alpha1.PipelineResourceTypeGit,
			Params: []v1alpha1.ResourceParam{
				{
					Name:  "url",
					Value: "https://github.com/GoogleContainerTools/skaffold",
				},
				{
					Name:  "revision",
					Value: "main",
				},
			},
		},
	}
}

func getFaultGitResource(rname string, namespace string) *v1alpha1.PipelineResource {
	return &v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rname,
			Namespace: namespace,
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: v1alpha1.PipelineResourceTypeGit,
			Params: []v1alpha1.ResourceParam{
				{
					Name:  "url",
					Value: "https://github.com/GoogleContainerTools/skaffold-1",
				},
				{
					Name:  "revision",
					Value: "main",
				},
			},
		},
	}
}

func getCreateFileTask(taskname string, namespace string) *v1alpha1.Task {
	return &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      taskname,
			Namespace: namespace,
		},
		Spec: v1alpha1.TaskSpec{
			TaskSpec: v1beta1.TaskSpec{
				Steps: []v1alpha1.Step{
					{
						Container: corev1.Container{
							Name:    "read-docs-old",
							Image:   "ubuntu",
							Command: []string{"/bin/bash"},
							Args:    []string{"-c", "ls -la /workspace/damnworkspace/docs/README.md"},
						},
					},
					{
						Container: corev1.Container{
							Name:    "write-new-stuff",
							Image:   "ubuntu",
							Command: []string{"bash"},
							Args:    []string{"-c", "ln -s /workspace/damnworkspace /workspace/output/workspace && echo some stuff > /workspace/output/workspace/stuff"},
						},
					},
				},
			},

			Inputs: &v1alpha1.Inputs{
				Resources: []v1alpha1.TaskResource{
					{
						ResourceDeclaration: v1alpha1.ResourceDeclaration{
							Name:       "workspace",
							Type:       v1alpha1.PipelineResourceTypeGit,
							TargetPath: "damnworkspace",
						},
					},
				},
			},
			Outputs: &v1alpha1.Outputs{
				Resources: []v1beta1.TaskResource{
					{
						ResourceDeclaration: v1alpha1.ResourceDeclaration{
							Name: "workspace",
							Type: v1alpha1.PipelineResourceTypeImage,
						},
					},
				},
			},
		},
	}
}

func getReadFileTask(taskname string, namespace string) *v1alpha1.Task {
	return &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      taskname,
			Namespace: namespace,
		},
		Spec: v1alpha1.TaskSpec{
			TaskSpec: v1beta1.TaskSpec{
				Steps: []v1alpha1.Step{
					{
						Container: corev1.Container{
							Name:    "read",
							Image:   "ubuntu",
							Command: []string{"/bin/bash"},
							Args:    []string{"-c", "cat /workspace/newworkspace/stuff"},
						},
					},
				},
			},

			Inputs: &v1alpha1.Inputs{
				Resources: []v1alpha1.TaskResource{
					{
						ResourceDeclaration: v1alpha1.ResourceDeclaration{
							Name:       "workspace",
							Type:       v1alpha1.PipelineResourceTypeGit,
							TargetPath: "newworkspace",
						},
					},
				},
			},
		},
	}
}

func getPipeline(pipelineName string, namespace string, createFiletaskName string, readFileTaskName string) *v1alpha1.Pipeline {
	return &v1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineName,
			Namespace: namespace,
		},
		Spec: v1alpha1.PipelineSpec{
			Resources: []v1beta1.PipelineDeclaredResource{
				{
					Name: "source-repo",
					Type: v1alpha1.PipelineResourceTypeGit,
				},
			},
			Tasks: []v1alpha1.PipelineTask{
				{
					Name: "first-create-file",
					TaskRef: &v1alpha1.TaskRef{
						Name: createFiletaskName,
					},
					Resources: &v1alpha1.PipelineTaskResources{
						Inputs: []v1alpha1.PipelineTaskInputResource{
							{
								Name:     "workspace",
								Resource: "source-repo",
							},
						},
						Outputs: []v1alpha1.PipelineTaskOutputResource{
							{
								Name:     "workspace",
								Resource: "source-repo",
							},
						},
					},
				},
				{
					Name: "then-check",
					TaskRef: &v1alpha1.TaskRef{
						Name: readFileTaskName,
					},
					Resources: &v1alpha1.PipelineTaskResources{
						Inputs: []v1alpha1.PipelineTaskInputResource{
							{
								Name:     "workspace",
								Resource: "source-repo",
								From:     []string{"first-create-file"},
							},
						},
					},
				},
			},
		},
	}
}
