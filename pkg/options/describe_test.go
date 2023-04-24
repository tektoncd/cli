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

	options6 := []string{
		"clustertask1",
		"clustertask2",
		"clustertask3",
	}

	options7 := []string{
		"eventlistener1",
		"eventlistener2",
		"eventlistener3",
	}

	options8 := []string{
		"triggertemplate1",
		"triggertemplate2",
		"triggertemplate3",
	}

	options9 := []string{
		"triggerbinding1",
		"triggerbinding2",
		"triggerbinding3",
	}

	options10 := []string{
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
					if _, err := c.SendLine(options2[0]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options2,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "sample-pipeline-run1",
				TaskName:                  "",
				TaskrunName:               "",
				ClusterTaskName:           "",
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
					if _, err := c.SendLine(options2[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options2,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "sample-pipeline-run3",
				TaskName:                  "",
				TaskrunName:               "",
				ClusterTaskName:           "",
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
					if _, err := c.SendLine(options4[0]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options4,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "sample-task-run1",
				ClusterTaskName:           "",
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
					if _, err := c.SendLine(options4[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options4,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "sample-task-run3",
				ClusterTaskName:           "",
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
					if _, err := c.SendLine(options3[0]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options3,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "task1",
				TaskrunName:               "",
				ClusterTaskName:           "",
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
					if _, err := c.SendLine(options3[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options3,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "task3",
				TaskrunName:               "",
				ClusterTaskName:           "",
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
					if _, err := c.SendLine(options[0]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options,
			want: DescribeOptions{
				PipelineName:              "pipeline1",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "",
				ClusterTaskName:           "",
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
					if _, err := c.SendLine(options[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options,
			want: DescribeOptions{
				PipelineName:              "pipeline3",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "",
				ClusterTaskName:           "",
				TriggerTemplateName:       "",
				TriggerBindingName:        "",
				ClusterTriggerBindingName: "",
				EventListenerName:         "",
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
					if _, err := c.SendLine(options6[0]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options6,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "",
				ClusterTaskName:           "clustertask1",
				TriggerTemplateName:       "",
				TriggerBindingName:        "",
				ClusterTriggerBindingName: "",
				EventListenerName:         "",
			},
		},
		{
			name:     "select clustertask name option 3",
			resource: ResourceNameClusterTask,
			prompt: prompt.Prompt{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select clustertask:"); err != nil {
						return err
					}
					if _, err := c.SendLine(options6[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options6,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "",
				ClusterTaskName:           "clustertask3",
				TriggerTemplateName:       "",
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
					if _, err := c.SendLine(options8[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options8,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "",
				ClusterTaskName:           "",
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
					if _, err := c.SendLine(options9[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options9,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "",
				ClusterTaskName:           "",
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
					if _, err := c.SendLine(options10[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options10,
			want: DescribeOptions{
				PipelineName:              "",
				PipelineRunName:           "",
				TaskName:                  "",
				TaskrunName:               "",
				ClusterTaskName:           "",
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
					if _, err := c.SendLine(options7[2]); err != nil {
						return err
					}
					return nil
				},
			},
			options: options7,
			want: DescribeOptions{
				PipelineName:      "",
				PipelineRunName:   "",
				TaskName:          "",
				TaskrunName:       "",
				ClusterTaskName:   "",
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
			if opts.ClusterTaskName != tp.want.ClusterTaskName {
				t.Errorf("Unexpected ClusterTask Name")
			}
			if opts.EventListenerName != tp.want.EventListenerName {
				t.Errorf("Unexpected EventListener Name")
			}
		})
	}
}
