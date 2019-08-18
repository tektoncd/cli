package pipelinerun

import (
	"errors"
	"testing"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	"k8s.io/apimachinery/pkg/runtime"
	k8stest "k8s.io/client-go/testing"

	tu "github.com/tektoncd/cli/pkg/test"
)

func Test_cancel_pipelinerun(t *testing.T) {

	prName := "test-pipeline-run-123"

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName, "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipelineName"),
			tb.PipelineRunSpec("pipelineName",
				tb.PipelineRunServiceAccount("test-sa"),
				tb.PipelineRunResourceBinding("git-repo", tb.PipelineResourceBindingRef("some-repo")),
				tb.PipelineRunResourceBinding("build-image", tb.PipelineResourceBindingRef("some-image")),
				tb.PipelineRunParam("pipeline-param-1", "somethingmorefun"),
				tb.PipelineRunParam("rev-param", "revision1"),
			),
		),
	}

	cs, _ := test.SeedTestData(t, test.Data{PipelineRuns: prs})
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Pipelinerun cancelled: " + prName + "\n"
	tu.AssertOutput(t, expected, got)
}

func Test_cancel_pipelinerun_not_found(t *testing.T) {

	prName := "test-pipeline-run-123"

	cs, _ := test.SeedTestData(t, test.Data{})
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: Failed to find pipelinerun: " + prName + "\n"
	tu.AssertOutput(t, expected, got)
}

func Test_cancel_pipelinerun_client_err(t *testing.T) {

	prName := "test-pipeline-run-123"
	errStr := "test generated error"

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun(prName, "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", "pipelineName"),
			tb.PipelineRunSpec("pipelineName",
				tb.PipelineRunServiceAccount("test-sa"),
				tb.PipelineRunResourceBinding("git-repo", tb.PipelineResourceBindingRef("some-repo")),
				tb.PipelineRunResourceBinding("build-image", tb.PipelineResourceBindingRef("some-image")),
				tb.PipelineRunParam("pipeline-param-1", "somethingmorefun"),
				tb.PipelineRunParam("rev-param", "revision1"),
			),
		),
	}

	cs, _ := test.SeedTestData(t, test.Data{PipelineRuns: prs})
	p := &tu.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	cs.Pipeline.PrependReactor("update", "pipelineruns", func(action k8stest.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New(errStr)
	})

	pRun := Command(p)
	got, _ := tu.ExecuteCommand(pRun, "cancel", prName, "-n", "ns")

	expected := "Error: Failed to cancel pipelinerun: " + prName + ", err: " + errStr + "\n"
	tu.AssertOutput(t, expected, got)
}
