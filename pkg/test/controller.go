// Copyright © 2022 The Tekton Authors.
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

package test

import (
	"context"
	"testing"

	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	fakepipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	informersv1beta1 "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1beta1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	fakepipelineinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipeline/fake"
	fakepipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun/fake"
	faketaskinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/task/fake"
	faketaskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun/fake"
	"github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	fakefilteredpodinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/pod/filtered/fake"
)

type Data struct {
	PipelineRuns []*v1beta1.PipelineRun
	Pipelines    []*v1beta1.Pipeline
	TaskRuns     []*v1beta1.TaskRun
	Tasks        []*v1beta1.Task
	Namespaces   []*corev1.Namespace
	Pods         []*corev1.Pod
}

// Clients holds references to clients which are useful for reconciler tests.
type Clients struct {
	Pipeline *fakepipelineclientset.Clientset
	Kube     *fakekubeclientset.Clientset
}

// Informers holds references to informers which are useful for reconciler tests.
type Informers struct {
	PipelineRun informersv1beta1.PipelineRunInformer
	Pipeline    informersv1beta1.PipelineInformer
	TaskRun     informersv1beta1.TaskRunInformer
	Task        informersv1beta1.TaskInformer
	Pod         coreinformers.PodInformer
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
		PipelineRun: fakepipelineruninformer.Get(ctx),
		Pipeline:    fakepipelineinformer.Get(ctx),
		TaskRun:     faketaskruninformer.Get(ctx),
		Task:        faketaskinformer.Get(ctx),
		Pod:         fakefilteredpodinformer.Get(ctx, v1beta1.ManagedByLabelKey),
	}

	// Attach reactors that add resource mutations to the appropriate
	// informer index, and simulate optimistic concurrency failures when
	// the resource version is mismatched.
	c.Pipeline.PrependReactor("*", "pipelineruns", test.AddToInformer(t, i.PipelineRun.Informer().GetIndexer()))
	for _, pr := range d.PipelineRuns {
		pr := pr.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := c.Pipeline.TektonV1beta1().PipelineRuns(pr.Namespace).Create(ctx, pr, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	c.Pipeline.PrependReactor("*", "pipelines", test.AddToInformer(t, i.Pipeline.Informer().GetIndexer()))
	for _, p := range d.Pipelines {
		p := p.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := c.Pipeline.TektonV1beta1().Pipelines(p.Namespace).Create(ctx, p, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	c.Pipeline.PrependReactor("*", "taskruns", test.AddToInformer(t, i.TaskRun.Informer().GetIndexer()))
	for _, tr := range d.TaskRuns {
		tr := tr.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := c.Pipeline.TektonV1beta1().TaskRuns(tr.Namespace).Create(ctx, tr, metav1.CreateOptions{}); err != nil {
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
	c.Kube.PrependReactor("*", "pods", test.AddToInformer(t, i.Pod.Informer().GetIndexer()))
	for _, p := range d.Pods {
		p := p.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := c.Kube.CoreV1().Pods(p.Namespace).Create(ctx, p, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, n := range d.Namespaces {
		n := n.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := c.Kube.CoreV1().Namespaces().Create(ctx, n, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	c.Pipeline.ClearActions()
	c.Kube.ClearActions()
	return c, i
}
