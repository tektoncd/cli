package e2e

import (
	"log"
	"os"
	"testing"

	tb "github.com/tektoncd/pipeline/test/builder"
)

//Create on configured cluster Pipeline Run in a new namespace everytime ,and wait for its completion state (Failure or Passed)
func CreatePipelineRun(t *testing.T, taskName string, PipelineName string, PipelineRunName string) (string, *Clients) {
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
	pipelineRun := tb.PipelineRun(PipelineRunName, namespace, tb.PipelineRunSpec(pipeline.Name))
	if _, err := c.PipelineClient.Create(pipeline); err != nil {
		log.Fatalf("Failed to create Pipeline `%s`: %s", pipeline.Name, err)
	}
	if _, err := c.PipelineRunClient.Create(pipelineRun); err != nil {
		log.Fatalf("Failed to create PipelineRun `%s`: %s", pipelineRun.Name, err)
	}
	WaitForPipelineRunToComplete(c, pipelineRun.Name, namespace)
	return namespace, c
}

func CreatePipelineRunViaYaml(t *testing.T, PipelineRunName string) (string, *Clients) {
	t.Helper()

	c, namespace := Setup(t)

	actual := CmdShouldPass("kubectl apply -f " + os.Getenv("GOPATH") + "/src/github.com/tektoncd/cli/test/resources/output-pipelinerun.yaml -n " + namespace)
	log.Printf("Running cmd \n %s  -> \n %s", "kubectl apply -f output-pipelinerun.yaml -n "+namespace, actual)

	log.Printf("Creating PipelineRun in namespace %s", namespace)

	//logic to wait for pipelineRun to its excecution
	WaitForPipelineRunToComplete(c, PipelineRunName, namespace)

	return namespace, c

}
