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

package clustertask

import (
	"fmt"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"gotest.tools/v3/golden"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterTaskList_Empty(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{})
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"clustertask"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Dynamic: dynamic}
	clustertask := Command(p)

	output, err := test.ExecuteCommand(clustertask, "list")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, emptyMsg+"\n", output)
}

func TestClusterTaskListOnlyClusterTasksv1alpha1(t *testing.T) {
	clock := clockwork.NewFakeClock()

	clustertasks := []*v1alpha1.ClusterTask{
		tb.ClusterTask("guavas", cb.ClusterTaskCreationTime(clock.Now().Add(-1*time.Minute))),
		tb.ClusterTask("avocados", cb.ClusterTaskCreationTime(clock.Now().Add(-20*time.Second))),
		tb.ClusterTask("pineapple", tb.ClusterTaskSpec(tb.TaskDescription("a test clustertask")), cb.ClusterTaskCreationTime(clock.Now().Add(-512*time.Hour))),
		tb.ClusterTask("apple", tb.ClusterTaskSpec(tb.TaskDescription("a clustertask to test description")), cb.ClusterTaskCreationTime(clock.Now().Add(-513*time.Hour))),
		tb.ClusterTask("mango", tb.ClusterTaskSpec(tb.TaskDescription("")), cb.ClusterTaskCreationTime(clock.Now().Add(-514*time.Hour))),
	}

	version := "v1alpha1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredCT(clustertasks[0], version),
		cb.UnstructuredCT(clustertasks[1], version),
		cb.UnstructuredCT(clustertasks[2], version),
		cb.UnstructuredCT(clustertasks[3], version),
		cb.UnstructuredCT(clustertasks[4], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{ClusterTasks: clustertasks})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Clock: clock, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask"})

	clustertask := Command(p)
	output, err := test.ExecuteCommand(clustertask, "list")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}

func TestClusterTaskListOnlyClusterTasksv1beta1(t *testing.T) {
	clock := clockwork.NewFakeClock()

	clustertasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "guavas",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "avocados",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pineapple",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test clustertask",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "apple",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a clustertask to test description",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "mango",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-514 * time.Hour)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], version),
		cb.UnstructuredV1beta1CT(clustertasks[1], version),
		cb.UnstructuredV1beta1CT(clustertasks[2], version),
		cb.UnstructuredV1beta1CT(clustertasks[3], version),
		cb.UnstructuredV1beta1CT(clustertasks[4], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{ClusterTasks: clustertasks})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Clock: clock, Dynamic: dynamic}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask"})

	clustertask := Command(p)
	output, err := test.ExecuteCommand(clustertask, "list")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
}
