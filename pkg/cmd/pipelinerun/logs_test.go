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

package pipelinerun

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/pods/fake"
	"github.com/tektoncd/cli/pkg/pods/stream"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

const (
	version        = "v1"
	versionv1beta1 = "v1beta1"
)

func TestLog_invalid_namespace_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client()
	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipelinerun"})
	c := Command(p)

	out, err := test.ExecuteCommand(c, "logs", "pipelinerun", "-n", "invalid")

	if err == nil {
		t.Errorf("Expecting error for invalid namespace")
	}

	expected := "Error: pipelineruns.tekton.dev \"pipelinerun\" not found\n"
	test.AssertOutput(t, expected, out)
}

func TestLog_invalid_namespace(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}
	tdc := testDynamic.Options{}
	dc, _ := tdc.Client()
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	c := Command(p)

	out, err := test.ExecuteCommand(c, "logs", "pipelinerun", "-n", "invalid")

	if err == nil {
		t.Errorf("Expecting error for invalid namespace")
	}

	expected := "Error: pipelineruns.tekton.dev \"pipelinerun\" not found\n"
	test.AssertOutput(t, expected, out)
}

func TestLog_invalid_limit_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}
	c := Command(p)

	limit := "0"
	_, err := test.ExecuteCommand(c, "logs", "-n", "ns", "--limit", limit)

	if err == nil {
		t.Errorf("Expecting error for --limit option being <=0")
	}

	expected := "limit was " + limit + " but must be a positive number"
	test.AssertOutput(t, expected, err.Error())
}

func TestLog_invalid_limit(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}
	c := Command(p)

	limit := "0"
	_, err := test.ExecuteCommand(c, "logs", "-n", "ns", "--limit", limit)

	if err == nil {
		t.Errorf("Expecting error for --limit option being <=0")
	}

	expected := "limit was " + limit + " but must be a positive number"
	test.AssertOutput(t, expected, err.Error())
}

func TestLog_no_pipelinerun_argument_v1beta1(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}
	c := Command(p)

	_, err := test.ExecuteCommand(c, "logs", "-n", "ns")

	if err == nil {
		t.Error("Expecting an error but it's empty")
	}
}

func TestLog_no_pipelinerun_argument(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}
	c := Command(p)

	_, err := test.ExecuteCommand(c, "logs", "-n", "ns")

	if err == nil {
		t.Error("Expecting an error but it's empty")
	}
}

func TestLog_PrStatusToUnixStatus(t *testing.T) {
	testCases := []struct {
		name     string
		pr       *v1.PipelineRun
		expected int
	}{
		{
			name: "No conditions",
			pr: &v1.PipelineRun{
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{},
					},
				},
			},
			expected: 2,
		},
		{
			name: "Condition status is false",
			pr: &v1.PipelineRun{
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "Condition status true",
			pr: &v1.PipelineRun{
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := prStatusToUnixStatus(tc.pr)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestLog_run_found_v1beta1(t *testing.T) {
	clock := test.FakeClock()
	pdata := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-15 * time.Minute)},
			},
		},
	}
	prdata := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipelinerun1",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		Pipelines:    pdata,
		PipelineRuns: prdata,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(pdata[0], versionv1beta1),
		cb.UnstructuredV1beta1PR(prdata[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	c := Command(p)
	_, err = test.ExecuteCommand(c, "logs", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
}

func TestLog_run_found(t *testing.T) {
	clock := test.FakeClock()
	pdata := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-15 * time.Minute)},
			},
		},
	}
	prdata := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipelinerun1",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines:    pdata,
		PipelineRuns: prdata,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(pdata[0], version),
		cb.UnstructuredPR(prdata[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	c := Command(p)
	_, err = test.ExecuteCommand(c, "logs", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
}

func TestLog_run_not_found_v1beta1(t *testing.T) {
	pr := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "output-pipeline-1",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "output-pipeline-1"},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
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

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: pr, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(pr[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	c := Command(p)
	_, err = test.ExecuteCommand(c, "logs", "output-pipeline-2", "-n", "ns")
	expected := "pipelineruns.tekton.dev \"output-pipeline-2\" not found"
	test.AssertOutput(t, expected, err.Error())
}

func TestLog_run_not_found(t *testing.T) {
	pr := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "output-pipeline-1",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "output-pipeline-1"},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1.PipelineRunReasonSuccessful.String(),
						},
					},
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

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: pr, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(pr[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	c := Command(p)
	_, err = test.ExecuteCommand(c, "logs", "output-pipeline-2", "-n", "ns")
	expected := "pipelineruns.tekton.dev \"output-pipeline-2\" not found"
	test.AssertOutput(t, expected, err.Error())
}

func TestPipelinerunLogs_v1beta1(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		prstart      = test.FakeClock()
		ns           = "namespace"

		task1Name    = "output-task"
		tr1Name      = "output-task-1"
		tr1StartTime = prstart.Now().Add(20 * time.Second)
		tr1Pod       = "output-task-pod-123456"
		tr1Step1Name = "writefile-step"
		tr1InitStep1 = "credential-initializer-mdzbr"
		tr1InitStep2 = "place-tools"

		task2Name    = "read-task"
		tr2Name      = "read-task-1"
		tr2StartTime = prstart.Now().Add(2 * time.Minute)
		tr2Pod       = "read-task-pod-123456"
		tr2Step1Name = "readfile-step"

		nopStep = "nop"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      tr1Name,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: task1Name,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: tr1StartTime},
					PodName:   tr1Pod,
					Steps: []v1beta1.StepState{
						{
							Name: tr1Step1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
						{
							Name: nopStep,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      tr2Name,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: task2Name,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: tr2StartTime},
					PodName:   tr2Pod,
					Steps: []v1beta1.StepState{
						{
							Name: tr2Step1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
						{
							Name: nopStep,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
					},
				},
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					ChildReferences: []v1beta1.ChildStatusReference{
						{
							Name:             tr1Name,
							PipelineTaskName: task1Name,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1beta1",
								Kind:       "TaskRun",
							},
						}, {
							Name:             tr2Name,
							PipelineTaskName: task2Name,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1beta1",
								Kind:       "TaskRun",
							},
						},
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}
	pps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: task1Name,
						TaskRef: &v1beta1.TaskRef{
							Name: task1Name,
						},
					},
					{
						Name: task2Name,
						TaskRef: &v1beta1.TaskRef{
							Name: task2Name,
						},
					},
				},
			},
		},
	}

	p := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tr1Pod,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/task": pipelineName},
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
						Name:  tr1InitStep1,
						Image: "override-with-creds:latest",
					},
					{
						Name:  tr1InitStep2,
						Image: "override-with-tools:latest",
					},
				},
				Containers: []corev1.Container{
					{
						Name:  tr1Step1Name,
						Image: tr1Step1Name + ":latest",
					},
					{
						Name:  nopStep,
						Image: "override-with-nop:latest",
					},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPhase(corev1.PodSucceeded),
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  tr1InitStep1,
						Image: "override-with-creds:latest",
					},
					{
						Name:  tr1InitStep2,
						Image: "override-with-tools:latest",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tr2Pod,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/task": pipelineName},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  tr2Step1Name,
						Image: tr1Step1Name + ":latest",
					},
					{
						Name:  nopStep,
						Image: "override-with-nop:latest",
					},
				},
			},
		},
	}

	fakeLogs := fake.Logs(
		fake.Task(tr1Pod,
			fake.Step(tr1InitStep1, "initialized the credentials"),
			fake.Step(tr1InitStep2, "place tools log"),
			fake.Step(tr1Step1Name, "written a file"),
			fake.Step(nopStep, "Build successful"),
		),
		fake.Task(tr2Pod,
			fake.Step(tr2Step1Name, "able to read a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	scenarios := []struct {
		name         string
		allSteps     bool
		tasks        []string
		expectedLogs []string
		prefixing    bool
	}{
		{
			name:      "for all tasks",
			allSteps:  false,
			prefixing: true,
			expectedLogs: []string{
				"[output-task : writefile-step] written a file\n",
				"[output-task : nop] Build successful\n",
				"[read-task : readfile-step] able to read a file\n",
				"[read-task : nop] Build successful\n",
			},
		}, {
			name:      "for all tasks",
			allSteps:  false,
			prefixing: false,
			expectedLogs: []string{
				"written a file\n",
				"Build successful\n",
				"able to read a file\n",
				"Build successful\n",
			},
		}, {
			name:      "for task1 only",
			allSteps:  false,
			prefixing: true,
			tasks:     []string{task1Name},
			expectedLogs: []string{
				"[output-task : writefile-step] written a file\n",
				"[output-task : nop] Build successful\n",
			},
		}, {
			name:      "including init steps",
			allSteps:  true,
			prefixing: true,
			expectedLogs: []string{
				"[output-task : credential-initializer-mdzbr] initialized the credentials\n",
				"[output-task : place-tools] place tools log\n",
				"[output-task : writefile-step] written a file\n",
				"[output-task : nop] Build successful\n",
				"[read-task : readfile-step] able to read a file\n",
				"[read-task : nop] Build successful\n",
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Pipelines: pps, TaskRuns: trs, Pods: p, Namespaces: nsList})
			cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"task", "taskrun", "pipeline", "pipelinerun"})
			tdc := testDynamic.Options{}
			dc, err := tdc.Client(
				cb.UnstructuredV1beta1P(pps[0], versionv1beta1),
				cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
				cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
				cb.UnstructuredV1beta1TR(trs[1], versionv1beta1),
			)
			if err != nil {
				t.Errorf("unable to create dynamic client: %v", err)
			}
			prlo := logOptsv1beta1(prName, ns, cs, dc, fake.Streamer(fakeLogs), s.allSteps, false, s.prefixing, s.tasks...)
			output, _ := fetchLogs(prlo)

			expected := strings.Join(s.expectedLogs, "\n") + "\n"

			test.AssertOutput(t, expected, output)
		})
	}
}

