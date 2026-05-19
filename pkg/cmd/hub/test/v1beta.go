package test

import (
	"context"
	"testing"

	fakepipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	informersv1beta1 "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1beta1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	fakepipelineinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipeline/fake"
	faketaskinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/task/fake"
	"github.com/tektoncd/pipeline/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
)

type Clients struct {
	Pipeline *fakepipelineclientset.Clientset
	Kube     *fakekubeclientset.Clientset
}

// Informers holds references to informers which are useful for reconciler tests.
type Informers struct {
	Pipeline informersv1beta1.PipelineInformer
	Task     informersv1beta1.TaskInformer
}

// seedTestData returns Clients and Informers populated with the
// given Data.
// nolint: revive
func seedTestData(t *testing.T, ctx context.Context, d Data) (Clients, Informers) {
	c := Clients{

		Kube:     fakekubeclient.Get(ctx),
		Pipeline: fakepipelineclient.Get(ctx),
	}

	// Every time a resource is modified, change the metadata.resourceVersion.
	test.PrependResourceVersionReactor(&c.Pipeline.Fake)

	i := Informers{
		Pipeline: fakepipelineinformer.Get(ctx),
		Task:     faketaskinformer.Get(ctx),
	}

	// Attach reactors that add resource mutations to the appropriate
	// informer index, and simulate optimistic concurrency failures when
	// the resource version is mismatched.
	c.Pipeline.PrependReactor("*", "pipelines", test.AddToInformer(t, i.Pipeline.Informer().GetIndexer()))
	for _, p := range d.Pipelines {
		p := p.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := c.Pipeline.TektonV1beta1().Pipelines(p.Namespace).Create(ctx, p, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	c.Pipeline.PrependReactor("*", "tasks", test.AddToInformer(t, i.Task.Informer().GetIndexer()))
	for _, ta := range d.Tasks {
		ta := ta.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := c.Pipeline.TektonV1beta1().Tasks(ta.Namespace).Create(ctx, ta, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	c.Pipeline.ClearActions()
	c.Kube.ClearActions()
	return c, i
}
