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
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stest "k8s.io/client-go/testing"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func Test_ClusterTaskDescribe(t *testing.T) {
	clock := test.FakeClock()

	clustertasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "clustertask-full",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(1 * time.Minute)},
			},
			Spec: v1beta1.TaskSpec{
				Params: []v1beta1.ParamSpec{
					{
						Name:        "myarg",
						Type:        v1beta1.ParamTypeString,
						Description: "param type is string",
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "default",
						},
					},
					{
						Name:        "print",
						Type:        v1beta1.ParamTypeArray,
						Description: "param type is array",
						Default: &v1beta1.ParamValue{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
				Steps: []v1beta1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
					{
						Name:  "exit",
						Image: "busybox",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "clustertask-one-everything",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(1 * time.Minute)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a test description",
				Params: []v1beta1.ParamSpec{
					{
						Name: "myarg",
						Type: v1beta1.ParamTypeString,
					},
				},
				Steps: []v1beta1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "clustertask-justname",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
	}

	taskruns := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-1",
				Namespace: "ns",
				Labels: map[string]string{
					"tekton.dev/clusterTask": "clustertask-full",
				},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "clustertask-full",
					Kind: v1beta1.ClusterTaskKind,
				},
				ServiceAccountName: "svc",
				Params: []v1beta1.Param{
					{
						Name: "myarg",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "value",
						},
					},
					{
						Name: "print",
						Value: v1beta1.ParamValue{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(17 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-2",
				Namespace: "ns",
				Labels: map[string]string{
					"tekton.dev/clusterTask": "clustertask-one-everything",
				},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "clustertask-one-everything",
					Kind: v1beta1.ClusterTaskKind,
				},
				ServiceAccountName: "svc",
				Params: []v1beta1.Param{
					{
						Name: "myarg",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "value",
						},
					},
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(17 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "taskrun-3",
				Namespace: "ns",
				Labels: map[string]string{
					"tekton.dev/clusterTask": "clustertask-full",
				},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "clustertask-full",
					Kind: v1beta1.ClusterTaskKind,
				},
				ServiceAccountName: "svc",
				Params: []v1beta1.Param{
					{
						Name: "myarg",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "value",
						},
					},
					{
						Name: "print",
						Value: v1beta1.ParamValue{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: v1beta1.TaskRunReasonRunning.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: clock.Now().Add(-12 * time.Minute)},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], version),
		cb.UnstructuredV1beta1CT(clustertasks[1], version),
		cb.UnstructuredV1beta1CT(clustertasks[2], version),
		cb.UnstructuredV1beta1TR(taskruns[0], version),
		cb.UnstructuredV1beta1TR(taskruns[1], version),
		cb.UnstructuredV1beta1TR(taskruns[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{ClusterTasks: clustertasks, Namespaces: ns, TaskRuns: taskruns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	p.SetNamespace("ns")

	tdc2 := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{{
			Verb:     "list",
			Resource: "taskruns",
			Action: func(_ k8stest.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("fake list taskrun error")
			}}}}
	dynamic2, err := tdc2.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], version),
		cb.UnstructuredV1beta1CT(clustertasks[1], version),
		cb.UnstructuredV1beta1CT(clustertasks[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	cs2, _ := test.SeedV1beta1TestData(t, test.Data{ClusterTasks: clustertasks, Namespaces: ns})
	cs2.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask", "taskrun"})
	p2 := &test.Params{Tekton: cs2.Pipeline, Kube: cs2.Kube, Dynamic: dynamic2, Clock: clock}
	p2.SetNamespace("ns")

	testParams := []struct {
		name        string
		command     []string
		param       *test.Params
		inputStream io.Reader
		wantError   bool
	}{
		{
			name:        "Describe with no arguments",
			command:     []string{"describe", "notexist"},
			param:       p,
			inputStream: nil,
			wantError:   true,
		},
		{
			name:        "Describe clustertask with name only",
			command:     []string{"describe", "clustertask-justname"},
			param:       p,
			inputStream: nil,
			wantError:   false,
		},
		{
			name:        "Describe full clustertask multiple taskruns",
			command:     []string{"describe", "clustertask-full"},
			param:       p,
			inputStream: nil,
			wantError:   false,
		},
		{
			name:        "Describe clustertask missing param default one of everything",
			command:     []string{"describe", "clustertask-one-everything"},
			param:       p,
			inputStream: nil,
			wantError:   false,
		},
		{
			name:        "Failure from listing taskruns",
			command:     []string{"describe", "clustertask-full"},
			param:       p2,
			inputStream: nil,
			wantError:   true,
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			clustertask := Command(tp.param)

			if tp.inputStream != nil {
				clustertask.SetIn(tp.inputStream)
			}

			got, err := test.ExecuteCommand(clustertask, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("Error expected here")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error")
				}
				golden.Assert(t, got, strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
			}
		})
	}
}

func TestClusterTaskDescribe_WithoutNameIfOnlyOneClusterTaskPresent(t *testing.T) {
	cstasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "task-1",
			},
		},
	}
	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CT(cstasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{ClusterTasks: cstasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	clusterTask := Command(p)
	out, err := test.ExecuteCommand(clusterTask, "desc")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestClusterTask_custom_output(t *testing.T) {
	name := "clustertask"
	expected := "Command \"describe\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nclustertask.tekton.dev/" + name

	clock := test.FakeClock()

	cstasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		ClusterTasks: cstasks,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "",
				},
			},
		},
	})

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CT(cstasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	clustertask := Command(p)
	got, err := test.ExecuteCommand(clustertask, "desc", "-o", "name", name)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	got = strings.TrimSpace(got)
	if got != expected {
		t.Errorf("Result should be '%s' != '%s'", got, expected)
	}
}