func TestPipelinerunLogs(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		prstart      = test.FakeClock()
		ns           = "namespace"

		task1Name    = "output-task"
		tr1Name      = "output-task-1"
		tr1StartTime = prstart.Now().Add(20 * time.Second)
		tr1Pod       = "output-task-pod-123456"
		tr1Step1Name = "writefile-step"
		tr1InitStep1 = "credential-initializer-mdzbr"
		tr1InitStep2 = "place-tools"

		task2Name    = "read-task"
		tr2Name      = "read-task-1"
		tr2StartTime = prstart.Now().Add(2 * time.Minute)
		tr2Pod       = "read-task-pod-123456"
		tr2Step1Name = "readfile-step"

		nopStep = "nop"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      tr1Name,
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: task1Name,
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: tr1StartTime},
					PodName:   tr1Pod,
					Steps: []v1.StepState{
						{
							Name: tr1Step1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
						{
							Name: nopStep,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      tr2Name,
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: task2Name,
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: tr2StartTime},
					PodName:   tr2Pod,
					Steps: []v1.StepState{
						{
							Name: tr2Step1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
						{
							Name: nopStep,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
					},
				},
			},
		},
	}

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             tr1Name,
							PipelineTaskName: task1Name,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1",
								Kind:       "TaskRun",
							},
						}, {
							Name:             tr2Name,
							PipelineTaskName: task2Name,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1",
								Kind:       "TaskRun",
							},
						},
					},
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}
	pps := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: task1Name,
						TaskRef: &v1.TaskRef{
							Name: task1Name,
						},
					},
					{
						Name: task2Name,
						TaskRef: &v1.TaskRef{
							Name: task2Name,
						},
					},
				},
			},
		},
	}

	p := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tr1Pod,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/task": pipelineName},
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
						Name:  tr1InitStep1,
						Image: "override-with-creds:latest",
					},
					{
						Name:  tr1InitStep2,
						Image: "override-with-tools:latest",
					},
				},
				Containers: []corev1.Container{
					{
						Name:  tr1Step1Name,
						Image: tr1Step1Name + ":latest",
					},
					{
						Name:  nopStep,
						Image: "override-with-nop:latest",
					},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPhase(corev1.PodSucceeded),
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  tr1InitStep1,
						Image: "override-with-creds:latest",
					},
					{
						Name:  tr1InitStep2,
						Image: "override-with-tools:latest",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tr2Pod,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/task": pipelineName},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  tr2Step1Name,
						Image: tr1Step1Name + ":latest",
					},
					{
						Name:  nopStep,
						Image: "override-with-nop:latest",
					},
				},
			},
		},
	}

	fakeLogs := fake.Logs(
		fake.Task(tr1Pod,
			fake.Step(tr1InitStep1, "initialized the credentials"),
			fake.Step(tr1InitStep2, "place tools log"),
			fake.Step(tr1Step1Name, "written a file"),
			fake.Step(nopStep, "Build successful"),
		),
		fake.Task(tr2Pod,
			fake.Step(tr2Step1Name, "able to read a file"),
			fake.Step(nopStep, "Build successful"),
		),
	)

	scenarios := []struct {
		name         string
		allSteps     bool
		tasks        []string
		expectedLogs []string
		prefixing    bool
	}{
		{
			name:      "for all tasks",
			allSteps:  false,
			prefixing: true,
			expectedLogs: []string{
				"[output-task : writefile-step] written a file\n",
				"[output-task : nop] Build successful\n",
				"[read-task : readfile-step] able to read a file\n",
				"[read-task : nop] Build successful\n",
			},
		}, {
			name:      "for all tasks",
			allSteps:  false,
			prefixing: false,
			expectedLogs: []string{
				"written a file\n",
				"Build successful\n",
				"able to read a file\n",
				"Build successful\n",
			},
		}, {
			name:      "for task1 only",
			allSteps:  false,
			prefixing: true,
			tasks:     []string{task1Name},
			expectedLogs: []string{
				"[output-task : writefile-step] written a file\n",
				"[output-task : nop] Build successful\n",
			},
		}, {
			name:      "including init steps",
			allSteps:  true,
			prefixing: true,
			expectedLogs: []string{
				"[output-task : credential-initializer-mdzbr] initialized the credentials\n",
				"[output-task : place-tools] place tools log\n",
				"[output-task : writefile-step] written a file\n",
				"[output-task : nop] Build successful\n",
				"[read-task : readfile-step] able to read a file\n",
				"[read-task : nop] Build successful\n",
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: pps, TaskRuns: trs, Pods: p, Namespaces: nsList})
			cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun", "pipeline", "pipelinerun"})
			tdc := testDynamic.Options{}
			dc, err := tdc.Client(
				cb.UnstructuredP(pps[0], version),
				cb.UnstructuredPR(prs[0], version),
				cb.UnstructuredTR(trs[0], version),
				cb.UnstructuredTR(trs[1], version),
			)
			if err != nil {
				t.Errorf("unable to create dynamic client: %v", err)
			}
			prlo := logOpts(prName, ns, cs, dc, fake.Streamer(fakeLogs), s.allSteps, false, s.prefixing, s.tasks...)
			output, _ := fetchLogs(prlo)

			expected := strings.Join(s.expectedLogs, "\n") + "\n"

			test.AssertOutput(t, expected, output)
		})
	}
}

