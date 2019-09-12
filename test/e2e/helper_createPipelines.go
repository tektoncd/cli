package e2e

import (
	"log"
	"testing"
	"time"

	tb "github.com/tektoncd/pipeline/test/builder"
)

//Create Pipeline on configured cluster in a new namespace on each run
func CreatePipeline(t *testing.T, taskName string, PipelineName string) (string, *Clients) {
	t.Helper()
	c, namespace := Setup(t)

	log.Printf("Creating Task in namespace %s", namespace)
	task := tb.Task(taskName, namespace, tb.TaskSpec(
		tb.Step("foo", "busybox", tb.StepCommand("/bin/sh"), tb.StepArgs("-c", "echo Hello World!!")),
		tb.Step("foo-1", "busybox", tb.StepCommand("/bin/sh"), tb.StepArgs("-c", "echo Welcome to Tekton")),
	))
	if _, err := c.TaskClient.Create(task); err != nil {
		log.Fatalf("Failed to create Task `%s`: %s", task.Name, err)
	}

	pipeline := tb.Pipeline(PipelineName, namespace,
		tb.PipelineSpec(tb.PipelineTask("foo", taskName)),
	)
	if _, err := c.PipelineClient.Create(pipeline); err != nil {
		log.Fatalf("Failed to create Pipeline `%s`: %s", pipeline.Name, err)
	}
	time.Sleep(1 * time.Second)

	return namespace, c
}
