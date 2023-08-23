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
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/options"
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

var (
	pipelineName = "output-pipeline"
	prName       = "output-pipeline-run"
	prName2      = "output-pipeline-run-2"
	ns           = "namespace"
)

const (
	versionv1beta1 = "v1beta1"
	version        = "v1"
)

func TestPipelineLog_v1beta1(t *testing.T) {
	clock := test.FakeClock()
	pdata := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		Pipelines: pdata,
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
		cb.UnstructuredV1beta1P(pdata[0], versionv1beta1))
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs2, _ := test.SeedV1beta1TestData(t, test.Data{
		Pipelines: pdata,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})
	cs2.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1P(pdata[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	pdata3 := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              pipelineName,
				Namespace:         ns,
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-15 * time.Minute)},
			},
		},
	}
	prdata3 := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              prName,
				Namespace:         ns,
				Labels:            map[string]string{"tekton.dev/pipeline": pipelineName},
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
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
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					// pipeline run started 5 minutes ago
					StartTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
					// takes 10 minutes to complete
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              prName2,
				Namespace:         ns,
				Labels:            map[string]string{"tekton.dev/pipeline": pipelineName},
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-8 * time.Minute)},
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
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					// pipeline run started 3 minutes ago
					StartTime: &metav1.Time{Time: clock.Now().Add(-3 * time.Minute)},
					// takes 10 minutes to complete
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
				},
			},
		},
	}

	cs3, _ := test.SeedV1beta1TestData(t, test.Data{
		Pipelines:    pdata3,
		PipelineRuns: prdata3,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})
	cs3.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc3 := testDynamic.Options{}
	dc3, err := tdc3.Client(
		cb.UnstructuredV1beta1P(pdata3[0], versionv1beta1),
		cb.UnstructuredV1beta1PR(prdata3[0], versionv1beta1),
		cb.UnstructuredV1beta1PR(prdata3[1], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	testParams := []struct {
		name      string
		command   []string
		namespace string
		dynamic   dynamic.Interface
		input     test.Clients
		wantError bool
		want      string
	}{
		{
			name:      "Invalid namespace",
			command:   []string{"logs", "-n", "invalid"},
			namespace: "",
			dynamic:   dc,
			input:     cs,
			wantError: false,
			want:      "No Pipelines found in namespace invalid",
		},
		{
			name:      "Found no pipelines",
			command:   []string{"logs", "-n", "ns"},
			namespace: "",
			dynamic:   dc,
			input:     cs,
			wantError: false,
			want:      "No Pipelines found in namespace ns",
		},
		{
			name:      "Found no pipelineruns",
			command:   []string{"logs", pipelineName, "-n", ns},
			namespace: "",
			dynamic:   dc2,
			input:     cs2,
			wantError: false,
			want:      "No PipelineRuns found for Pipeline output-pipeline",
		},
		{
			name:      "Pipeline does not exist",
			command:   []string{"logs", "pipeline", "-n", ns},
			namespace: "",
			dynamic:   dc2,
			input:     cs2,
			wantError: true,
			want:      "pipelines.tekton.dev \"pipeline\" not found",
		},
		{
			name:      "Pipelinerun does not exist",
			command:   []string{"logs", "pipeline", "pipelinerun", "-n", "ns"},
			namespace: "ns",
			dynamic:   dc,
			input:     cs,
			wantError: true,
			want:      "pipelineruns.tekton.dev \"pipelinerun\" not found",
		},
		{
			name:      "Invalid limit number",
			command:   []string{"logs", pipelineName, "-n", ns, "--limit", "-1"},
			namespace: "",
			dynamic:   dc2,
			input:     cs2,
			wantError: true,
			want:      "limit was -1 but must be a positive number",
		},
		{
			name:      "Specify last flag",
			command:   []string{"logs", pipelineName, "-n", ns, "-L"},
			namespace: "default",
			dynamic:   dc3,
			input:     cs3,
			wantError: false,
			want:      "",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Clock: clock, Kube: tp.input.Kube, Dynamic: tp.dynamic}
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

func TestPipelineLog(t *testing.T) {
	clock := test.FakeClock()
	pdata := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
		},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: pdata,
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
		cb.UnstructuredP(pdata[0], version))
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: pdata,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})
	cs2.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredP(pdata[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	pdata3 := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              pipelineName,
				Namespace:         ns,
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-15 * time.Minute)},
			},
		},
	}
	prdata3 := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              prName,
				Namespace:         ns,
				Labels:            map[string]string{"tekton.dev/pipeline": pipelineName},
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
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
							Reason: v1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					// pipeline run started 5 minutes ago
					StartTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
					// takes 10 minutes to complete
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              prName2,
				Namespace:         ns,
				Labels:            map[string]string{"tekton.dev/pipeline": pipelineName},
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-8 * time.Minute)},
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
							Reason: v1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					// pipeline run started 3 minutes ago
					StartTime: &metav1.Time{Time: clock.Now().Add(-3 * time.Minute)},
					// takes 10 minutes to complete
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
				},
			},
		},
	}

	cs3, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines:    pdata3,
		PipelineRuns: prdata3,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})
	cs3.Pipeline.Resources = cb.APIResourceList(version, []string{"pipeline", "pipelinerun"})
	tdc3 := testDynamic.Options{}
	dc3, err := tdc3.Client(
		cb.UnstructuredP(pdata3[0], version),
		cb.UnstructuredPR(prdata3[0], version),
		cb.UnstructuredPR(prdata3[1], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	testParams := []struct {
		name      string
		command   []string
		namespace string
		dynamic   dynamic.Interface
		input     pipelinetest.Clients
		wantError bool
		want      string
	}{
		{
			name:      "Invalid namespace",
			command:   []string{"logs", "-n", "invalid"},
			namespace: "",
			dynamic:   dc,
			input:     cs,
			wantError: false,
			want:      "No Pipelines found in namespace invalid",
		},
		{
			name:      "Found no pipelines",
			command:   []string{"logs", "-n", "ns"},
			namespace: "",
			dynamic:   dc,
			input:     cs,
			wantError: false,
			want:      "No Pipelines found in namespace ns",
		},
		{
			name:      "Found no pipelineruns",
			command:   []string{"logs", pipelineName, "-n", ns},
			namespace: "",
			dynamic:   dc2,
			input:     cs2,
			wantError: false,
			want:      "No PipelineRuns found for Pipeline output-pipeline",
		},
		{
			name:      "Pipeline does not exist",
			command:   []string{"logs", "pipeline", "-n", ns},
			namespace: "",
			dynamic:   dc2,
			input:     cs2,
			wantError: true,
			want:      "pipelines.tekton.dev \"pipeline\" not found",
		},
		{
			name:      "Pipelinerun does not exist",
			command:   []string{"logs", "pipeline", "pipelinerun", "-n", "ns"},
			namespace: "ns",
			dynamic:   dc,
			input:     cs,
			wantError: true,
			want:      "pipelineruns.tekton.dev \"pipelinerun\" not found",
		},
		{
			name:      "Invalid limit number",
			command:   []string{"logs", pipelineName, "-n", ns, "--limit", "-1"},
			namespace: "",
			dynamic:   dc2,
			input:     cs2,
			wantError: true,
			want:      "limit was -1 but must be a positive number",
		},
		{
			name:      "Specify last flag",
			command:   []string{"logs", pipelineName, "-n", ns, "-L"},
			namespace: "default",
			dynamic:   dc3,
			input:     cs3,
			wantError: false,
			want:      "",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Clock: clock, Kube: tp.input.Kube, Dynamic: tp.dynamic}
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

func TestPipelineLog_Interactive_v1beta1(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	clock := test.FakeClock()

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		Pipelines: []*v1beta1.Pipeline{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pipelineName,
					Namespace: ns,
					// created  15 minutes back
					CreationTimestamp: metav1.Time{Time: clock.Now().Add(-15 * time.Minute)},
				},
			},
		},
		PipelineRuns: []*v1beta1.PipelineRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              prName,
					Namespace:         ns,
					Labels:            map[string]string{"tekton.dev/pipeline": pipelineName},
					CreationTimestamp: metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
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
								Reason: v1beta1.PipelineRunReasonSuccessful.String(),
							},
						},
					},
					PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
						// pipeline run started 5 minutes ago
						StartTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
						// takes 10 minutes to complete
						CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              prName2,
					Namespace:         ns,
					Labels:            map[string]string{"tekton.dev/pipeline": pipelineName},
					CreationTimestamp: metav1.Time{Time: clock.Now().Add(-8 * time.Minute)},
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
								Reason: v1beta1.PipelineRunReasonSuccessful.String(),
							},
						},
					},
					PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
						// pipeline run started 3 minutes ago
						StartTime: &metav1.Time{Time: clock.Now().Add(-3 * time.Minute)},
						// takes 10 minutes to complete
						CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					},
				},
			},
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			},
		},
	})

	cs2, _ := test.SeedV1beta1TestData(t, test.Data{
		Pipelines: []*v1beta1.Pipeline{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pipelineName,
					Namespace: ns,
					// created  15 minutes back
					CreationTimestamp: metav1.Time{Time: clock.Now().Add(-15 * time.Minute)},
				},
			},
		},
		PipelineRuns: []*v1beta1.PipelineRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              prName,
					Namespace:         ns,
					Labels:            map[string]string{"tekton.dev/pipeline": pipelineName},
					CreationTimestamp: metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
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
								Reason: v1beta1.PipelineRunReasonSuccessful.String(),
							},
						},
					},
					PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
						// pipeline run started 5 minutes ago
						StartTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
						// takes 10 minutes to complete
						CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					},
				},
			},
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
		input     test.Clients
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

