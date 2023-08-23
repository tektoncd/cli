// Copyright © 2023 The Tekton Authors.
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
	"fmt"
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPipelineExport_v1beta1(t *testing.T) {
	clock := test.FakeClock()
	pipelines := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(2 * time.Minute)},
				ResourceVersion:   "123456",
				Annotations: map[string]string{
					"kubectl.kubernetes.io/last-applied-configuration": "test",
					"pipeline.dev": "cli",
				},
				GenerateName: "generate-name",
				Generation:   5,
				UID:          "f54b8b67-ce52-4509-8a4a-f245b093b62e",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "task-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "task-1",
						},
					},
					{
						Name: "task-2",
						TaskRef: &v1beta1.TaskRef{
							Name: "task-2",
						},
					},
				},
				Results: []v1beta1.PipelineResult{
					{
						Name:        "result-1",
						Description: "This is a description for result 1",
						Type:        v1beta1.ResultsTypeString,
						Value: v1beta1.ParamValue{
							Type: v1beta1.ParamTypeString,
						},
					},
					{
						Name:        "result-2",
						Description: "This is a description for result 2",
						Type:        v1beta1.ResultsTypeString,
						Value: v1beta1.ParamValue{
							Type: v1beta1.ParamTypeString,
						},
					},
					{
						Name: "result-3",
						Type: v1beta1.ResultsTypeString,
						Value: v1beta1.ParamValue{
							Type: v1beta1.ParamTypeString,
						},
					},
				},
				Workspaces: []v1beta1.PipelineWorkspaceDeclaration{
					{
						Name:        "test",
						Description: "test workspace",
					},
				},
			},
		},
	}
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1P(pipelines[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: namespaces, Pipelines: pipelines})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}
	pipeline := Command(p)

	got, err := test.ExecuteCommand(pipeline, "export", "-n", "ns", "pipeline")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineExport(t *testing.T) {
	clock := test.FakeClock()
	pipelines := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(2 * time.Minute)},
				ResourceVersion:   "123456",
				Annotations: map[string]string{
					"kubectl.kubernetes.io/last-applied-configuration": "test",
					"pipeline.dev": "cli",
				},
				GenerateName: "generate-name",
				Generation:   5,
				UID:          "f54b8b67-ce52-4509-8a4a-f245b093b62e",
			},
			Spec: v1.PipelineSpec{
				Description: "",
				Tasks: []v1.PipelineTask{
					{
						Name: "task",
						TaskRef: &v1.TaskRef{
							Name: "taskref",
						},
						RunAfter: []string{
							"one",
							"two",
						},
						Params: []v1.Param{{
							Name: "param", Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "value"},
						}},
						Timeout: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
					},
				},
				Results: []v1.PipelineResult{
					{
						Name:        "result-1",
						Description: "This is a description for result 1",
						Type:        v1.ResultsTypeString,
						Value: v1.ParamValue{
							Type: v1.ParamTypeString,
						},
					},
					{
						Name:        "result-2",
						Description: "This is a description for result 2",
						Type:        v1.ResultsTypeString,
						Value: v1.ParamValue{
							Type: v1.ParamTypeString,
						},
					},
					{
						Name: "result-3",
						Type: v1.ResultsTypeString,
						Value: v1.ParamValue{
							Type: v1.ParamTypeString,
						},
					},
				},
				Workspaces: []v1.PipelineWorkspaceDeclaration{
					{
						Name:        "test",
						Description: "test workspace",
					},
				},
				Params: []v1.ParamSpec{
					{
						Name: "pipeline-param",
						Type: v1.ParamTypeString,
						Default: &v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "rev-param",
						Type: v1.ParamTypeArray,
						Default: &v1.ParamValue{
							Type:     v1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
					{
						Name: "pipeline-param",
						Type: v1.ParamTypeString,
					},
					{
						Name: "rev-param2",
						Type: v1.ParamTypeArray,
					},
				},
			},
		},
	}
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredP(pipelines[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: namespaces, Pipelines: pipelines})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dynamic}

	got, err := test.ExecuteCommand(Command(p), "export", "-n", "ns", "pipeline")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}
