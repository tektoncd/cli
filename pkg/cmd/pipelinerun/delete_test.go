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
	"io"
	"strings"
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestPipelineRunDelete_v1beta1(t *testing.T) {

	version := "v1beta1"
	clock := test.FakeClock()

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	pdata := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	prdata := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-1",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
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
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					// pipeline run starts now
					StartTime: &metav1.Time{Time: clock.Now()},
					// takes 10 minutes to complete
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-2",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
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
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					// pipeline run starts now
					StartTime: &metav1.Time{Time: clock.Now()},
					// takes 10 minutes to complete
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-3",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
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
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					// pipeline run starts now
					StartTime: &metav1.Time{Time: clock.Now()},
					// takes 10 minutes to complete
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
				},
			},
		},
		{
			// test pipelinerun status condition nil
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-4",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1beta1.PipelineRunStatus{},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-5",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
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
							Reason: v1beta1.PipelineRunReasonPending.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-6",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
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
							Reason: v1beta1.PipelineRunReasonRunning.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					// pipeline run starts now
					StartTime: &metav1.Time{Time: clock.Now()},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-7",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
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
							Status: corev1.ConditionUnknown,
							Reason: v1beta1.PipelineRunReasonPending.String(),
						},
					},
				},
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					// pipeline run starts now
					StartTime: &metav1.Time{Time: clock.Now()},
				},
			},
		},
	}

	type clients struct {
		pipelineClient test.Clients
		dynamicClient  dynamic.Interface
	}

	seeds := make([]clients, 0)
	for i := 0; i < 16; i++ {
		cs, _ := test.SeedV1beta1TestData(t, test.Data{
			Pipelines:    pdata,
			PipelineRuns: prdata,
			Namespaces:   ns,
		})
		cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredV1beta1PR(prdata[0], version),
			cb.UnstructuredV1beta1PR(prdata[1], version),
			cb.UnstructuredV1beta1PR(prdata[2], version),
			cb.UnstructuredV1beta1PR(prdata[3], version),
			cb.UnstructuredV1beta1PR(prdata[4], version),
			cb.UnstructuredV1beta1PR(prdata[5], version),
			cb.UnstructuredV1beta1PR(prdata[6], version),
		)
		if err != nil {
			t.Errorf("unable to create dynamic client: %v", err)
		}
		seeds = append(seeds, clients{cs, dc})
	}

	testParams := []struct {
		name        string
		command     []string
		dynamic     dynamic.Interface
		input       test.Clients
		inputStream io.Reader
		wantError   bool
		want        string
	}{
		{
			name:        "Invalid namespace",
			command:     []string{"rm", "pipeline-run-1", "-n", "invalid"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "pipelineruns.tekton.dev \"pipeline-run-1\" not found",
		},
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "pipeline-run-1", "-n", "ns", "-f"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "PipelineRuns deleted: \"pipeline-run-1\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "pipeline-run-1", "-n", "ns", "--force"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "PipelineRuns deleted: \"pipeline-run-1\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "pipeline-run-1", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting PipelineRun(s) \"pipeline-run-1\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "pipeline-run-1", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete PipelineRun(s) \"pipeline-run-1\" (y/n): PipelineRuns deleted: \"pipeline-run-1\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "pipelineruns.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "pipelineruns.tekton.dev \"nonexistent\" not found; pipelineruns.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Attempt remove forgetting to include pipelinerun names",
			command:     []string{"rm", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "must provide PipelineRun name(s) or use --pipeline flag or --all flag to use delete",
		},
		{
			name:        "Remove pipelineruns of a pipeline",
			command:     []string{"rm", "--pipeline", "pipeline", "-n", "ns"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns related to Pipeline \"pipeline\" (y/n): All PipelineRuns(Completed) associated with Pipeline \"pipeline\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns in namespace \"ns\" (y/n): All PipelineRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All PipelineRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all keeping 2",
			command:     []string{"delete", "--all", "-f", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 2 PipelineRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Keep -1 is a no go",
			command:     []string{"delete", "--all", "-f", "--keep", "-1", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "keep option should not be lower than 0",
		},
		{
			name:        "Error from using pipelinerun name with --all",
			command:     []string{"delete", "pipeline-run-1", "--all", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from deleting PipelineRun with non-existing Pipeline",
			command:     []string{"delete", "pipeline-run-1", "-p", "non-existing-pipeline", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "no PipelineRuns associated with Pipeline \"non-existing-pipeline\"",
		},
		{
			name:        "Remove pipelineruns of a pipeline using --keep",
			command:     []string{"rm", "--pipeline", "pipeline", "-n", "ns", "--keep", "2"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns related to Pipeline \"pipeline\" keeping 2 PipelineRuns (y/n): All but 2 PipelineRuns(Completed) associated with Pipeline \"pipeline\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from using argument with --keep",
			command:     []string{"rm", "pipeline-run-1", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--keep flag should not have any arguments specified with it",
		},
		{
			name:        "Delete all PipelineRuns older than 60mn associated with Pipeline pipeline",
			command:     []string{"delete", "-f", "--pipeline", "pipeline", "--keep-since", "60", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 3 expired PipelineRuns(Completed) associated with Pipeline \"pipeline\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all PipelineRuns older than 60mn and keeping 2 PipelineRuns",
			command:     []string{"delete", "-f", "--keep-since", "60", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[11].dynamicClient,
			input:       seeds[11].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "1 PipelineRuns(Completed) has been deleted in namespace \"ns\", kept 2\n",
		},
		{
			name:        "Delete all PipelineRuns older than 60mn and keeping 2 PipelineRuns associated with Pipeline pipeline",
			command:     []string{"delete", "-f", "--pipeline", "pipeline", "--keep-since", "60", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[10].dynamicClient,
			input:       seeds[10].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "1 PipelineRuns(Completed) associated with Pipeline \"pipeline\" has been deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error --keep-since less than zero",
			command:     []string{"delete", "-f", "--keep-since", "-1", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "keep-since option should not be lower than 0",
		},
		{
			name:        "Error --keep-since and --keep less than zero",
			command:     []string{"delete", "-f", "--keep-since", "-1", "--keep", "-1", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "keep and keep-since option should not be lower than 0",
		},
		{
			name:        "Error --keep-since, --all and --task cannot be used",
			command:     []string{"delete", "-f", "--keep-since", "1", "--all", "-p", "foobar", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--keep or --keep-since, --all and --pipeline cannot be used together",
		},
		{
			name:        "Error --keep-since, --keep, --all and --task cannot be used",
			command:     []string{"delete", "-f", "--keep-since", "1", "--keep", "1", "--all", "-p", "foobar", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--keep or --keep-since, --all and --pipeline cannot be used together",
		},
		{
			name:        "Delete all with explicit ignore-running true",
			command:     []string{"rm", "--all", "-n", "ns", "--ignore-running=true"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns in namespace \"ns\" (y/n): All PipelineRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with default ignore-running",
			command:     []string{"rm", "--all", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns in namespace \"ns\" (y/n): All PipelineRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with ignore-running false",
			command:     []string{"rm", "--all", "-n", "ns", "--ignore-running=false"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns in namespace \"ns\" (y/n): All PipelineRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete the PipelineRun present and give error for non-existent PipelineRun",
			command:     []string{"delete", "nonexistent", "pipeline-run-1", "-n", "ns"},
			dynamic:     seeds[7].dynamicClient,
			input:       seeds[7].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "pipelineruns.tekton.dev \"nonexistent\" not found",
		},
		// test pipelinerun delete with status as nil
		{
			name:        "Delete all the pipelineruns with keep-since except one without status conditions",
			command:     []string{"delete", "--keep-since", "1", "-n", "ns", "-f"},
			dynamic:     seeds[8].dynamicClient,
			input:       seeds[8].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "3 expired PipelineRuns(Completed) has been deleted in namespace \"ns\", kept 0\n",
		},
		{
			name:        "Delete all the pipelineruns with keep-since except one without status conditions",
			command:     []string{"delete", "--keep", "1", "-n", "ns", "-f", "--ignore-running=false"},
			dynamic:     seeds[8].dynamicClient,
			input:       seeds[8].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 1 PipelineRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all the pipelineruns with keep-since ignore-running false except one without status conditions",
			command:     []string{"delete", "--keep-since", "1", "-n", "ns", "-f", "--ignore-running=false"},
			dynamic:     seeds[9].dynamicClient,
			input:       seeds[9].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "6 expired PipelineRuns has been deleted in namespace \"ns\", kept 0\n",
		},
		{
			name:        "Delete all the pipelineruns including one without status",
			command:     []string{"delete", "--all", "-n", "ns", "-f", "--ignore-running=false"},
			dynamic:     seeds[9].dynamicClient,
			input:       seeds[9].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All PipelineRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "No pipelineruns after deleting all including running and nil status",
			command:     []string{"ls", "-n", "ns"},
			dynamic:     seeds[9].dynamicClient,
			input:       seeds[9].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "No PipelineRuns found\n",
		},
		{
			name:        "Attempt to delete PieplineRun by keeping more than existing PipelineRun for a Pipeline",
			command:     []string{"delete", "-i", "-f", "--keep", "3", "--pipeline", "pipeline", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "There is/are only 2 PipelineRun(s) associated for Pipeline: pipeline \n",
		},
		{
			name:        "Attempt to delete PieplineRun by keeping more than existing PipelineRun for a Pipeline",
			command:     []string{"delete", "-i", "-f", "--keep", "2", "--pipeline", "pipeline", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Associated PipelineRun (2) for Pipeline:pipeline is/are equal to keep (2) \n",
		},
		{
			name:        "Delete all of a pipeline by default ignore-running",
			command:     []string{"rm", "-p", "pipeline", "-n", "ns"},
			dynamic:     seeds[12].dynamicClient,
			input:       seeds[12].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns related to Pipeline \"pipeline\" (y/n): All PipelineRuns(Completed) associated with Pipeline \"pipeline\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all of a pipeline by explicit ignore-running true",
			command:     []string{"rm", "-p", "pipeline", "-n", "ns", "--ignore-running=true"},
			dynamic:     seeds[13].dynamicClient,
			input:       seeds[13].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns related to Pipeline \"pipeline\" (y/n): All PipelineRuns(Completed) associated with Pipeline \"pipeline\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all of a pipeline by ignore-running false",
			command:     []string{"rm", "-p", "pipeline", "-n", "ns", "--ignore-running=false"},
			dynamic:     seeds[14].dynamicClient,
			input:       seeds[14].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns related to Pipeline \"pipeline\" (y/n): All PipelineRuns associated with Pipeline \"pipeline\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Ignore deleting PipelineRun with Status Unknown",
			command:     []string{"delete", "--all", "-n", "ns", "-f"},
			dynamic:     seeds[15].dynamicClient,
			input:       seeds[15].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All PipelineRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "PipelineRun with Status Unknown exists",
			command:     []string{"ls", "-n", "ns"},
			dynamic:     seeds[15].dynamicClient,
			input:       seeds[15].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "NAME             STARTED         DURATION   STATUS\npipeline-run-4   ---             ---        ---\npipeline-run-5   ---             ---        Pending\npipeline-run-6   0 seconds ago   ---        Succeeded(Running)\npipeline-run-7   0 seconds ago   ---        Pending\n",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube, Dynamic: tp.dynamic}
			pipelinerun := Command(p)

			if tp.inputStream != nil {
				pipelinerun.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(pipelinerun, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				} else {
					test.AssertOutput(t, tp.want, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}

func TestPipelineRunDelete(t *testing.T) {
	version := "v1"
	clock := test.FakeClock()

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	pdata := []*v1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pipeline",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-5 * time.Minute)},
			},
		},
	}

	prdata := []*v1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-1",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
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
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					// pipeline run starts now
					StartTime: &metav1.Time{Time: clock.Now()},
					// takes 10 minutes to complete
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-2",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
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
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					// pipeline run starts now
					StartTime: &metav1.Time{Time: clock.Now()},
					// takes 10 minutes to complete
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-3",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
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
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					// pipeline run starts now
					StartTime: &metav1.Time{Time: clock.Now()},
					// takes 10 minutes to complete
					CompletionTime: &metav1.Time{Time: clock.Now().Add(10 * time.Minute)},
				},
			},
		},
		{
			// test pipelinerun status condition nil
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-4",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
			},
			Spec: v1.PipelineRunSpec{
				PipelineRef: &v1.PipelineRef{
					Name: "pipeline",
				},
			},
			Status: v1.PipelineRunStatus{},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-5",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
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
							Reason: v1.PipelineRunReasonPending.String(),
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-6",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
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
							Reason: v1beta1.PipelineRunReasonRunning.String(),
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					// pipeline run starts now
					StartTime: &metav1.Time{Time: clock.Now()},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "ns",
				Name:              "pipeline-run-7",
				Labels:            map[string]string{"tekton.dev/pipeline": "pipeline"},
				CreationTimestamp: metav1.Time{Time: clock.Now()},
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
							Status: corev1.ConditionUnknown,
							Reason: v1.PipelineRunReasonPending.String(),
						},
					},
				},
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					// pipeline run starts now
					StartTime: &metav1.Time{Time: clock.Now()},
				},
			},
		},
	}

	type clients struct {
		pipelineClient pipelinetest.Clients
		dynamicClient  dynamic.Interface
	}

	seeds := make([]clients, 0)
	for i := 0; i < 16; i++ {
		cs, _ := test.SeedTestData(t, pipelinetest.Data{
			Pipelines:    pdata,
			PipelineRuns: prdata,
			Namespaces:   ns,
		})
		cs.Pipeline.Resources = cb.APIResourceList(version, []string{"pipelinerun"})
		tdc := testDynamic.Options{}
		dc, err := tdc.Client(
			cb.UnstructuredPR(prdata[0], version),
			cb.UnstructuredPR(prdata[1], version),
			cb.UnstructuredPR(prdata[2], version),
			cb.UnstructuredPR(prdata[3], version),
			cb.UnstructuredPR(prdata[4], version),
			cb.UnstructuredPR(prdata[5], version),
			cb.UnstructuredPR(prdata[6], version),
		)
		if err != nil {
			t.Errorf("unable to create dynamic client: %v", err)
		}
		seeds = append(seeds, clients{cs, dc})
	}

	testParams := []struct {
		name        string
		command     []string
		dynamic     dynamic.Interface
		input       pipelinetest.Clients
		inputStream io.Reader
		wantError   bool
		want        string
	}{
		{
			name:        "Invalid namespace",
			command:     []string{"rm", "pipeline-run-1", "-n", "invalid"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "pipelineruns.tekton.dev \"pipeline-run-1\" not found",
		},
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "pipeline-run-1", "-n", "ns", "-f"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "PipelineRuns deleted: \"pipeline-run-1\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "pipeline-run-1", "-n", "ns", "--force"},
			dynamic:     seeds[1].dynamicClient,
			input:       seeds[1].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "PipelineRuns deleted: \"pipeline-run-1\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "pipeline-run-1", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting PipelineRun(s) \"pipeline-run-1\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "pipeline-run-1", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete PipelineRun(s) \"pipeline-run-1\" (y/n): PipelineRuns deleted: \"pipeline-run-1\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "pipelineruns.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "pipelineruns.tekton.dev \"nonexistent\" not found; pipelineruns.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Attempt remove forgetting to include pipelinerun names",
			command:     []string{"rm", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "must provide PipelineRun name(s) or use --pipeline flag or --all flag to use delete",
		},
		{
			name:        "Remove pipelineruns of a pipeline",
			command:     []string{"rm", "--pipeline", "pipeline", "-n", "ns"},
			dynamic:     seeds[0].dynamicClient,
			input:       seeds[0].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns related to Pipeline \"pipeline\" (y/n): All PipelineRuns(Completed) associated with Pipeline \"pipeline\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			dynamic:     seeds[3].dynamicClient,
			input:       seeds[3].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns in namespace \"ns\" (y/n): All PipelineRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All PipelineRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all keeping 2",
			command:     []string{"delete", "--all", "-f", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 2 PipelineRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Keep -1 is a no go",
			command:     []string{"delete", "--all", "-f", "--keep", "-1", "-n", "ns"},
			dynamic:     seeds[4].dynamicClient,
			input:       seeds[4].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "keep option should not be lower than 0",
		},
		{
			name:        "Error from using pipelinerun name with --all",
			command:     []string{"delete", "pipeline-run-1", "--all", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from deleting PipelineRun with non-existing Pipeline",
			command:     []string{"delete", "pipeline-run-1", "-p", "non-existing-pipeline", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "no PipelineRuns associated with Pipeline \"non-existing-pipeline\"",
		},
		{
			name:        "Remove pipelineruns of a pipeline using --keep",
			command:     []string{"rm", "--pipeline", "pipeline", "-n", "ns", "--keep", "2"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns related to Pipeline \"pipeline\" keeping 2 PipelineRuns (y/n): All but 2 PipelineRuns(Completed) associated with Pipeline \"pipeline\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from using argument with --keep",
			command:     []string{"rm", "pipeline-run-1", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--keep flag should not have any arguments specified with it",
		},
		{
			name:        "Delete all PipelineRuns older than 60mn associated with Pipeline pipeline",
			command:     []string{"delete", "-f", "--pipeline", "pipeline", "--keep-since", "60", "-n", "ns"},
			dynamic:     seeds[6].dynamicClient,
			input:       seeds[6].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 3 expired PipelineRuns(Completed) associated with Pipeline \"pipeline\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all PipelineRuns older than 60mn and keeping 2 PipelineRuns",
			command:     []string{"delete", "-f", "--keep-since", "60", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[11].dynamicClient,
			input:       seeds[11].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "1 PipelineRuns(Completed) has been deleted in namespace \"ns\", kept 2\n",
		},
		{
			name:        "Delete all PipelineRuns older than 60mn and keeping 2 PipelineRuns associated with Pipeline pipeline",
			command:     []string{"delete", "-f", "--pipeline", "pipeline", "--keep-since", "60", "--keep", "2", "-n", "ns"},
			dynamic:     seeds[10].dynamicClient,
			input:       seeds[10].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "1 PipelineRuns(Completed) associated with Pipeline \"pipeline\" has been deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error --keep-since less than zero",
			command:     []string{"delete", "-f", "--keep-since", "-1", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "keep-since option should not be lower than 0",
		},
		{
			name:        "Error --keep-since and --keep less than zero",
			command:     []string{"delete", "-f", "--keep-since", "-1", "--keep", "-1", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "keep and keep-since option should not be lower than 0",
		},
		{
			name:        "Error --keep-since, --all and --task cannot be used",
			command:     []string{"delete", "-f", "--keep-since", "1", "--all", "-p", "foobar", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--keep or --keep-since, --all and --pipeline cannot be used together",
		},
		{
			name:        "Error --keep-since, --keep, --all and --task cannot be used",
			command:     []string{"delete", "-f", "--keep-since", "1", "--keep", "1", "--all", "-p", "foobar", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "--keep or --keep-since, --all and --pipeline cannot be used together",
		},
		{
			name:        "Delete all with explicit ignore-running true",
			command:     []string{"rm", "--all", "-n", "ns", "--ignore-running=true"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns in namespace \"ns\" (y/n): All PipelineRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with default ignore-running",
			command:     []string{"rm", "--all", "-n", "ns"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns in namespace \"ns\" (y/n): All PipelineRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with ignore-running false",
			command:     []string{"rm", "--all", "-n", "ns", "--ignore-running=false"},
			dynamic:     seeds[5].dynamicClient,
			input:       seeds[5].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns in namespace \"ns\" (y/n): All PipelineRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete the PipelineRun present and give error for non-existent PipelineRun",
			command:     []string{"delete", "nonexistent", "pipeline-run-1", "-n", "ns"},
			dynamic:     seeds[7].dynamicClient,
			input:       seeds[7].pipelineClient,
			inputStream: nil,
			wantError:   true,
			want:        "pipelineruns.tekton.dev \"nonexistent\" not found",
		},
		// test pipelinerun delete with status as nil
		{
			name:        "Delete all the pipelineruns with keep-since except one without status conditions",
			command:     []string{"delete", "--keep-since", "1", "-n", "ns", "-f"},
			dynamic:     seeds[8].dynamicClient,
			input:       seeds[8].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "3 expired PipelineRuns(Completed) has been deleted in namespace \"ns\", kept 0\n",
		},
		{
			name:        "Delete all the pipelineruns with keep-since except one without status conditions",
			command:     []string{"delete", "--keep", "1", "-n", "ns", "-f", "--ignore-running=false"},
			dynamic:     seeds[8].dynamicClient,
			input:       seeds[8].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All but 1 PipelineRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all the pipelineruns with keep-since ignore-running false except one without status conditions",
			command:     []string{"delete", "--keep-since", "1", "-n", "ns", "-f", "--ignore-running=false"},
			dynamic:     seeds[9].dynamicClient,
			input:       seeds[9].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "6 expired PipelineRuns has been deleted in namespace \"ns\", kept 0\n",
		},
		{
			name:        "Delete all the pipelineruns including one without status",
			command:     []string{"delete", "--all", "-n", "ns", "-f", "--ignore-running=false"},
			dynamic:     seeds[9].dynamicClient,
			input:       seeds[9].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All PipelineRuns deleted in namespace \"ns\"\n",
		},
		{
			name:        "No pipelineruns after deleting all including running and nil status",
			command:     []string{"ls", "-n", "ns"},
			dynamic:     seeds[9].dynamicClient,
			input:       seeds[9].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "No PipelineRuns found\n",
		},
		{
			name:        "Attempt to delete PieplineRun by keeping more than existing PipelineRun for a Pipeline",
			command:     []string{"delete", "-i", "-f", "--keep", "3", "--pipeline", "pipeline", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "There is/are only 2 PipelineRun(s) associated for Pipeline: pipeline \n",
		},
		{
			name:        "Attempt to delete PieplineRun by keeping more than existing PipelineRun for a Pipeline",
			command:     []string{"delete", "-i", "-f", "--keep", "2", "--pipeline", "pipeline", "-n", "ns"},
			dynamic:     seeds[2].dynamicClient,
			input:       seeds[2].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Associated PipelineRun (2) for Pipeline:pipeline is/are equal to keep (2) \n",
		},
		{
			name:        "Delete all of a pipeline by default ignore-running",
			command:     []string{"rm", "-p", "pipeline", "-n", "ns"},
			dynamic:     seeds[12].dynamicClient,
			input:       seeds[12].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns related to Pipeline \"pipeline\" (y/n): All PipelineRuns(Completed) associated with Pipeline \"pipeline\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all of a pipeline by explicit ignore-running true",
			command:     []string{"rm", "-p", "pipeline", "-n", "ns", "--ignore-running=true"},
			dynamic:     seeds[13].dynamicClient,
			input:       seeds[13].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns related to Pipeline \"pipeline\" (y/n): All PipelineRuns(Completed) associated with Pipeline \"pipeline\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all of a pipeline by ignore-running false",
			command:     []string{"rm", "-p", "pipeline", "-n", "ns", "--ignore-running=false"},
			dynamic:     seeds[14].dynamicClient,
			input:       seeds[14].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "Are you sure you want to delete all PipelineRuns related to Pipeline \"pipeline\" (y/n): All PipelineRuns associated with Pipeline \"pipeline\" deleted in namespace \"ns\"\n",
		},
		{
			name:        "Ignore deleting PipelineRun with Status Unknown",
			command:     []string{"delete", "--all", "-n", "ns", "-f"},
			dynamic:     seeds[15].dynamicClient,
			input:       seeds[15].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "All PipelineRuns(Completed) deleted in namespace \"ns\"\n",
		},
		{
			name:        "PipelineRun with Status Unknown exists",
			command:     []string{"ls", "-n", "ns"},
			dynamic:     seeds[15].dynamicClient,
			input:       seeds[15].pipelineClient,
			inputStream: nil,
			wantError:   false,
			want:        "NAME             STARTED         DURATION   STATUS\npipeline-run-4   ---             ---        ---\npipeline-run-5   ---             ---        Pending\npipeline-run-6   0 seconds ago   ---        Succeeded(Running)\npipeline-run-7   0 seconds ago   ---        Pending\n",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube, Dynamic: tp.dynamic}
			pipelinerun := Command(p)

			if tp.inputStream != nil {
				pipelinerun.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(pipelinerun, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				} else {
					test.AssertOutput(t, tp.want, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}