func updatePRv1beta1(finalRuns []*v1beta1.PipelineRun, watcher *watch.FakeWatcher) {
	go func() {
		for _, pr := range finalRuns {
			time.Sleep(time.Second * 1)
			watcher.Modify(pr)
		}
	}()
}

func updatePR(finalRuns []*v1.PipelineRun, watcher *watch.FakeWatcher) {
	go func() {
		for _, pr := range finalRuns {
			time.Sleep(time.Second * 1)
			watcher.Modify(pr)
		}
	}()
}

func TestPipelinerunLog_completed_taskrun_only_v1beta1(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		prstart      = test.FakeClock()
		ns           = "namespace"

		task1Name    = "output-task"
		tr1Name      = "output-task-1"
		tr1StartTime = prstart.Now().Add(20 * time.Second)
		tr1Pod       = "output-task-pod-123456"
		tr1Step1Name = "writefile-step"

		// these are pipeline tasks for which pipeline has not
		// scheduled any taskrun
		task2Name = "read-task"

		task3Name = "notify"

		task4Name = "teardown"
	)

	trdata := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      tr1Name,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: task1Name,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: tr1StartTime},
					PodName:   tr1Pod,
					Steps: []v1beta1.StepState{
						{
							Name: tr1Step1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: "nop",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
					},
				},
			},
		},
	}

	pdata := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: task1Name,
						TaskRef: &v1beta1.TaskRef{
							Name: task1Name,
						},
					},
					{
						Name: task2Name,
						TaskRef: &v1beta1.TaskRef{
							Name: task2Name,
						},
					},
					{
						Name: task3Name,
						TaskRef: &v1beta1.TaskRef{
							Name: task3Name,
						},
					},
					{
						Name: task4Name,
						TaskRef: &v1beta1.TaskRef{
							Name: task4Name,
						},
					},
				},
			},
		},
	}

	prdata := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonRunning.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					ChildReferences: []v1beta1.ChildStatusReference{
						{
							Name:             tr1Name,
							TypeMeta:         runtime.TypeMeta{Kind: "TaskRun"},
							PipelineTaskName: task1Name,
						},
					},
				},
			},
		},
	}

	// define pipeline in pipelineRef
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		TaskRuns:     trdata,
		Pipelines:    pdata,
		PipelineRuns: prdata,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
		Pods: []*corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tr1Pod,
					Namespace: ns,
					Labels:    map[string]string{"tekton.dev/task": pipelineName},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  tr1Step1Name,
							Image: tr1Step1Name + ":latest",
						},
						{
							Name:  "nop",
							Image: "override-with-nop:latest",
						},
					},
				},
			},
		},
	})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"task", "taskrun", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trdata[0], versionv1beta1),
		cb.UnstructuredV1beta1P(pdata[0], versionv1beta1),
		cb.UnstructuredV1beta1PR(prdata[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	tdata2 := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "output-task2",
				CreationTimestamp: metav1.Time{Time: test.FakeClock().Now()},
			},
		},
	}

	trdata2 := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "output-taskrun2",
				Labels:    map[string]string{"tekton.dev/task": "task"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "output-task2",
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: test.FakeClock().Now()},
					PodName:   "output-task-pod-embedded",
					Steps: []v1beta1.StepState{
						{
							Name: "test-step",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
					},
				},
			},
		},
	}

	prdata2 := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "embedded-pipeline-1",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "embedded-pipeline-1"},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineSpec: &v1beta1.PipelineSpec{
					Tasks: []v1beta1.PipelineTask{
						{
							Name: "output-task2",
							TaskRef: &v1beta1.TaskRef{
								Name: "output-task2",
							},
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonRunning.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					ChildReferences: []v1beta1.ChildStatusReference{
						{
							Name:             "output-taskrun2",
							TypeMeta:         runtime.TypeMeta{Kind: "TaskRun"},
							PipelineTaskName: "output-task2",
						},
					},
				},
			},
		},
	}
	// define embedded pipeline
	cs2, _ := test.SeedV1beta1TestData(t, test.Data{
		Tasks:        tdata2,
		TaskRuns:     trdata2,
		PipelineRuns: prdata2,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
		Pods: []*corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "output-task-pod-embedded",
					Namespace: "ns",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-step",
							Image: "test-step1:latest",
						},
						{
							Name:  "nop2",
							Image: "override-with-nop:latest",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPhase(corev1.PodSucceeded),
				},
			},
		},
	})
	cs2.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"task", "taskrun", "pipeline", "pipelinerun"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1T(tdata2[0], versionv1beta1),
		cb.UnstructuredV1beta1TR(trdata2[0], versionv1beta1),
		cb.UnstructuredV1beta1PR(prdata2[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	for _, tt := range []struct {
		name            string
		pipelineRunName string
		namespace       string
		dynamic         dynamic.Interface
		input           test.Clients
		logs            []fake.Log
		want            []string
		prefixing       bool
	}{
		{
			name:            "Test PipelineRef",
			pipelineRunName: prName,
			namespace:       ns,
			dynamic:         dc,
			input:           cs,
			prefixing:       true,
			logs: fake.Logs(
				fake.Task(tr1Pod,
					fake.Step(tr1Step1Name, "wrote a file"),
					fake.Step("nop", "Build successful"),
				),
			),
			want: []string{
				"[output-task : writefile-step] wrote a file\n",
				"[output-task : nop] Build successful\n",
				"",
			},
		},
		{
			name:            "Test PipelineRef no prefixing",
			pipelineRunName: prName,
			namespace:       ns,
			dynamic:         dc,
			input:           cs,
			prefixing:       false,
			logs: fake.Logs(
				fake.Task(tr1Pod,
					fake.Step(tr1Step1Name, "wrote a file"),
					fake.Step("nop", "Build successful"),
				),
			),
			want: []string{
				"wrote a file\n",
				"Build successful\n",
				"",
			},
		},
		{
			name:            "Test embedded Pipeline",
			pipelineRunName: "embedded-pipeline-1",
			namespace:       "ns",
			dynamic:         dc2,
			input:           cs2,
			prefixing:       true,
			logs: fake.Logs(
				fake.Task("output-task-pod-embedded",
					fake.Step("test-step", "test embedded"),
					fake.Step("nop2", "Test successful"),
				),
			),
			want: []string{
				"[output-task2 : test-step] test embedded\n",
				"[output-task2 : nop2] Test successful\n",
				"",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prlo := logOptsv1beta1(tt.pipelineRunName, tt.namespace, tt.input, tt.dynamic, fake.Streamer(tt.logs), false, false, tt.prefixing)
			output, _ := fetchLogs(prlo)
			test.AssertOutput(t, strings.Join(tt.want, "\n"), output)
		})
	}
}

func TestPipelinerunLog_follow_mode_v1beta1(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		prstart      = test.FakeClock()
		ns           = "namespace"

		task1Name    = "output-task"
		tr1Name      = "output-task-1"
		tr1StartTime = prstart.Now().Add(20 * time.Second)
		tr1Pod       = "output-task-pod-123456"
		tr1Step1Name = "writefile-step"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      tr1Name,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: task1Name,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					PodName:        tr1Pod,
					StartTime:      &metav1.Time{Time: tr1StartTime},
					CompletionTime: nil,
					Steps: []v1beta1.StepState{
						{
							Name: tr1Step1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: "nop",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
					},
				},
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonRunning.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					ChildReferences: []v1beta1.ChildStatusReference{
						{
							Name:             tr1Name,
							PipelineTaskName: task1Name,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1beta1",
								Kind:       "TaskRun",
							},
						},
					},
				},
			},
		},
	}

	pps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: task1Name,
						TaskRef: &v1beta1.TaskRef{
							Name: task1Name,
						},
					},
				},
			},
		},
	}

	p := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tr1Pod,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/task": pipelineName},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  tr1Step1Name,
						Image: tr1Step1Name + ":latest",
					},
					{
						Name:  "nop",
						Image: "override-with-nop:latest",
					},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPhase(corev1.PodSucceeded),
			},
		},
	}

	fakeLogStream := fake.Logs(
		fake.Task(tr1Pod,
			fake.Step(tr1Step1Name,
				"wrote a file1",
				"wrote a file2",
				"wrote a file3",
				"wrote a file4",
			),
			fake.Step("nop", "Build successful"),
		),
	)

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Pipelines: pps, TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"task", "taskrun", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
		cb.UnstructuredV1beta1P(pps[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	prlo := logOptsv1beta1(prName, ns, cs, dc, fake.Streamer(fakeLogStream), false, true, true)
	output, _ := fetchLogs(prlo)

	expectedLogs := []string{
		"[output-task : writefile-step] wrote a file1",
		"[output-task : writefile-step] wrote a file2",
		"[output-task : writefile-step] wrote a file3",
		"[output-task : writefile-step] wrote a file4\n",
		"[output-task : nop] Build successful\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"
	test.AssertOutput(t, expected, output)
}

func TestPipelinerunLog_follow_mode(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		prstart      = test.FakeClock()
		ns           = "namespace"

		task1Name    = "output-task"
		tr1Name      = "output-task-1"
		tr1StartTime = prstart.Now().Add(20 * time.Second)
		tr1Pod       = "output-task-pod-123456"
		tr1Step1Name = "writefile-step"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      tr1Name,
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: task1Name,
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					PodName:        tr1Pod,
					StartTime:      &metav1.Time{Time: tr1StartTime},
					CompletionTime: nil,
					Steps: []v1.StepState{
						{
							Name: tr1Step1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
						{
							Name: "nop",
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
					},
				},
			},
		},
	}

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1.PipelineRunReasonRunning.String(),
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             tr1Name,
							PipelineTaskName: task1Name,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1",
								Kind:       "TaskRun",
							},
						},
					},
				},
			},
		},
	}

	pps := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: task1Name,
						TaskRef: &v1.TaskRef{
							Name: task1Name,
						},
					},
				},
			},
		},
	}

	p := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tr1Pod,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/task": pipelineName},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  tr1Step1Name,
						Image: tr1Step1Name + ":latest",
					},
					{
						Name:  "nop",
						Image: "override-with-nop:latest",
					},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPhase(corev1.PodSucceeded),
			},
		},
	}

	fakeLogStream := fake.Logs(
		fake.Task(tr1Pod,
			fake.Step(tr1Step1Name,
				"wrote a file1",
				"wrote a file2",
				"wrote a file3",
				"wrote a file4",
			),
			fake.Step("nop", "Build successful"),
		),
	)

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: pps, TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
		cb.UnstructuredPR(prs[0], version),
		cb.UnstructuredP(pps[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	prlo := logOpts(prName, ns, cs, dc, fake.Streamer(fakeLogStream), false, true, true)
	output, _ := fetchLogs(prlo)

	expectedLogs := []string{
		"[output-task : writefile-step] wrote a file1",
		"[output-task : writefile-step] wrote a file2",
		"[output-task : writefile-step] wrote a file3",
		"[output-task : writefile-step] wrote a file4\n",
		"[output-task : nop] Build successful\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"
	test.AssertOutput(t, expected, output)
}

func TestLogs_error_log_v1beta1(t *testing.T) {
	var (
		pipelineName = "errlogs-pipeline"
		prName       = "errlogs-run"
		ns           = "namespace"
		taskName     = "errlogs-task"
		errMsg       = "Pipeline tektoncd/errlog-pipeline can't be Run; it contains Tasks that don't exist: Couldn't retrieve Task errlog-tasks: task.tekton.dev errlog-tasks not found"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	ts := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      taskName,
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Message: errMsg,
						},
					},
				},
			},
		},
	}

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1beta1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Pipelines: ps, Tasks: ts, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"task", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1T(ts[0], versionv1beta1),
		cb.UnstructuredV1beta1P(ps[0], versionv1beta1),
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOptsv1beta1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false, true)
	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, errMsg+"\n", output)
}

