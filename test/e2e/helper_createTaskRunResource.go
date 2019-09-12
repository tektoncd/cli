package e2e

import (
	"log"
	"testing"

	tb "github.com/tektoncd/pipeline/test/builder"
)

//Create TaskRun on configured cluster in a new namespace on each run
func CreateTaskRun(t *testing.T, taskName string, taskRunName string) (string, *Clients) {
	c, namespace := Setup(t)

	log.Printf("Creating Task and TaskRun in namespace %s", namespace)
	task := tb.Task(taskName, namespace,
		tb.TaskSpec(tb.Step("amazing-busybox", "busybox", tb.StepCommand("/bin/sh"), tb.StepArgs("-c", "echo Hello")),
			tb.Step("amazing-busybox-1", "busybox", tb.StepCommand("/bin/sh"), tb.StepArgs("-c", "echo Welcome To Tekton!!")),
		))
	if _, err := c.TaskClient.Create(task); err != nil {
		log.Fatalf("Failed to create Task `%s`: %s", task.Name, err)
	}
	taskRun := tb.TaskRun(taskRunName, namespace, tb.TaskRunSpec(tb.TaskRunTaskRef(task.Name)))

	if _, err := c.TaskRunClient.Create(taskRun); err != nil {
		log.Fatalf("Failed to create TaskRun `%s`: %s", taskRun.Name, err)
	}

	// Logic to wait for task Run to wait for its Execution
	WaitForTaskRunToComplete(c, taskRun.Name, namespace)

	return namespace, c
}