func TestClusterTaskV1beta1_custom_output(t *testing.T) {
	name := "clustertask"
	expected := "Command \"describe\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nclustertask.tekton.dev/" + name

	clock := test.FakeClock()

	cstasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		ClusterTasks: cstasks,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "",
				},
			},
		},
	})

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CT(cstasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic, Clock: clock}
	clustertask := Command(p)
	got, err := test.ExecuteCommand(clustertask, "desc", "-o", "name", name)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	got = strings.TrimSpace(got)
	if got != expected {
		t.Errorf("Result should be '%s' != '%s'", got, expected)
	}
}

func TestClusterTaskDescribe_With_Results(t *testing.T) {
	clock := test.FakeClock()

	clustertasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "clustertask-1",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a clustertest description",
				Steps: []v1beta1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
				},
				Results: []v1beta1.TaskResult{
					{
						Name:        "result-1",
						Description: "This is a description for result 1",
					},
					{
						Name:        "result-2",
						Description: "This is a description for result 2",
					},
					{
						Name: "result-3",
					},
				},
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{ClusterTasks: clustertasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Dynamic: dynamic}
	clustertask := Command(p)
	out, err := test.ExecuteCommand(clustertask, "desc", "clustertask-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestClusterTaskDescribe_With_Workspaces(t *testing.T) {
	clock := test.FakeClock()

	clustertasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "clustertask-1",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
			Spec: v1beta1.TaskSpec{
				Description: "a clustertest description",
				Steps: []v1beta1.Step{
					{
						Name:  "hello",
						Image: "busybox",
					},
				},
				Workspaces: []v1beta1.WorkspaceDeclaration{
					{
						Name:        "test",
						Description: "test workspace",
						MountPath:   "/workspace/test/file",
						ReadOnly:    true,
					},
				},
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CT(clustertasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{ClusterTasks: clustertasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Dynamic: dynamic}
	clustertask := Command(p)
	out, err := test.ExecuteCommand(clustertask, "desc", "clustertask-1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestClusterTaskDescribe_WithoutNameIfOnlyOneV1beta1ClusterTaskPresent(t *testing.T) {
	cttasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "task-1",
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CT(cttasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{ClusterTasks: cttasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	cttask := Command(p)
	out, err := test.ExecuteCommand(cttask, "desc")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestClusterTaskDescribe_with_annotations(t *testing.T) {
	cttasks := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clustertask-1",
				Annotations: map[string]string{
					corev1.LastAppliedConfigAnnotation: "LastAppliedConfig",
					"tekton.dev/tags":                  "testing",
				},
			},
		},
	}

	version := "v1beta1"
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client(
		cb.UnstructuredV1beta1CT(cttasks[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{ClusterTasks: cttasks})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"clustertask", "taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	cttask := Command(p)
	out, err := test.ExecuteCommand(cttask, "desc")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}