func TestLogs_error_log(t *testing.T) {
	var (
		pipelineName = "errlogs-pipeline"
		prName       = "errlogs-run"
		ns           = "namespace"
		taskName     = "errlogs-task"
		errMsg       = "Pipeline tektoncd/errlog-pipeline can't be Run; it contains Tasks that don't exist: Couldn't retrieve Task errlog-tasks: task.tekton.dev errlog-tasks not found"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	ts := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      taskName,
			},
		},
	}

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionFalse,
							Message: errMsg,
						},
					},
				},
			},
		},
	}

	ps := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: ps, Tasks: ts, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredT(ts[0], version),
		cb.UnstructuredP(ps[0], version),
		cb.UnstructuredPR(prs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOpts(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false, true)
	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, errMsg+"\n", output)
}

func TestLogs_nologs_v1beta1(t *testing.T) {
	var (
		pipelineName = "nologs-pipeline"
		prName       = "nologs-run"
		ns           = "namespace"
		taskName     = "nologs-task"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1beta1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Pipelines: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], versionv1beta1),
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOptsv1beta1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false, true)
	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, "PipelineRun is still running: Running\n", output)
}

func TestLogs_nologs(t *testing.T) {
	var (
		pipelineName = "nologs-pipeline"
		prName       = "nologs-run"
		ns           = "namespace"
		taskName     = "nologs-task"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	ps := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], version),
		cb.UnstructuredPR(prs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOpts(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false, true)
	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, "PipelineRun is still running: Running\n", output)
}

