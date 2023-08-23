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

package task

import (
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
	goexpect "github.com/Netflix/go-expect"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/pods/fake"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/cli/test/prompt"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

const (
	versionv1beta1 = "v1beta1"
	version        = "v1"
)

func TestTaskLog_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	task1 := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		Tasks: task1,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"task", "taskrun"})
	tdc1 := testDynamic.Options{}
	dc1, err := tdc1.Client(
		cb.UnstructuredV1beta1T(task1[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	task2 := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "namespace",
			},
		},
	}
	cs2, _ := test.SeedV1beta1TestData(t, test.Data{
		Tasks: task2,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})
	cs2.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"task", "taskrun"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1T(task2[0], versionv1beta1),
	)
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
			name:      "Invalid namespace",
			command:   []string{"logs", "-n", "invalid"},
			input:     cs,
			dc:        dc1,
			wantError: true,
			want:      "Error: no Tasks found in namespace invalid\n",
		},
		{
			name:      "Found no tasks",
			command:   []string{"logs", "-n", "ns"},
			input:     cs2,
			dc:        dc2,
			wantError: true,
			want:      "Error: no Tasks found in namespace ns\n",
		},
		{
			name:      "Found no taskruns",
			command:   []string{"logs", "task", "-n", "ns"},
			input:     cs,
			dc:        dc1,
			wantError: true,
			want:      "Error: no TaskRuns found for Task task\n",
		},
		{
			name:      "Specify notexist task name",
			command:   []string{"logs", "notexist", "-n", "ns"},
			input:     cs,
			dc:        dc1,
			wantError: true,
			want:      "Error: tasks.tekton.dev \"notexist\" not found\n",
		},
		{
			name:      "Specify notexist taskrun name",
			command:   []string{"logs", "task", "notexist", "-n", "ns"},
			input:     cs,
			dc:        dc1,
			wantError: true,
			want:      "Error: Unable to get TaskRun: taskruns.tekton.dev \"notexist\" not found\n",
		},
		{
			name:      "Specify negative number to limit",
			command:   []string{"logs", "task", "-n", "ns", "--limit", "-1"},
			input:     cs,
			dc:        dc1,
			wantError: true,
			want:      "Error: limit was -1 but must be a positive number\n",
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

func TestTaskLog(t *testing.T) {
	clock := test.FakeClock()

	task1 := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "ns",
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks: task1,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc1 := testDynamic.Options{}
	dc1, err := tdc1.Client(
		cb.UnstructuredT(task1[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	task2 := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "task",
				Namespace: "namespace",
			},
		},
	}
	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks: task2,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})
	cs2.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredT(task2[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	testParams := []struct {
		name      string
		command   []string
		input     pipelinetest.Clients
		dc        dynamic.Interface
		wantError bool
		want      string
	}{
		{
			name:      "Invalid namespace",
			command:   []string{"logs", "-n", "invalid"},
			input:     cs,
			dc:        dc1,
			wantError: true,
			want:      "Error: no Tasks found in namespace invalid\n",
		},
		{
			name:      "Found no tasks",
			command:   []string{"logs", "-n", "ns"},
			input:     cs2,
			dc:        dc2,
			wantError: true,
			want:      "Error: no Tasks found in namespace ns\n",
		},
		{
			name:      "Found no taskruns",
			command:   []string{"logs", "task", "-n", "ns"},
			input:     cs,
			dc:        dc1,
			wantError: true,
			want:      "Error: no TaskRuns found for Task task\n",
		},
		{
			name:      "Specify notexist task name",
			command:   []string{"logs", "notexist", "-n", "ns"},
			input:     cs,
			dc:        dc1,
			wantError: true,
			want:      "Error: tasks.tekton.dev \"notexist\" not found\n",
		},
		{
			name:      "Specify notexist taskrun name",
			command:   []string{"logs", "task", "notexist", "-n", "ns"},
			input:     cs,
			dc:        dc1,
			wantError: true,
			want:      "Error: Unable to get TaskRun: taskruns.tekton.dev \"notexist\" not found\n",
		},
		{
			name:      "Specify negative number to limit",
			command:   []string{"logs", "task", "-n", "ns", "--limit", "-1"},
			input:     cs,
			dc:        dc1,
			wantError: true,
			want:      "Error: limit was -1 but must be a positive number\n",
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

func TestTaskLogInteractive_v1beta1(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	clock := test.FakeClock()
	taskCreated := clock.Now()

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		Tasks: []*v1beta1.Task{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "task",
					Namespace:         "ns",
					CreationTimestamp: metav1.Time{Time: taskCreated},
				},
			},
		},
		TaskRuns: []*v1beta1.TaskRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "taskrun1",
					Namespace: "ns",
					Labels:    map[string]string{"tekton.dev/task": "task"},
				},
				Spec: v1beta1.TaskRunSpec{
					TaskRef: &v1beta1.TaskRef{
						Name: "task",
						Kind: v1beta1.NamespacedTaskKind,
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
						StartTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
						PodName:   "pod",
						Steps: []v1beta1.StepState{
							{
								Name: "step1",
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
					Name:      "taskrun2",
					Namespace: "ns",
					Labels:    map[string]string{"tekton.dev/task": "task"},
				},
				Spec: v1beta1.TaskRunSpec{
					TaskRef: &v1beta1.TaskRef{
						Name: "task",
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
						StartTime: &metav1.Time{Time: clock.Now().Add(-3 * time.Minute)},
						PodName:   "pod",
						Steps: []v1beta1.StepState{
							{
								Name: "step2",
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
		},
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
					Name:      "pod",
					Namespace: "ns",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "step1",
							Image: "step1:latest",
						},
						{
							Name:  "step2",
							Image: "step2:latest",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
				},
			},
		},
	})

	cs2, _ := test.SeedV1beta1TestData(t, test.Data{
		Tasks: []*v1beta1.Task{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "task",
					Namespace:         "ns",
					CreationTimestamp: metav1.Time{Time: taskCreated},
				},
			},
		},
		TaskRuns: []*v1beta1.TaskRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "taskrun1",
					Namespace: "ns",
					Labels:    map[string]string{"tekton.dev/task": "task"},
				},
				Spec: v1beta1.TaskRunSpec{
					TaskRef: &v1beta1.TaskRef{
						Name: "task",
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
						StartTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
						PodName:   "pod",
						Steps: []v1beta1.StepState{
							{
								Name: "step1",
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
		},
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
					Name:      "pod",
					Namespace: "ns",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "step1",
							Image: "step1:latest",
						},
						{
							Name:  "step2",
							Image: "step2:latest",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
				},
			},
		},
	})

	logs := fake.Logs(
		fake.Task("pod",
			fake.Step("step1", "step1 log"),
			fake.Step("step2", "step2 log"),
		),
	)

	testParams := []struct {
		name      string
		limit     int
		last      bool
		namespace string
		input     test.Clients
		prompt    prompt.Prompt
	}{
		{
			name:      "Get all input",
			limit:     5,
			last:      false,
			namespace: "ns",
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select task:"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("task"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Select taskrun:"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("taskrun2 started 3 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("taskrun1 started 5 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowUp)); err != nil {
						return err
					}

					if _, err := c.ExpectString("taskrun2 started 3 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Specify task name and choice taskrun name from interactive menu",
			limit:     5,
			last:      false,
			namespace: "ns",
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{"task"},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select taskrun:"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("taskrun2 started 3 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("taskrun1 started 5 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Specify task name and limit as 2",
			limit:     2,
			last:      false,
			namespace: "ns",
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{"task"},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select taskrun:"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("taskrun2 started 3 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("taskrun1 started 5 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Specify task name and limit as 1",
			limit:     1,
			last:      false,
			namespace: "ns",
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{"task"},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("step1 log\r\n"); err != nil {
						return err
					}

					if _, err := c.ExpectString("step2 log\r\n"); err != nil {
						return err
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Specify last flag as true",
			limit:     5,
			last:      true,
			namespace: "ns",
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select task:"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("task"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Specify last flag as true and task name",
			limit:     5,
			last:      true,
			namespace: "ns",
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{"task"},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Specify task name when taskrun is single",
			limit:     5,
			last:      false,
			namespace: "ns",
			input:     cs2,
			prompt: prompt.Prompt{
				CmdArgs: []string{"task"},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					return nil
				},
			},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := test.Params{
				Kube:   tp.input.Kube,
				Tekton: tp.input.Pipeline,
			}
			p.SetNamespace(tp.namespace)

			opts := &options.LogOptions{
				Limit:    tp.limit,
				Last:     tp.last,
				Params:   &p,
				Streamer: fake.Streamer(logs),
			}

			tp.prompt.RunTest(t, tp.prompt.Procedure, func(stdio terminal.Stdio) error {
				opts.AskOpts = prompt.WithStdio(stdio)
				opts.Stream = &cli.Stream{Out: stdio.Out, Err: stdio.Err}

				return run(opts, tp.prompt.CmdArgs)
			})
		})
	}
}

