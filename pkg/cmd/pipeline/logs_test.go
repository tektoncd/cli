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
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
	goexpect "github.com/Netflix/go-expect"
	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/helper/options"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	"github.com/tektoncd/cli/test/prompt"
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

func TestPipelineLog(t *testing.T) {
	clock := clockwork.NewFakeClock()

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

	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
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

	cs3, _ := test.SeedTestData(t, pipelinetest.Data{
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

	testParams := []struct {
		name      string
		command   []string
		namespace string
		input     pipelinetest.Clients
		wantError bool
		want      string
	}{
		{
			name:      "Invalid namespace",
			command:   []string{"logs", "-n", "invalid"},
			namespace: "",
			input:     cs,
			wantError: true,
			want:      "namespaces \"invalid\" not found",
		},
		{
			name:      "Found no pipelines",
			command:   []string{"logs", "-n", "ns"},
			namespace: "",
			input:     cs,
			wantError: false,
			want:      "No pipelines found in namespace: ns\n",
		},
		{
			name:      "Found no pipelineruns",
			command:   []string{"logs", pipelineName, "-n", ns},
			namespace: "",
			input:     cs2,
			wantError: false,
			want:      "No pipelineruns found for pipeline: output-pipeline\n",
		},
		{
			name:      "Pipeline does not exist",
			command:   []string{"logs", "pipeline", "-n", ns},
			namespace: "",
			input:     cs2,
			wantError: true,
			want:      "pipelines.tekton.dev \"pipeline\" not found",
		},
		{
			name:      "Pipelinerun does not exist",
			command:   []string{"logs", "pipeline", "pipelinerun", "-n", "ns"},
			namespace: "ns",
			input:     cs,
			wantError: true,
			want:      "pipelineruns.tekton.dev \"pipelinerun\" not found",
		},
		{
			name:      "Invalid limit number",
			command:   []string{"logs", pipelineName, "-n", ns, "--limit", "-1"},
			namespace: "",
			input:     cs2,
			wantError: true,
			want:      "limit was -1 but must be a positive number",
		},
		{
			name:      "Specify last flag",
			command:   []string{"logs", pipelineName, "-n", ns, "-L"},
			namespace: "default",
			input:     cs3,
			wantError: false,
			want:      "",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Clock: clock, Kube: tp.input.Kube}
			if tp.namespace != "" {
				p.SetNamespace(tp.namespace)
			}
			c := Command(p)

			out, err := test.ExecuteCommand(c, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}

func TestPipelineLog_Interactive(t *testing.T) {
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

	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
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

	testParams := []struct {
		name      string
		limit     int
		last      bool
		namespace string
		input     pipelinetest.Clients
		prompt    prompt.Prompt
	}{
		{
			name:      "Select pipeline output-pipeline and output-pipeline-run-2 logs",
			limit:     5,
			last:      false,
			namespace: ns,
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select pipeline:"); err != nil {
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

					if _, err := c.ExpectString("Select pipelinerun:"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("output-pipeline-run-2 started 3 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("output-pipeline-run started 5 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowUp)); err != nil {
						return err
					}

					if _, err := c.ExpectString("output-pipeline-run-2 started 3 minutes ago"); err != nil {
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
			name:      "Pipeline included with command and select pipelinerun output-pipeline-run",
			limit:     5,
			last:      false,
			namespace: ns,
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{pipelineName},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select pipelinerun:"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("output-pipeline-run-2 started 3 minutes ago"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("output-pipeline-run started 5 minutes ago"); err != nil {
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
			name:      "Set pipelinerun limit=2 and select pipelinerun output-pipeline-run",
			limit:     2,
			last:      false,
			namespace: ns,
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{pipelineName},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select pipelinerun:"); err != nil {
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
		},
		{
			name:      "Set pipelinerun limit=1 and select pipelinerun output-pipeline-run-2",
			limit:     1,
			last:      false,
			namespace: ns,
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{pipelineName},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Select pipeline output-pipeline with last=true and no pipelineruns",
			limit:     5,
			last:      true,
			namespace: ns,
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select pipeline:"); err != nil {
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

					if _, err := c.ExpectString("Select pipelinerun:"); err == nil {
						return errors.New("unexpected error")
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Include pipeline output-pipeline as part of command with last=true",
			limit:     5,
			last:      true,
			namespace: ns,
			input:     cs,
			prompt: prompt.Prompt{
				CmdArgs: []string{pipelineName},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("output-pipeline"); err == nil {
						return errors.New("unexpected error")
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					return nil
				},
			},
		},
		{
			name:      "Select pipeline output-pipeline",
			limit:     5,
			last:      false,
			namespace: ns,
			input:     cs2,
			prompt: prompt.Prompt{
				CmdArgs: []string{pipelineName},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("output-pipeline"); err == nil {
						return errors.New("unexpected error")
					}

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
				PipelineRunName: prName,
				Limit:           tp.limit,
				Last:            tp.last,
				Params:          &p,
			}

			tp.prompt.RunTest(t, tp.prompt.Procedure, func(stdio terminal.Stdio) error {
				opts.AskOpts = prompt.WithStdio(stdio)
				opts.Stream = &cli.Stream{Out: stdio.Out, Err: stdio.Err}

				return run(opts, tp.prompt.CmdArgs)
			})
		})
	}
}

func TestLogs_Auto_Select_FirstPipeline(t *testing.T) {
	pipelineName := "blahblah"
	ns := "chouchou"
	clock := clockwork.NewFakeClock()

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline(pipelineName, ns),
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

	p := test.Params{
		Kube:   cs.Kube,
		Tekton: cs.Pipeline,
	}
	p.SetNamespace(ns)

	lopt := &options.LogOptions{
		Follow: false,
		Limit:  5,
		Params: &p,
	}
	err := getAllInputs(lopt)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if lopt.PipelineName != pipelineName {
		t.Error("No auto selection of the first pipeline when we have only one")
	}
}