func TestLog_run_failed_with_and_without_follow_v1beta1(t *testing.T) {
	var (
		pipelineName = "fail-pipeline"
		prName       = "fail-run"
		ns           = "namespace"
		taskName     = "fail-task"
		failMessage  = "Failed because I wanted"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  corev1.ConditionFalse,
							Message: failMessage,
						},
					},
				},
			},
		},
	}

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1beta1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Pipelines: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], versionv1beta1),
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	// follow mode disabled
	prlo := logOptsv1beta1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false, true)
	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, failMessage+"\n", output)

	// follow mode enabled
	prlo = logOptsv1beta1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, true, true)
	output, err = fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, failMessage+"\n", output)
}

func TestLog_run_failed_with_and_without_follow(t *testing.T) {
	var (
		pipelineName = "fail-pipeline"
		prName       = "fail-run"
		ns           = "namespace"
		taskName     = "fail-task"
		failMessage  = "Failed because I wanted"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  corev1.ConditionFalse,
							Message: failMessage,
						},
					},
				},
			},
		},
	}

	ps := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], version),
		cb.UnstructuredPR(prs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	// follow mode disabled
	prlo := logOpts(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false, true)
	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, failMessage+"\n", output)

	// follow mode enabled
	prlo = logOpts(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, true, true)
	output, err = fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, failMessage+"\n", output)
}

func TestLog_pipelinerun_still_running_v1beta1(t *testing.T) {
	var (
		pipelineName = "inprogress-pipeline"
		prName       = "inprogress-run"
		ns           = "namespace"
		taskName     = "inprogress-task"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	initialPRs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionUnknown,
							Type:    apis.ConditionSucceeded,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	finalPRs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionUnknown,
							Type:    apis.ConditionSucceeded,
							Message: "Running",
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Type:    apis.ConditionSucceeded,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1beta1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: initialPRs, Pipelines: ps, Namespaces: nsList})
	watcher := watch.NewFake()
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"taskrun", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{WatchResource: "pipelineruns", Watcher: watcher}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], versionv1beta1),
		cb.UnstructuredV1beta1PR(initialPRs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOptsv1beta1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false, true)

	updatePRv1beta1(finalPRs, watcher)

	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, "Pipeline still running ..."+"\n", output)
}

func TestLog_pipelinerun_still_running(t *testing.T) {
	var (
		pipelineName = "inprogress-pipeline"
		prName       = "inprogress-run"
		ns           = "namespace"
		taskName     = "inprogress-task"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	initialPRs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionUnknown,
							Type:    apis.ConditionSucceeded,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	finalPRs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionUnknown,
							Type:    apis.ConditionSucceeded,
							Message: "Running",
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Type:    apis.ConditionSucceeded,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	ps := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: initialPRs, Pipelines: ps, Namespaces: nsList})
	watcher := watch.NewFake()
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun", "pipeline", "pipelinerun"})
	tdc := testDynamic.Options{WatchResource: "pipelineruns", Watcher: watcher}
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], version),
		cb.UnstructuredPR(initialPRs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOpts(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false, true)

	updatePR(finalPRs, watcher)

	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, "Pipeline still running ..."+"\n", output)
}

func TestLog_pipelinerun_status_done_v1beta1(t *testing.T) {
	var (
		pipelineName = "done-pipeline"
		prName       = "done-run"
		ns           = "namespace"
		taskName     = "done-task"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1beta1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Pipelines: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	watcher := watch.NewFake()
	tdc := testDynamic.Options{WatchResource: "pipelineruns", Watcher: watcher}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], versionv1beta1),
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOptsv1beta1(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false, true)

	go func() {
		time.Sleep(time.Second * 1)
		for _, pr := range prs {
			pr.Status.Conditions[0].Status = corev1.ConditionTrue
			pr.Status.Conditions[0].Message = "completed"
			watcher.Modify(pr)
		}
	}()

	start := time.Now()
	output, err := fetchLogs(prlo)
	elapsed := time.Since(start).Seconds()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if elapsed > 10 {
		t.Errorf("Timed out")
	}
	test.AssertOutput(t, "", output)
}