func TestTaskLogInteractive(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	clock := test.FakeClock()
	taskCreated := clock.Now()

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks: []*v1.Task{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "task",
					Namespace:         "ns",
					CreationTimestamp: metav1.Time{Time: taskCreated},
				},
			},
		},
		TaskRuns: []*v1.TaskRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "taskrun1",
					Namespace: "ns",
					Labels:    map[string]string{"tekton.dev/task": "task"},
				},
				Spec: v1.TaskRunSpec{
					TaskRef: &v1.TaskRef{
						Name: "task",
						Kind: v1.NamespacedTaskKind,
					},
				},
				Status: v1.TaskRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Status: corev1.ConditionTrue,
								Reason: v1beta1.TaskRunReasonSuccessful.String(),
							},
						},
					},
					TaskRunStatusFields: v1.TaskRunStatusFields{
						StartTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
						PodName:   "pod",
						Steps: []v1.StepState{
							{
								Name: "step1",
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
					Name:      "taskrun2",
					Namespace: "ns",
					Labels:    map[string]string{"tekton.dev/task": "task"},
				},
				Spec: v1.TaskRunSpec{
					TaskRef: &v1.TaskRef{
						Name: "task",
					},
				},
				Status: v1.TaskRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Status: corev1.ConditionTrue,
								Reason: v1beta1.TaskRunReasonSuccessful.String(),
							},
						},
					},
					TaskRunStatusFields: v1.TaskRunStatusFields{
						StartTime: &metav1.Time{Time: clock.Now().Add(-3 * time.Minute)},
						PodName:   "pod",
						Steps: []v1.StepState{
							{
								Name: "step2",
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
		},
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
					Name:      "pod",
					Namespace: "ns",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "step1",
							Image: "step1:latest",
						},
						{
							Name:  "step2",
							Image: "step2:latest",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
				},
			},
		},
	})

	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks: []*v1.Task{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "task",
					Namespace:         "ns",
					CreationTimestamp: metav1.Time{Time: taskCreated},
				},
			},
		},
		TaskRuns: []*v1.TaskRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "taskrun1",
					Namespace: "ns",
					Labels:    map[string]string{"tekton.dev/task": "task"},
				},
				Spec: v1.TaskRunSpec{
					TaskRef: &v1.TaskRef{
						Name: "task",
					},
				},
				Status: v1.TaskRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Status: corev1.ConditionTrue,
								Reason: v1beta1.TaskRunReasonSuccessful.String(),
							},
						},
					},
					TaskRunStatusFields: v1.TaskRunStatusFields{
						StartTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
						PodName:   "pod",
						Steps: []v1.StepState{
							{
								Name: "step1",
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
		},
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
					Name:      "pod",
					Namespace: "ns",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "step1",
							Image: "step1:latest",
						},
						{
							Name:  "step2",
							Image: "step2:latest",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
				},
			},
		},
	})

	logs := fake.Logs(
		fake.Task("pod",
			fake.Step("step1", "step1 log"),
			fake.Step("step2", "step2 log"),
		),
	)

	testParams := []struct {
		name      string
		limit     int
		last      bool
		namespace string
		input     pipelinetest.Clients
		prompt    prompt.Prompt
	}{
		{
			name:      "Get all input",
			limit:     5,
			last:      false,
			namespace: "ns",
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select task:"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("task"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Select taskrun:"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("taskrun2 started 3 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("taskrun1 started 5 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowUp)); err != nil {
						return err
					}

					if _, err := c.ExpectString("taskrun2 started 3 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Specify task name and choice taskrun name from interactive menu",
			limit:     5,
			last:      false,
			namespace: "ns",
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{"task"},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select taskrun:"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("taskrun2 started 3 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("taskrun1 started 5 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Specify task name and limit as 2",
			limit:     2,
			last:      false,
			namespace: "ns",
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{"task"},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select taskrun:"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("taskrun2 started 3 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("taskrun1 started 5 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Specify task name and limit as 1",
			limit:     1,
			last:      false,
			namespace: "ns",
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{"task"},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("step1 log\r\n"); err != nil {
						return err
					}

					if _, err := c.ExpectString("step2 log\r\n"); err != nil {
						return err
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Specify last flag as true",
			limit:     5,
			last:      true,
			namespace: "ns",
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select task:"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("task"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Specify last flag as true and task name",
			limit:     5,
			last:      true,
			namespace: "ns",
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{"task"},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Specify task name when taskrun is single",
			limit:     5,
			last:      false,
			namespace: "ns",
			input:     cs2,
			prompt: prompt.Prompt{
				CmdArgs: []string{"task"},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					return nil
				},
			},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := test.Params{
				Kube:   tp.input.Kube,
				Tekton: tp.input.Pipeline,
			}
			p.SetNamespace(tp.namespace)

			opts := &options.LogOptions{
				Limit:    tp.limit,
				Last:     tp.last,
				Params:   &p,
				Streamer: fake.Streamer(logs),
			}

			tp.prompt.RunTest(t, tp.prompt.Procedure, func(stdio terminal.Stdio) error {
				opts.AskOpts = prompt.WithStdio(stdio)
				opts.Stream = &cli.Stream{Out: stdio.Out, Err: stdio.Err}

				return run(opts, tp.prompt.CmdArgs)
			})
		})
	}
}