func TestPipelineLog_Interactive(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	clock := test.FakeClock()

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1.Pipeline{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pipelineName,
					Namespace: ns,
					// created  15 minutes back
					CreationTimestamp: metav1.Time{Time: clock.Now().Add(-15 * time.Minute)},
				},
			},
		},
		PipelineRuns: []*v1.PipelineRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              prName,
					Namespace:         ns,
					Labels:            map[string]string{"tekton.dev/pipeline": pipelineName},
					CreationTimestamp: metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
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
								Reason: v1beta1.PipelineRunReasonSuccessful.String(),
							},
						},
					},
					PipelineRunStatusFields: v1.PipelineRunStatusFields{
						// pipeline run started 5 minutes ago
						StartTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
						// takes 10 minutes to complete
						CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              prName2,
					Namespace:         ns,
					Labels:            map[string]string{"tekton.dev/pipeline": pipelineName},
					CreationTimestamp: metav1.Time{Time: clock.Now().Add(-8 * time.Minute)},
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
								Reason: v1beta1.PipelineRunReasonSuccessful.String(),
							},
						},
					},
					PipelineRunStatusFields: v1.PipelineRunStatusFields{
						// pipeline run started 3 minutes ago
						StartTime: &metav1.Time{Time: clock.Now().Add(-3 * time.Minute)},
						// takes 10 minutes to complete
						CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					},
				},
			},
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
		Pipelines: []*v1.Pipeline{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pipelineName,
					Namespace: ns,
					// created  15 minutes back
					CreationTimestamp: metav1.Time{Time: clock.Now().Add(-15 * time.Minute)},
				},
			},
		},
		PipelineRuns: []*v1.PipelineRun{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              prName,
					Namespace:         ns,
					Labels:            map[string]string{"tekton.dev/pipeline": pipelineName},
					CreationTimestamp: metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
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
								Reason: v1.PipelineRunReasonSuccessful.String(),
							},
						},
					},
					PipelineRunStatusFields: v1.PipelineRunStatusFields{
						// pipeline run started 5 minutes ago
						StartTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
						// takes 10 minutes to complete
						CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
					},
				},
			},
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
	clock := test.FakeClock()

	pdata := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: ns,
			},
		},
	}

	prdata := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              prName,
				Namespace:         ns,
				Labels:            map[string]string{"tekton.dev/pipeline": pipelineName},
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-10 * time.Minute)},
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
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					// pipeline run started 5 minutes ago
					StartTime: &metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
					// takes 10 minutes to complete
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
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
					Name: ns,
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
	p := test.Params{
		Kube:    cs.Kube,
		Tekton:  cs.Pipeline,
		Dynamic: dc,
	}
	p.SetNamespace(ns)

	lopt := &options.LogOptions{
		Follow: false,
		Limit:  5,
		Params: &p,
	}
	err = getAllInputs(lopt)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if lopt.PipelineName != pipelineName {
		t.Error("No auto selection of the first pipeline when we have only one")
	}
}