func TestLog_pipelinerun_status_done(t *testing.T) {
	var (
		pipelineName = "done-pipeline"
		prName       = "done-run"
		ns           = "namespace"
		taskName     = "done-task"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	ps := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: ps, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	watcher := watch.NewFake()
	tdc := testDynamic.Options{WatchResource: "pipelineruns", Watcher: watcher}
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], version),
		cb.UnstructuredPR(prs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	prlo := logOpts(prName, ns, cs, dc, fake.Streamer([]fake.Log{}), false, false, true)

	go func() {
		time.Sleep(time.Second * 1)
		for _, pr := range prs {
			pr.Status.Conditions[0].Status = corev1.ConditionTrue
			pr.Status.Conditions[0].Message = "completed"
			watcher.Modify(pr)
		}
	}()

	start := time.Now()
	output, err := fetchLogs(prlo)
	elapsed := time.Since(start).Seconds()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if elapsed > 10 {
		t.Errorf("Timed out")
	}
	test.AssertOutput(t, "", output)
}

func TestLog_pipelinerun_last_v1beta1(t *testing.T) {
	var (
		pipelineName = "pipeline1"
		prName       = "pr1"
		prName2      = "pr2"
		ns           = "namespaces"
		taskName     = "task1"
	)

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}
	pipelines := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1beta1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	pr1StartTime := metav1.NewTime(time.Now())
	pr2StartTime := metav1.NewTime(time.Now().Add(time.Second))

	pipelineruns := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName2,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime: &pr2StartTime,
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					StartTime: &pr1StartTime,
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: pipelineruns, Pipelines: pipelines, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(pipelineruns[0], versionv1beta1),
		cb.UnstructuredV1beta1PR(pipelineruns[1], versionv1beta1),
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
	lopt := options.LogOptions{
		Params: &p,
		Last:   true,
		Limit:  len(pipelineruns),
	}
	err = askRunName(&lopt)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, prName2, lopt.PipelineRunName)
}

func TestLog_pipelinerun_last(t *testing.T) {
	var (
		pipelineName = "pipeline1"
		prName       = "pr1"
		prName2      = "pr2"
		ns           = "namespaces"
		taskName     = "task1"
	)

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}
	pipelines := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	pr1StartTime := metav1.NewTime(time.Now())
	pr2StartTime := metav1.NewTime(time.Now().Add(time.Second))

	pipelineruns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName2,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					StartTime: &pr2StartTime,
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					StartTime: &pr1StartTime,
				},
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: pipelineruns, Pipelines: pipelines, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(pipelineruns[0], version),
		cb.UnstructuredPR(pipelineruns[1], version),
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
	lopt := options.LogOptions{
		Params: &p,
		Last:   true,
		Limit:  len(pipelineruns),
	}
	err = askRunName(&lopt)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, prName2, lopt.PipelineRunName)
}

func TestLog_pipelinerun_only_one_v1beta1(t *testing.T) {
	var (
		pipelineName = "pipeline1"
		prName       = "pr1"
		ns           = "namespaces"
		taskName     = "task1"
	)

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	pipelines := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1beta1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	pipelineruns := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: pipelineruns, Pipelines: pipelines, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1PR(pipelineruns[0], versionv1beta1),
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
	lopt := options.LogOptions{
		Params: &p,
		// This code https://git.io/JvCMV seems buggy so have to set the upper
		// Limit.. but I guess that's another fight for another day.
		Limit: len(pipelineruns),
	}
	err = askRunName(&lopt)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, prName, lopt.PipelineRunName)
}

func TestLog_pipelinerun_only_one(t *testing.T) {
	var (
		pipelineName = "pipeline1"
		prName       = "pr1"
		ns           = "namespaces"
		taskName     = "task1"
	)

	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	pipelines := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	pipelineruns := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  corev1.ConditionUnknown,
							Message: "Running",
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: pipelineruns, Pipelines: pipelines, Namespaces: namespaces})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredPR(pipelineruns[0], version),
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
	lopt := options.LogOptions{
		Params: &p,
		// This code https://git.io/JvCMV seems buggy so have to set the upper
		// Limit.. but I guess that's another fight for another day.
		Limit: len(pipelineruns),
	}
	err = askRunName(&lopt)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	test.AssertOutput(t, prName, lopt.PipelineRunName)
}

