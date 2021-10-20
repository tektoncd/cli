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

package clustertask

import (
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterTask_GetAllTaskNames(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()
	ctdata := []*v1alpha1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask",
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		ClusterTasks: ctdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredCT(ctdata[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	ctdata2 := []*v1alpha1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask",
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask2",
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}
	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
		ClusterTasks: ctdata2,
	})
	cs2.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredCT(ctdata2[0], version),
		cb.UnstructuredCT(ctdata2[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}

	testParams := []struct {
		name   string
		params *test.Params
		want   []string
	}{
		{
			name:   "Single ClusterTask",
			params: p,
			want:   []string{"clustertask"},
		},
		{
			name:   "Multi ClusterTasks",
			params: p2,
			want:   []string{"clustertask", "clustertask2"},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := GetAllClusterTaskNames(tp.params)
			if err != nil {
				t.Errorf("unexpected Error")
			}
			test.AssertOutput(t, tp.want, got)
		})
	}
}

func TestClusterTask_List(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()
	ctdata := []*v1alpha1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask",
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		ClusterTasks: ctdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredCT(ctdata[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	ctdata2 := []*v1alpha1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask",
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask2",
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}
	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
		ClusterTasks: ctdata2,
	})
	cs2.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredCT(ctdata2[0], version),
		cb.UnstructuredCT(ctdata2[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}

	c1, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	c2, err := p2.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	testParams := []struct {
		name   string
		client *cli.Clients
		want   []string
	}{
		{
			name:   "Single clusterTask",
			client: c1,
			want:   []string{"clustertask"},
		},
		{
			name:   "Multi clusterTasks",
			client: c2,
			want:   []string{"clustertask", "clustertask2"},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := List(tp.client, metav1.ListOptions{})
			if err != nil {
				t.Errorf("unexpected Error")
			}

			ctnames := []string{}
			for _, ct := range got.Items {
				ctnames = append(ctnames, ct.Name)
			}
			test.AssertOutput(t, tp.want, ctnames)
		})
	}
}

func TestClusterTaskV1beta1_List(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	ctdata := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		ClusterTasks: ctdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1CT(ctdata[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	ctdata2 := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask2",
			},
		},
	}
	cs2, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		ClusterTasks: ctdata2,
	})
	cs2.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1CT(ctdata2[0], version),
		cb.UnstructuredV1beta1CT(ctdata2[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	p2 := &test.Params{Tekton: cs2.Pipeline, Clock: clock, Kube: cs2.Kube, Dynamic: dc2}

	c1, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	c2, err := p2.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	testParams := []struct {
		name   string
		client *cli.Clients
		want   []string
	}{
		{
			name:   "Single clusterTask",
			client: c1,
			want:   []string{"clustertask"},
		},
		{
			name:   "Multi clusterTasks",
			client: c2,
			want:   []string{"clustertask", "clustertask2"},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := List(tp.client, metav1.ListOptions{})
			if err != nil {
				t.Errorf("unexpected Error")
			}

			ctnames := []string{}
			for _, ct := range got.Items {
				ctnames = append(ctnames, ct.Name)
			}
			test.AssertOutput(t, tp.want, ctnames)
		})
	}
}

func TestClusterTask_Get(t *testing.T) {
	version := "v1alpha1"
	clock := clockwork.NewFakeClock()
	ctdata := []*v1alpha1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask2",
				// created  5 minutes back
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		ClusterTasks: ctdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredCT(ctdata[0], version),
		cb.UnstructuredCT(ctdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Get(c, "clustertask", metav1.GetOptions{})
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "clustertask", got.Name)
}

func TestClusterTaskV1beta1_Get(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	ctdata := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask2",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		ClusterTasks: ctdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1CT(ctdata[0], version),
		cb.UnstructuredV1beta1CT(ctdata[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Get(c, "clustertask", metav1.GetOptions{})
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "clustertask", got.Name)
}

func TestClusterTask_Create(t *testing.T) {
	version := "v1beta1"
	clock := clockwork.NewFakeClock()
	ctdata := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	c, err := p.Clients()
	if err != nil {
		t.Errorf("unable to create client: %v", err)
	}

	got, err := Create(c, ctdata[0], metav1.CreateOptions{})
	if err != nil {
		t.Errorf("unexpected Error")
	}
	test.AssertOutput(t, "clustertask", got.Name)
}