func TestLogs_Auto_Select_FirstTask_v1beta1(t *testing.T) {
	taskName := "dummyTask"
	ns := "dummyNamespaces"
	clock := test.FakeClock()

	tdata := []*v1beta1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      taskName,
				Namespace: ns,
			},
		},
	}

	trdata := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      "dummyTR",
				Labels:    map[string]string{"tekton.dev/task": taskName},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "task",
					Kind: v1beta1.NamespacedTaskKind,
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
		Tasks:    tdata,
		TaskRuns: trdata,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1T(tdata[0], versionv1beta1),
		cb.UnstructuredV1beta1TR(trdata[0], versionv1beta1),
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

	if lopt.TaskName != taskName {
		t.Error("No auto selection of the first task when we have only one")
	}
}

func TestLogs_Auto_Select_FirstTask(t *testing.T) {
	taskName := "dummyTask"
	ns := "dummyNamespaces"
	clock := test.FakeClock()

	tdata := []*v1.Task{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      taskName,
				Namespace: ns,
			},
		},
	}

	trdata := []*v1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      "dummyTR",
				Labels:    map[string]string{"tekton.dev/task": taskName},
			},
			Spec: v1.TaskRunSpec{
				TaskRef: &v1.TaskRef{
					Name: "task",
					Kind: v1.NamespacedTaskKind,
				},
			},
			Status: v1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
				TaskRunStatusFields: v1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
				},
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Tasks:    tdata,
		TaskRuns: trdata,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task", "taskrun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredT(tdata[0], version),
		cb.UnstructuredTR(trdata[0], version),
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

	if lopt.TaskName != taskName {
		t.Error("No auto selection of the first task when we have only one")
	}
}
