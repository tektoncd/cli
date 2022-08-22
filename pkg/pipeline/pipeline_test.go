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

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const versionB1 = "v1beta1"

func TestPipelinesList_v1beta1(t *testing.T) {
	clock := clockwork.NewFakeClock()

	pdata := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: "ns",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Pipelines: pdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"pipeline"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(pdata[0], versionB1),
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
	cs2, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Pipelines: pdata2,
	})
	cs2.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"pipeline", "pipelinerun"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1P(pdata2[0], versionB1),
		cb.UnstructuredV1beta1P(pdata2[1], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}
	p3 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}
	p3.SetNamespace("unknown")

	testParams := []struct {
		name   string
		params *test.Params
		want   []string
	}{
		{
			name:   "Single Pipeline",
			params: p,
			want:   []string{"pipeline"},
		},
		{
			name:   "Multi Pipelines",
			params: p2,
			want:   []string{"pipeline", "pipeline2"},
		},
		{
			name:   "Unknown namespace",
			params: p3,
			want:   []string{},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllPipelineNames(tp.params)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestPipelineGet_v1beta1(t *testing.T) {
	clock := clockwork.NewFakeClock()

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
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Pipelines: pdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(pdata[0], versionB1),
		cb.UnstructuredV1beta1P(pdata[1], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Get(c, "pipeline", metav1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "pipeline", got.Name)
}
