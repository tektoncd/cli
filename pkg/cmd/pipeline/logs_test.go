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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
	goexpect "github.com/Netflix/go-expect"
	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/helper/options"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinetest "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

var (
	pipelineName = "output-pipeline"
	prName       = "output-pipeline-run"
	prName2      = "output-pipeline-run-2"
	ns           = "namespace"
)

func TestLogs_invalid_namespace(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}
	c := Command(p)
	out, err := test.ExecuteCommand(c, "logs", "-n", "invalid")
	if err == nil {
		t.Errorf("Expected error for invalid namespace")
	}

	expected := "Error: namespaces \"invalid\" not found\n"
	test.AssertOutput(t, expected, out)
}

func TestLogs_no_pipeline(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	c := Command(p)
	out, err := test.ExecuteCommand(c, "logs", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := "No pipelines found in namespace: ns\n"
	test.AssertOutput(t, expected, out)
}

func TestLogs_no_runs(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	c := Command(p)
	out, err := test.ExecuteCommand(c, "logs", pipelineName, "-n", ns)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := "No pipelineruns found for pipeline: output-pipeline\n"
	test.AssertOutput(t, expected, out)
}

func TestLogs_wrong_pipeline(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	c := Command(p)
	_, err := test.ExecuteCommand(c, "logs", "pipeline", "-n", ns)

	expected := "pipelines.tekton.dev \"pipeline\" not found"
	test.AssertOutput(t, expected, err.Error())
}

func TestLogs_wrong_run(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	c := Command(p)
	_, err := test.ExecuteCommand(c, "logs", "pipeline", "pipelinerun", "-n", "ns")

	expected := "pipelineruns.tekton.dev \"pipelinerun\" not found"
	test.AssertOutput(t, expected, err.Error())
}

func TestLogs_negative_limit(t *testing.T) {
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}

	c := Command(p)
	_, err := test.ExecuteCommand(c, "logs", pipelineName, "-n", ns, "--limit", fmt.Sprintf("%d", -1))

	expected := "limit was -1 but must be a positive number"
	test.AssertOutput(t, expected, err.Error())
}

func TestLogs_interactive_get_all_inputs(t *testing.T) {

	clock := clockwork.NewFakeClock()

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns,
				// created  15 minutes back
				cb.PipelineCreationTimestamp(clock.Now().Add(-15*time.Minute)),
			),
		},
		PipelineRuns: []*v1alpha1.PipelineRun{

			tb.PipelineRun(prName, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-10*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 5 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-5*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
			tb.PipelineRun(prName2, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-8*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 3 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-3*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})

	tests := []promptTest{
		{
			name:    "basic interaction",
			cmdArgs: []string{},

			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Select pipeline :"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("output-pipeline"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select pipelinerun :"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString(prName2 + " started 3 minutes ago"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString(prName + " started 2 minutes ago"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyArrowUp)); err != nil {
					return err
				}

				if _, err := c.ExpectString(prName2 + " started 3 minutes ago"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
					return err
				}

				return nil
			},
		},
	}
	opts := logOpts(prName, ns, 5, false, cs)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RunPromptTest(t, test, opts)
		})
	}
}

func TestLogs_interactive_ask_runs(t *testing.T) {
	clock := clockwork.NewFakeClock()

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns,
				// created  15 minutes back
				cb.PipelineCreationTimestamp(clock.Now().Add(-15*time.Minute)),
			),
		},
		PipelineRuns: []*v1alpha1.PipelineRun{

			tb.PipelineRun(prName, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-10*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 5 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-5*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
			tb.PipelineRun(prName2, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-8*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 3 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-3*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})

	tests := []promptTest{
		{
			name:    "basic interaction",
			cmdArgs: []string{pipelineName},

			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Select pipelinerun :"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString(prName2 + " started 3 minutes ago"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString(prName + "started 5 minutes ago"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
					return err
				}

				return nil
			},
		},
	}
	opts := logOpts(prName, ns, 5, false, cs)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RunPromptTest(t, test, opts)
		})
	}
}

func TestLogs_interactive_limit_2(t *testing.T) {
	clock := clockwork.NewFakeClock()

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns,
				// created  15 minutes back
				cb.PipelineCreationTimestamp(clock.Now().Add(-15*time.Minute)),
			),
		},
		PipelineRuns: []*v1alpha1.PipelineRun{

			tb.PipelineRun(prName, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-10*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 5 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-5*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
			tb.PipelineRun(prName2, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-8*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 3 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-3*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})

	tests := []promptTest{
		{
			name:    "basic interaction",
			cmdArgs: []string{pipelineName},

			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("output-pipeline"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select pipelinerun :"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString(prName2 + " started 3 minutes ago"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString(prName + " started 5 minutes ago"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
					return err
				}

				return nil
			},
		},
	}
	opts := logOpts(prName, ns, 2, false, cs)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RunPromptTest(t, test, opts)
		})
	}
}

