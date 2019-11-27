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
	htest "github.com/tektoncd/cli/pkg/helper/test"
	"github.com/tektoncd/cli/pkg/test"
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
					t.Errorf("error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
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

	testParams := []struct {
		name     string
		resource string
		prompt   htest.PromptTest
		options  []string
		want     LogOptions
	}{
		{
			name:     "select pipeline name",
			resource: ResourceNamePipeline,
			prompt: htest.PromptTest{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select pipeline :"); err != nil {
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
			},
		},
		{
			name:     "select pipelinerun name",
			resource: ResourceNamePipelineRun,
			prompt: htest.PromptTest{
				CmdArgs: []string{},
				Procedure: func(c *goexpect.Console) error {
					if _, err := c.ExpectString("Select pipelinerun :"); err != nil {
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
			},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			opts := &LogOptions{}
			tp.prompt.RunTest(t, tp.prompt.Procedure, func(stdio terminal.Stdio) error {
				opts.AskOpts = htest.WithStdio(stdio)
				return opts.Ask(tp.resource, tp.options)
			})
			if opts.PipelineName != tp.want.PipelineName {
				t.Errorf("unexpected PipelineName")
			}
			if opts.PipelineRunName != tp.want.PipelineRunName {
				t.Errorf("unexpected PipelineRunName")
			}
		})
	}
}
