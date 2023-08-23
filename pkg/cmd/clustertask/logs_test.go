// Copyright © 2021 The Tekton Authors.
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

	"github.com/AlecAivazis/survey/v2/core"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestClusterTaskLog(t *testing.T) {
	clock := test.FakeClock()
	clustertask1 := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "task",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		ClusterTasks: clustertask1,
	})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"clustertask", "taskrun"})
	tdc1 := testDynamic.Options{}
	dc1, err := tdc1.Client(
		cb.UnstructuredV1beta1CT(clustertask1[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	testParams := []struct {
		name      string
		command   []string
		input     test.Clients
		dc        dynamic.Interface
		wantError bool
		want      string
	}{
		{
			name:      "Found no taskruns",
			command:   []string{"logs", "task", "-n", "ns"},
			input:     cs,
			dc:        dc1,
			wantError: true,
			want:      "Command \"logs\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nError: no TaskRuns found for ClusterTask task\n",
		},
		{
			name:      "Specify notexist task name",
			command:   []string{"logs", "notexist", "-n", "ns"},
			input:     cs,
			dc:        dc1,
			wantError: true,
			want:      "Command \"logs\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nError: clustertasks.tekton.dev \"notexist\" not found\n",
		},
		{
			name:      "Specify notexist taskrun name",
			command:   []string{"logs", "task", "notexist", "-n", "ns"},
			input:     cs,
			dc:        dc1,
			wantError: true,
			want:      "Command \"logs\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nError: Unable to get TaskRun: taskruns.tekton.dev \"notexist\" not found\n",
		},
		{
			name:      "Specify negative number to limit",
			command:   []string{"logs", "task", "-n", "ns", "--limit", "-1"},
			input:     cs,
			dc:        dc1,
			wantError: true,
			want:      "Command \"logs\" is deprecated, ClusterTasks are deprecated, this command will be removed in future releases.\nError: limit was -1 but must be a positive number\n",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Clock: clock, Kube: tp.input.Kube, Dynamic: tp.dc}
			c := Command(p)

			out, err := test.ExecuteCommand(c, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				}
				test.AssertOutput(t, tp.want, out)
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}

func TestLogs_Auto_Select_FirstClusterTask(t *testing.T) {
	taskName := "dummyTask"
	ns := "dummyNamespaces"
	clock := test.FakeClock()

	ctdata := []*v1beta1.ClusterTask{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: taskName,
			},
		},
	}

	trdata := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      "dummyTR",
				Labels:    map[string]string{"tekton.dev/clusterTask": taskName},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.ClusterTaskKind,
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
					StartTime:      &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		ClusterTasks: ctdata,
		TaskRuns:     trdata,
	})
	cs.Pipeline.Resources = cb.APIResourceList(versionB1, []string{"clustertask", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1CT(ctdata[0], versionB1),
		cb.UnstructuredV1beta1TR(trdata[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := test.Params{
		Kube:    cs.Kube,
		Tekton:  cs.Pipeline,
		Dynamic: dc,
	}
	p.SetNamespace(ns)

	lopt := &options.LogOptions{
		Params: &p,
		Follow: false,
		Limit:  5,
	}
	err = getAllInputs(lopt)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if lopt.ClusterTaskName != taskName {
		t.Error("No auto selection of the first ClusterTask when we have only one")
	}
}