func TestLogs_interactive_limit_1(t *testing.T) {
	clock := clockwork.NewFakeClock()

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns,
				// created  15 minutes back
				cb.PipelineCreationTimestamp(clock.Now().Add(-15*time.Minute)),
			),
		},
		PipelineRuns: []*v1alpha1.PipelineRun{

			tb.PipelineRun(prName, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-10*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 5 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-5*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
			tb.PipelineRun(prName2, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-8*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 3 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-3*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})

	tests := []promptTest{
		{
			name:    "basic interaction",
			cmdArgs: []string{pipelineName},

			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("output-pipeline"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select pipelinerun :"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString(prName2 + " started 3 minutes ago"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
					return err
				}

				return nil
			},
		},
	}
	opts := logOpts(prName, ns, 1, false, cs)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RunPromptTest(t, test, opts)
		})
	}
}

func TestLogs_interactive_ask_all_last_run(t *testing.T) {
	clock := clockwork.NewFakeClock()

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns,
				// created  15 minutes back
				cb.PipelineCreationTimestamp(clock.Now().Add(-15*time.Minute)),
			),
		},
		PipelineRuns: []*v1alpha1.PipelineRun{

			tb.PipelineRun(prName, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-10*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 5 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-5*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
			tb.PipelineRun(prName2, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-8*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 3 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-3*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})

	tests := []promptTest{
		{
			name:    "basic interaction",
			cmdArgs: []string{},

			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Select pipeline :"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("output-pipeline"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select pipelinerun :"); err == nil {
					return errors.New("unexpected error")
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				return nil
			},
		},
	}
	opts := logOpts(prName, ns, 5, true, cs)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RunPromptTest(t, test, opts)
		})
	}
}

func TestLogs_interactive_ask_run_last_run(t *testing.T) {
	clock := clockwork.NewFakeClock()

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns,
				// created  15 minutes back
				cb.PipelineCreationTimestamp(clock.Now().Add(-15*time.Minute)),
			),
		},
		PipelineRuns: []*v1alpha1.PipelineRun{

			tb.PipelineRun(prName, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-10*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 5 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-5*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
			tb.PipelineRun(prName2, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-8*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 3 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-3*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})

	tests := []promptTest{
		{
			name:    "basic interaction",
			cmdArgs: []string{pipelineName},

			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("output-pipeline"); err == nil {
					return errors.New("unexpected error")
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				return nil
			},
		},
	}
	opts := logOpts(prName, ns, 5, true, cs)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RunPromptTest(t, test, opts)
		})
	}
}

func TestLogs_last_run_diff_namespace(t *testing.T) {
	clock := clockwork.NewFakeClock()

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns,
				// created  15 minutes back
				cb.PipelineCreationTimestamp(clock.Now().Add(-15*time.Minute)),
			),
		},
		PipelineRuns: []*v1alpha1.PipelineRun{

			tb.PipelineRun(prName, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-10*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 5 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-5*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
			tb.PipelineRun(prName2, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-8*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 3 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-3*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})

	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube}
	p.SetNamespace("default")

	c := Command(p)
	_, err := test.ExecuteCommand(c, "logs", pipelineName, "-n", ns, "-L")
	if err != nil {
		t.Error("Expecting no error")
	}
}

func TestLogs_have_one_get_one(t *testing.T) {
	clock := clockwork.NewFakeClock()

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns,
				// created  15 minutes back
				cb.PipelineCreationTimestamp(clock.Now().Add(-15*time.Minute)),
			),
		},
		PipelineRuns: []*v1alpha1.PipelineRun{

			tb.PipelineRun(prName, ns,
				cb.PipelineRunCreationTimestamp(clock.Now().Add(-10*time.Minute)),
				tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
				tb.PipelineRunSpec(pipelineName),
				tb.PipelineRunStatus(
					tb.PipelineRunStatusCondition(apis.Condition{
						Status: corev1.ConditionTrue,
						Reason: resources.ReasonSucceeded,
					}),
					// pipeline run started 5 minutes ago
					tb.PipelineRunStartTime(clock.Now().Add(-5*time.Minute)),
					// takes 10 minutes to complete
					cb.PipelineRunCompletionTime(clock.Now().Add(10*time.Minute)),
				),
			),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})

	tests := []promptTest{
		{
			name:    "basic interaction",
			cmdArgs: []string{pipelineName},

			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("output-pipeline"); err == nil {
					return errors.New("unexpected error")
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				return nil
			},
		},
	}
	opts := logOpts(prName, ns, 5, false, cs)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RunPromptTest(t, test, opts)
		})
	}
}

func logOpts(name string, ns string, prLimit int, last bool, cs pipelinetest.Clients) *options.LogOptions {
	p := test.Params{
		Kube:   cs.Kube,
		Tekton: cs.Pipeline,
	}
	p.SetNamespace(ns)
	logOp := options.LogOptions{
		PipelineRunName: name,
		Limit:           prLimit,
		Last:            last,
		Params:          &p,
	}

	return &logOp
}
