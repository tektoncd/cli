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

package options

import (
	"testing"

	"github.com/AlecAivazis/survey/v2/terminal"
	goexpect "github.com/Netflix/go-expect"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/cli/test/prompt"
)

func TestLogOptions_ValidateOpts(t *testing.T) {

	testParams := []struct {
		name      string
		limit     int
		wantError bool
		want      string
	}{
		{
			name:      "valid limit",
			limit:     1,
			wantError: false,
			want:      "",
		},
		{
			name:      "invalid limit",
			limit:     0,
			wantError: true,
			want:      "limit was 0 but must be a positive number",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			opts := NewLogOptions(&cli.TektonParams{})
			opts.Limit = tp.limit

			err := opts.ValidateOpts()
			if tp.wantError {
				if err == nil {
					t.Errorf("Error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else if err != nil {
				t.Errorf("unexpected Error")
			}
		})
	}
}

func TestLogOptions_Ask(t *testing.T) {

	options := []string{
		"pipeline1",
		"pipeline2",
		"pipeline3",
	}
	options2 := []string{
		"sample-pipeline-run1 started 1 minutes ago",
		"sample-pipeline-run2 started 2 minutes ago",
		"sample-pipeline-run3 started 3 minutes ago",
	}
	options3 := []string{
		"task1",
		"task2",
		"task3",
	}
	options4 := []string{
		"sample-task-run1 started 1 minutes ago",
		"sample-task-run2 started 2 minutes ago",
		"sample-task-run3 started 3 minutes ago",
	}
	options5 := []string{
		"clustertask1",
		"clustertask2",
		"clustertask3",
	}

	testParams := []struct {
		name     string
		resource string
		prompt   prompt.Prompt
		options  []string
		want     LogOptions
	}{
		{
			name:     "select pipeline name",
			resource: ResourceNamePipeline,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select pipeline:"); err != nil {
						return err
					}
					if _, err := c.SendLine(options[0]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options,
			want: LogOptions{
				PipelineName:    "pipeline1",
				PipelineRunName: "",
				TaskName:        "",
				ClusterTaskName: "",
				TaskrunName:     "",
			},
		},
		{
			name:     "select pipelinerun name",
			resource: ResourceNamePipelineRun,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select pipelinerun:"); err != nil {
						return err
					}
					if _, err := c.SendLine(options2[0]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options2,
			want: LogOptions{
				PipelineName:    "",
				PipelineRunName: "sample-pipeline-run1",
				TaskName:        "",
				ClusterTaskName: "",
				TaskrunName:     "",
			},
		},
		{
			name:     "select task name",
			resource: ResourceNameTask,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select task:"); err != nil {
						return err
					}
					if _, err := c.SendLine(options3[0]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options3,
			want: LogOptions{
				PipelineName:    "",
				PipelineRunName: "",
				TaskName:        "task1",
				ClusterTaskName: "",
				TaskrunName:     "",
			},
		},
		{
			name:     "select taskrun name",
			resource: ResourceNameTaskRun,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select taskrun:"); err != nil {
						return err
					}
					if _, err := c.SendLine(options4[0]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options4,
			want: LogOptions{
				PipelineName:    "",
				PipelineRunName: "",
				TaskName:        "",
				ClusterTaskName: "",
				TaskrunName:     "sample-task-run1",
			},
		},
		{
			name:     "select clustertask name",
			resource: ResourceNameClusterTask,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select clustertask:"); err != nil {
						return err
					}
					if _, err := c.SendLine(options5[0]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options5,
			want: LogOptions{
				PipelineName:    "",
				PipelineRunName: "",
				TaskName:        "",
				ClusterTaskName: "clustertask1",
				TaskrunName:     "",
			},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			opts := &LogOptions{}
			tp.prompt.RunTest(t, tp.prompt.Procedure, func(stdio terminal.Stdio) error {
				opts.AskOpts = prompt.WithStdio(stdio)
				return opts.Ask(tp.resource, tp.options)
			})
			if opts.PipelineName != tp.want.PipelineName {
				t.Errorf("Unexpected Pipeline Name")
			}
			if opts.PipelineRunName != tp.want.PipelineRunName {
				t.Errorf("Unexpected PipelineRun Name")
			}
			if opts.TaskName != tp.want.TaskName {
				t.Errorf("Unexpected Task Name")
			}
			if opts.TaskrunName != tp.want.TaskrunName {
				t.Errorf("Unexpected TaskRun Name")
			}
			if opts.ClusterTaskName != tp.want.ClusterTaskName {
				t.Errorf("Unexpected ClusterTask Name")
			}
		})
	}
}
