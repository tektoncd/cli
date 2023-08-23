// Copyright © 2019 The Tekton Authors.
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
	"testing"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPipelinesList_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	pdata := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: "ns",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		Pipelines: pdata,
	})
	version := "v1beta1"
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(pdata[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	pdata2 := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline2",
				Namespace: "ns",
			},
		},
	}
	cs2, _ := test.SeedV1beta1TestData(t, test.Data{
		Pipelines: pdata2,
	})
	cs2.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1P(pdata2[0], version),
		cb.UnstructuredV1beta1P(pdata2[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}
	p3 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}
	p3.SetNamespace("unknown")

	c1, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	c2, err := p2.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	c3, err := p3.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	testParams := []struct {
		name      string
		namespace string
		client    *cli.Clients
		want      []string
	}{
		{
			name:      "Single Pipeline",
			namespace: p.Namespace(),
			client:    c1,
			want:      []string{"pipeline"},
		},
		{
			name:      "Multi Pipelines",
			namespace: p2.Namespace(),
			client:    c2,
			want:      []string{"pipeline", "pipeline2"},
		},
		{
			name:      "Unknown namespace",
			namespace: p3.Namespace(),
			client:    c3,
			want:      []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllPipelineNames(pipelineGroupResource, tp.client, tp.namespace)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestPipelinesList(t *testing.T) {
	clock := test.FakeClock()

	pdata := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: "ns",
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: pdata,
	})
	version := "v1"
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(pdata[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	pdata2 := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline2",
				Namespace: "ns",
			},
		},
	}
	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: pdata2,
	})
	cs2.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredP(pdata2[0], version),
		cb.UnstructuredP(pdata2[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}
	p3 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}
	p3.SetNamespace("unknown")

	c1, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	c2, err := p2.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	c3, err := p3.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	testParams := []struct {
		name      string
		namespace string
		client    *cli.Clients
		want      []string
	}{
		{
			name:      "Single Pipeline",
			namespace: p.Namespace(),
			client:    c1,
			want:      []string{"pipeline"},
		},
		{
			name:      "Multi Pipelines",
			namespace: p2.Namespace(),
			client:    c2,
			want:      []string{"pipeline", "pipeline2"},
		},
		{
			name:      "Unknown namespace",
			namespace: p3.Namespace(),
			client:    c3,
			want:      []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllPipelineNames(pipelineGroupResource, tp.client, tp.namespace)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestPipelineGet_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	pdata := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline2",
				Namespace: "ns",
			},
		},
	}
	version := "v1beta1"
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		Pipelines: pdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(pdata[0], version),
		cb.UnstructuredV1beta1P(pdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	var pipeline *v1beta1.Pipeline
	err = actions.GetV1(pipelineGroupResource, c, "pipeline", "ns", metav1.GetOptions{}, &pipeline)
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "pipeline", pipeline.Name)
}

func TestPipelineGet(t *testing.T) {
	clock := test.FakeClock()

	pdata := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline2",
				Namespace: "ns",
			},
		},
	}
	version := "v1"
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: pdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(pdata[0], version),
		cb.UnstructuredP(pdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	var pipeline *v1.Pipeline
	err = actions.GetV1(pipelineGroupResource, c, "pipeline", "ns", metav1.GetOptions{}, &pipeline)
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "pipeline", pipeline.Name)
}
