package e2e

import (
	"log"
	"testing"
	"time"

	tb "github.com/tektoncd/pipeline/test/builder"
)

//Create Task on configured cluster in a new namespace on each run
func CreateTask(t *testing.T, taskname string) (string, *Clients) {
	c, namespace := Setup(t)

	task := tb.Task(taskname, namespace,
		tb.TaskSpec(tb.Step("amazing-busybox", "busybox", tb.StepCommand("/bin/sh"), tb.StepArgs("-c", "echo Hello")),
			tb.Step("amazing-busybox-1", "busybox", tb.StepCommand("/bin/sh"), tb.StepArgs("-c", "echo Welcome To Tekton!!")),
		))
	if _, err := c.TaskClient.Create(task); err != nil {
		log.Fatalf("Failed to create Task `%s`: %s", task.Name, err)
	}

	time.Sleep(1 * time.Second)

	return namespace, c

}