func TestPipelinerunLog_finally_v1beta1(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		prstart      = test.FakeClock()
		ns           = "namespace"

		task1Name    = "output-task"
		tr1Name      = "output-task-1"
		tr1StartTime = prstart.Now().Add(20 * time.Second)
		tr1Pod       = "output-task-pod-123456"
		tr1Step1Name = "writefile-step"

		finallyName        = "finally-task"
		finallyTrName      = "finally-task-1"
		finallyStartTime   = prstart.Now().Add(30 * time.Second)
		finallyTrPod       = "finally-task-pod-123456"
		finallyTrStep1Name = "finally-step"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      tr1Name,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: task1Name,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					PodName:        tr1Pod,
					StartTime:      &metav1.Time{Time: tr1StartTime},
					CompletionTime: nil,
					Steps: []v1beta1.StepState{
						{
							Name: tr1Step1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      finallyTrName,
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: finallyName,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					PodName:        finallyTrPod,
					StartTime:      &metav1.Time{Time: finallyStartTime},
					CompletionTime: nil,
					Steps: []v1beta1.StepState{
						{
							Name: finallyTrStep1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
					},
				},
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonRunning.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					ChildReferences: []v1beta1.ChildStatusReference{
						{
							PipelineTaskName: task1Name,
							Name:             tr1Name,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1beta1",
								Kind:       "TaskRun",
							},
						},
						{
							PipelineTaskName: finallyName,
							Name:             finallyTrName,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1beta1",
								Kind:       "TaskRun",
							},
						},
					},
				},
			},
		},
	}

	pps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: task1Name,
						TaskRef: &v1beta1.TaskRef{
							Name: task1Name,
						},
					},
				},
				Finally: []v1beta1.PipelineTask{
					{
						Name: finallyName,
						TaskRef: &v1beta1.TaskRef{
							Name: finallyName,
						},
					},
				},
			},
		},
	}

	p := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tr1Pod,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/task": pipelineName},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  tr1Step1Name,
						Image: tr1Step1Name + ":latest",
					},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPhase(corev1.PodSucceeded),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      finallyTrPod,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/task": pipelineName},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  finallyTrStep1Name,
						Image: finallyTrStep1Name + ":latest",
					},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPhase(corev1.PodSucceeded),
			},
		},
	}

	fakeLogStream := fake.Logs(
		fake.Task(tr1Pod,
			fake.Step(tr1Step1Name, "wrote a file1"),
		),
		fake.Task(finallyTrPod,
			fake.Step(finallyTrStep1Name, "Finally"),
		),
	)

	cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Pipelines: pps, TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"task", "taskrun", "pipeline", "pipelinerun"})

	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1TR(trs[0], versionv1beta1),
		cb.UnstructuredV1beta1TR(trs[1], versionv1beta1),
		cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
		cb.UnstructuredV1beta1P(pps[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	prlo := logOptsv1beta1(prName, ns, cs, dc, fake.Streamer(fakeLogStream), false, false, true)

	output, _ := fetchLogs(prlo)

	expectedLogs := []string{
		"[output-task : writefile-step] wrote a file1\n",
		"[finally-task : finally-step] Finally\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"
	test.AssertOutput(t, expected, output)
}

func TestPipelinerunLog_finally(t *testing.T) {
	var (
		pipelineName = "output-pipeline"
		prName       = "output-pipeline-1"
		prstart      = test.FakeClock()
		ns           = "namespace"

		task1Name    = "output-task"
		tr1Name      = "output-task-1"
		tr1StartTime = prstart.Now().Add(20 * time.Second)
		tr1Pod       = "output-task-pod-123456"
		tr1Step1Name = "writefile-step"

		finallyName        = "finally-task"
		finallyTrName      = "finally-task-1"
		finallyStartTime   = prstart.Now().Add(30 * time.Second)
		finallyTrPod       = "finally-task-pod-123456"
		finallyTrStep1Name = "finally-step"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      tr1Name,
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: task1Name,
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					PodName:        tr1Pod,
					StartTime:      &metav1.Time{Time: tr1StartTime},
					CompletionTime: nil,
					Steps: []v1.StepState{
						{
							Name: tr1Step1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      finallyTrName,
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: finallyName,
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					PodName:        finallyTrPod,
					StartTime:      &metav1.Time{Time: finallyStartTime},
					CompletionTime: nil,
					Steps: []v1.StepState{
						{
							Name: finallyTrStep1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
					},
				},
			},
		},
	}

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonRunning.String(),
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							PipelineTaskName: task1Name,
							Name:             tr1Name,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1",
								Kind:       "TaskRun",
							},
						},
						{
							PipelineTaskName: finallyName,
							Name:             finallyTrName,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1",
								Kind:       "TaskRun",
							},
						},
					},
				},
			},
		},
	}

	pps := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: task1Name,
						TaskRef: &v1.TaskRef{
							Name: task1Name,
						},
					},
				},
				Finally: []v1.PipelineTask{
					{
						Name: finallyName,
						TaskRef: &v1.TaskRef{
							Name: finallyName,
						},
					},
				},
			},
		},
	}

	p := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tr1Pod,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/task": pipelineName},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  tr1Step1Name,
						Image: tr1Step1Name + ":latest",
					},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPhase(corev1.PodSucceeded),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      finallyTrPod,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/task": pipelineName},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  finallyTrStep1Name,
						Image: finallyTrStep1Name + ":latest",
					},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPhase(corev1.PodSucceeded),
			},
		},
	}

	fakeLogStream := fake.Logs(
		fake.Task(tr1Pod,
			fake.Step(tr1Step1Name, "wrote a file1"),
		),
		fake.Task(finallyTrPod,
			fake.Step(finallyTrStep1Name, "Finally"),
		),
	)

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: pps, TaskRuns: trs, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun", "pipeline", "pipelinerun"})

	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
		cb.UnstructuredTR(trs[1], version),
		cb.UnstructuredPR(prs[0], version),
		cb.UnstructuredP(pps[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	prlo := logOpts(prName, ns, cs, dc, fake.Streamer(fakeLogStream), false, false, true)
	output, _ := fetchLogs(prlo)

	expectedLogs := []string{
		"[output-task : writefile-step] wrote a file1\n",
		"[finally-task : finally-step] Finally\n",
	}
	expected := strings.Join(expectedLogs, "\n") + "\n"
	test.AssertOutput(t, expected, output)
}

func TestLogs_Cluster_Resolver(t *testing.T) {
	var (
		pipelineName = "pipeline"
		prName       = "pipeline-run"
		ns           = "namespace"
		taskName     = "task"
		trName       = "taskrun"
		taskPodName  = "taskPod"
		tr1StartTime = test.FakeClock().Now().Add(20 * time.Second)
		tr1Step1Name = "step-1"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					ResolverRef: v1.ResolverRef{
						Resolver: "cluster",
						Params: v1.Params{
							{
								Name:  "kind",
								Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "pipeline"},
							},
							{
								Name:  "name",
								Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: pipelineName},
							},
							{
								Name:  "namespace",
								Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: ns},
							},
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Message: "Success",
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             trName,
							PipelineTaskName: taskName,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1beta1",
								Kind:       "TaskRun",
							},
						},
					},
					PipelineSpec: &v1.PipelineSpec{
						Tasks: []v1.PipelineTask{
							{
								Name: taskName,
								TaskRef: &v1.TaskRef{
									Kind: "task",
									Name: taskName,
								},
							},
						},
					},
				},
			},
		},
	}

	ps := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	p := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      taskPodName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/task": pipelineName},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  tr1Step1Name,
						Image: tr1Step1Name + ":latest",
					},
				},
			},
		},
	}

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: tr1StartTime},
					PodName:   taskPodName,
					Steps: []v1.StepState{
						{
							Name: tr1Step1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: ps, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], version),
		cb.UnstructuredPR(prs[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	fakeLogStream := fake.Logs(
		fake.Task(taskPodName,
			fake.Step(tr1Step1Name, "task-1 completed\n"),
		),
	)
	prlo := logOpts(prName, ns, cs, dc, fake.Streamer(fakeLogStream), false, false, false, taskName)

	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "task-1 completed\n\n", output)
}

func TestLogs_Git_Resolver(t *testing.T) {
	var (
		pipelineName = "pipeline"
		prName       = "pipeline-run"
		ns           = "namespace"
		taskName     = "task"
		trName       = "taskrun"
		taskPodName  = "taskPod"
		tr1StartTime = test.FakeClock().Now().Add(20 * time.Second)
		tr1Step1Name = "step-1"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					ResolverRef: v1.ResolverRef{
						Resolver: "git",
						Params: v1.Params{
							{
								Name:  "kind",
								Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "pipeline"},
							},
							{
								Name:  "name",
								Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: pipelineName},
							},
							{
								Name:  "namespace",
								Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: ns},
							},
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Message: "Success",
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             trName,
							PipelineTaskName: taskName,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1beta1",
								Kind:       "TaskRun",
							},
						},
					},
					PipelineSpec: &v1.PipelineSpec{
						Tasks: []v1.PipelineTask{
							{
								Name: taskName,
								TaskRef: &v1.TaskRef{
									Kind: "task",
									Name: taskName,
								},
							},
						},
					},
				},
			},
		},
	}

	ps := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	p := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      taskPodName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/task": pipelineName},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  tr1Step1Name,
						Image: tr1Step1Name + ":latest",
					},
				},
			},
		},
	}

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: tr1StartTime},
					PodName:   taskPodName,
					Steps: []v1.StepState{
						{
							Name: tr1Step1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: ps, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], version),
		cb.UnstructuredPR(prs[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	fakeLogStream := fake.Logs(
		fake.Task(taskPodName,
			fake.Step(tr1Step1Name, "task-1 completed\n"),
		),
	)
	prlo := logOpts(prName, ns, cs, dc, fake.Streamer(fakeLogStream), false, false, false, taskName)

	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "task-1 completed\n\n", output)
}

