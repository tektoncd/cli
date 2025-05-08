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

package options

import (
	"testing"

	"github.com/AlecAivazis/survey/v2/terminal"
	goexpect "github.com/Netflix/go-expect"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/cli/test/prompt"
)

func TestDescribeOptions_ValidateOpts(t *testing.T) {

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
			opts := NewDescribeOptions(&cli.TektonParams{})
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

func TestDescribeOptions_Ask(t *testing.T) {
	pipelineOptions := []string{
		"pipeline1",
		"pipeline2",
		"pipeline3",
	}

	pipelineRunOptions := []string{
		"sample-pipeline-run1 started 1 minutes ago",
		"sample-pipeline-run2 started 2 minutes ago",
		"sample-pipeline-run3 started 3 minutes ago",
	}

	taskOptions := []string{
		"task1",
		"task2",
		"task3",
	}

	taskRunOptions := []string{
		"sample-task-run1 started 1 minutes ago",
		"sample-task-run2 started 2 minutes ago",
		"sample-task-run3 started 3 minutes ago",
	}

	eventListenerOptions := []string{
		"eventlistener1",
		"eventlistener2",
		"eventlistener3",
	}

	triggerTemplateOptions := []string{
		"triggertemplate1",
		"triggertemplate2",
		"triggertemplate3",
	}

	triggerBindingOptions := []string{
		"triggerbinding1",
		"triggerbinding2",
		"triggerbinding3",
	}

	clusterTriggerBindingOptions := []string{
		"clustertriggerbinding1",
		"clustertriggerbinding2",
		"clustertriggerbinding3",
	}

	testParams := []struct {
		name     string
		resource string
		prompt   prompt.Prompt
		options  []string
		want     DescribeOptions
	}{
		{
			name:     "select pipelinerun name",
			resource: ResourceNamePipelineRun,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select pipelinerun:"); err != nil {
						return err
					}
					if _, err := c.SendLine(pipelineRunOptions[0]); err != nil {
						return err
					}
					return nil
				},
			},
			options: pipelineRunOptions,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "sample-pipeline-run1",
				TaskName:                  "",
				TaskrunName:               "",
				TriggerTemplateName:       "",
				TriggerBindingName:        "",
				ClusterTriggerBindingName: "",
				EventListenerName:         "",
			},
		},
		{
			name:     "select pipelinerun name option 3",
			resource: ResourceNamePipelineRun,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select pipelinerun:"); err != nil {
						return err
					}
					if _, err := c.SendLine(pipelineRunOptions[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: pipelineRunOptions,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "sample-pipeline-run3",
				TaskName:                  "",
				TaskrunName:               "",
				TriggerTemplateName:       "",
				TriggerBindingName:        "",
				ClusterTriggerBindingName: "",
				EventListenerName:         "",
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
					if _, err := c.SendLine(taskRunOptions[0]); err != nil {
						return err
					}
					return nil
				},
			},
			options: taskRunOptions,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "sample-task-run1",
				TriggerTemplateName:       "",
				TriggerBindingName:        "",
				ClusterTriggerBindingName: "",
				EventListenerName:         "",
			},
		},
		{
			name:     "select taskrun name option 3",
			resource: ResourceNameTaskRun,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select taskrun:"); err != nil {
						return err
					}
					if _, err := c.SendLine(taskRunOptions[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: taskRunOptions,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "sample-task-run3",
				TriggerTemplateName:       "",
				TriggerBindingName:        "",
				ClusterTriggerBindingName: "",
				EventListenerName:         "",
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
					if _, err := c.SendLine(taskOptions[0]); err != nil {
						return err
					}
					return nil
				},
			},
			options: taskOptions,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "task1",
				TaskrunName:               "",
				TriggerTemplateName:       "",
				TriggerBindingName:        "",
				ClusterTriggerBindingName: "",
				EventListenerName:         "",
			},
		},
		{
			name:     "select task name option 3",
			resource: ResourceNameTask,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select task:"); err != nil {
						return err
					}
					if _, err := c.SendLine(taskOptions[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: taskOptions,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "task3",
				TaskrunName:               "",
				TriggerTemplateName:       "",
				TriggerBindingName:        "",
				ClusterTriggerBindingName: "",
				EventListenerName:         "",
			},
		},
		{
			name:     "select pipeline name",
			resource: ResourceNamePipeline,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select pipeline:"); err != nil {
						return err
					}
					if _, err := c.SendLine(pipelineOptions[0]); err != nil {
						return err
					}
					return nil
				},
			},
			options: pipelineOptions,
			want: DescribeOptions{
				PipelineName:              "pipeline1",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "",
				TriggerTemplateName:       "",
				TriggerBindingName:        "",
				ClusterTriggerBindingName: "",
				EventListenerName:         "",
			},
		},
		{
			name:     "select pipeline name option 3",
			resource: ResourceNamePipeline,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select pipeline:"); err != nil {
						return err
					}
					if _, err := c.SendLine(pipelineOptions[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: pipelineOptions,
			want: DescribeOptions{
				PipelineName:              "pipeline3",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "",
				TriggerTemplateName:       "",
				TriggerBindingName:        "",
				ClusterTriggerBindingName: "",
				EventListenerName:         "",
			},
		},
		{
			name:     "select triggertemplate name option 3",
			resource: ResourceNameTriggerTemplate,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select triggertemplate:"); err != nil {
						return err
					}
					if _, err := c.SendLine(triggerTemplateOptions[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: triggerTemplateOptions,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "",
				TriggerTemplateName:       "triggertemplate3",
				TriggerBindingName:        "",
				ClusterTriggerBindingName: "",
				EventListenerName:         "",
			},
		},
		{
			name:     "select triggerbinding name option 3",
			resource: ResourceNameTriggerBinding,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select triggerbinding:"); err != nil {
						return err
					}
					if _, err := c.SendLine(triggerBindingOptions[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: triggerBindingOptions,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "",
				TriggerTemplateName:       "",
				TriggerBindingName:        "triggerbinding3",
				ClusterTriggerBindingName: "",
				EventListenerName:         "",
			},
		},
		{
			name:     "select clustertriggerbinding name option 3",
			resource: ResourceNameClusterTriggerBinding,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select clustertriggerbinding:"); err != nil {
						return err
					}
					if _, err := c.SendLine(clusterTriggerBindingOptions[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: clusterTriggerBindingOptions,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "",
				TriggerTemplateName:       "",
				TriggerBindingName:        "",
				ClusterTriggerBindingName: "clustertriggerbinding3",
				EventListenerName:         "",
			},
		},
		{
			name:     "select eventlistener name option 3",
			resource: ResourceNameEventListener,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select eventlistener:"); err != nil {
						return err
					}
					if _, err := c.SendLine(eventListenerOptions[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: eventListenerOptions,
			want: DescribeOptions{
				PipelineName:      "",
				PipelineRunName:   "",
				TaskName:          "",
				TaskrunName:       "",
				EventListenerName: "eventlistener3",
			},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			opts := &DescribeOptions{}
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
			if opts.TaskrunName != tp.want.TaskrunName {
				t.Errorf("Unexpected TaskRun Name")
			}
			if opts.TriggerTemplateName != tp.want.TriggerTemplateName {
				t.Errorf("Unexpected TriggerTemplate Name")
			}
			if opts.TriggerBindingName != tp.want.TriggerBindingName {
				t.Errorf("Unexpected TriggerBinding Name")
			}
			if opts.ClusterTriggerBindingName != tp.want.ClusterTriggerBindingName {
				t.Errorf("Unexpected ClusterTriggerBinding Name")
			}
			if opts.EventListenerName != tp.want.EventListenerName {
				t.Errorf("Unexpected EventListener Name")
			}
		})
	}
}