func TestLogs_Bundle_Resolver(t *testing.T) {
	var (
		pipelineName = "pipeline"
		prName       = "pipeline-run"
		ns           = "namespace"
		taskName     = "task"
		trName       = "taskrun"
		taskPodName  = "taskPod"
		tr1StartTime = test.FakeClock().Now().Add(20 * time.Second)
		tr1Step1Name = "step-1"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					ResolverRef: v1.ResolverRef{
						Resolver: "bundle",
						Params: v1.Params{
							{
								Name:  "kind",
								Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "pipeline"},
							},
							{
								Name:  "name",
								Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: pipelineName},
							},
							{
								Name:  "namespace",
								Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: ns},
							},
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Message: "Success",
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             trName,
							PipelineTaskName: taskName,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1beta1",
								Kind:       "TaskRun",
							},
						},
					},
					PipelineSpec: &v1.PipelineSpec{
						Tasks: []v1.PipelineTask{
							{
								Name: taskName,
								TaskRef: &v1.TaskRef{
									Kind: "task",
									Name: taskName,
								},
							},
						},
					},
				},
			},
		},
	}

	ps := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	p := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      taskPodName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/task": pipelineName},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  tr1Step1Name,
						Image: tr1Step1Name + ":latest",
					},
				},
			},
		},
	}

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: tr1StartTime},
					PodName:   taskPodName,
					Steps: []v1.StepState{
						{
							Name: tr1Step1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: ps, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], version),
		cb.UnstructuredPR(prs[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	fakeLogStream := fake.Logs(
		fake.Task(taskPodName,
			fake.Step(tr1Step1Name, "task-1 completed\n"),
		),
	)
	prlo := logOpts(prName, ns, cs, dc, fake.Streamer(fakeLogStream), false, false, false, taskName)

	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "task-1 completed\n\n", output)
}

func TestLogs_Hub_Resolver(t *testing.T) {
	var (
		pipelineName = "pipeline"
		prName       = "pipeline-run"
		ns           = "namespace"
		taskName     = "task"
		trName       = "taskrun"
		taskPodName  = "taskPod"
		tr1StartTime = test.FakeClock().Now().Add(20 * time.Second)
		tr1Step1Name = "step-1"
	)

	nsList := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
	}

	prs := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/pipeline": prName},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					ResolverRef: v1.ResolverRef{
						Resolver: "hub",
						Params: v1.Params{
							{
								Name:  "kind",
								Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: "pipeline"},
							},
							{
								Name:  "name",
								Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: pipelineName},
							},
							{
								Name:  "namespace",
								Value: v1.ParamValue{Type: v1.ParamTypeString, StringVal: ns},
							},
						},
					},
				},
			},
			Status: v1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status:  corev1.ConditionTrue,
							Message: "Success",
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					ChildReferences: []v1.ChildStatusReference{
						{
							Name:             trName,
							PipelineTaskName: taskName,
							TypeMeta: runtime.TypeMeta{
								APIVersion: "tekton.dev/v1beta1",
								Kind:       "TaskRun",
							},
						},
					},
					PipelineSpec: &v1.PipelineSpec{
						Tasks: []v1.PipelineTask{
							{
								Name: taskName,
								TaskRef: &v1.TaskRef{
									Kind: "task",
									Name: taskName,
								},
							},
						},
					},
				},
			},
		},
	}

	ps := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
			Spec: v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: taskName,
						TaskRef: &v1.TaskRef{
							Name: taskName,
						},
					},
				},
			},
		},
	}

	p := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      taskPodName,
				Namespace: ns,
				Labels:    map[string]string{"tekton.dev/task": pipelineName},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  tr1Step1Name,
						Image: tr1Step1Name + ":latest",
					},
				},
			},
		},
	}

	trs := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      trName,
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: taskName,
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Type:   apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: tr1StartTime},
					PodName:   taskPodName,
					Steps: []v1.StepState{
						{
							Name: tr1Step1Name,
							ContainerState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
					},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Pipelines: ps, Pods: p, Namespaces: nsList})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], version),
		cb.UnstructuredPR(prs[0], version),
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	fakeLogStream := fake.Logs(
		fake.Task(taskPodName,
			fake.Step(tr1Step1Name, "task-1 completed\n"),
		),
	)
	prlo := logOpts(prName, ns, cs, dc, fake.Streamer(fakeLogStream), false, false, false, taskName)

	output, err := fetchLogs(prlo)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	test.AssertOutput(t, "task-1 completed\n\n", output)
}

func logOptsv1beta1(name string, ns string, cs test.Clients, dc dynamic.Interface, streamer stream.NewStreamerFunc, allSteps bool, follow bool, prefixing bool, tasks ...string) *options.LogOptions {
	p := test.Params{
		Kube:    cs.Kube,
		Tekton:  cs.Pipeline,
		Dynamic: dc,
	}
	p.SetNamespace(ns)

	logOptions := options.LogOptions{
		PipelineRunName: name,
		Tasks:           tasks,
		AllSteps:        allSteps,
		Follow:          follow,
		Params:          &p,
		Streamer:        streamer,
		Prefixing:       prefixing,
	}

	return &logOptions
}

func logOpts(name string, ns string, cs pipelinetest.Clients, dc dynamic.Interface, streamer stream.NewStreamerFunc, allSteps bool, follow bool, prefixing bool, tasks ...string) *options.LogOptions {
	p := test.Params{
		Kube:    cs.Kube,
		Tekton:  cs.Pipeline,
		Dynamic: dc,
	}
	p.SetNamespace(ns)

	logOptions := options.LogOptions{
		PipelineRunName: name,
		Tasks:           tasks,
		AllSteps:        allSteps,
		Follow:          follow,
		Params:          &p,
		Streamer:        streamer,
		Prefixing:       prefixing,
	}

	return &logOptions
}

func fetchLogs(lo *options.LogOptions) (string, error) {
	out := new(bytes.Buffer)
	lo.Stream = &cli.Stream{Out: out, Err: out}
	err := Run(lo)
	return out.String(), err
}
