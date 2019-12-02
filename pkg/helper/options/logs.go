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
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/helper/pods/stream"
)

const (
	ResourceNamePipeline    = "pipeline"
	ResourceNamePipelineRun = "pipelinerun"
	ResourceNameTaskRun     = "taskrun"
)

type LogOptions struct {
	AllSteps        bool
	Follow          bool
	Params          cli.Params
	PipelineName    string
	PipelineRunName string
	TaskrunName     string
	Stream          *cli.Stream
	Streamer        stream.NewStreamerFunc
	Tasks           []string
	Last            bool
	Limit           int
	AskOpts         survey.AskOpt
}

func NewLogOptions(p cli.Params) *LogOptions {
	return &LogOptions{Params: p,
		AskOpts: func(opt *survey.AskOptions) error {
			opt.Stdio = terminal.Stdio{
				In:  os.Stdin,
				Out: os.Stdout,
				Err: os.Stderr,
			}
			return nil
		},
	}
}

func (opts *LogOptions) ValidateOpts() error {
	if opts.Limit <= 0 {
		return fmt.Errorf("limit was %d but must be a positive number", opts.Limit)
	}
	return nil
}

func (opts *LogOptions) Ask(resource string, options []string) error {
	var ans string
	var qs = []*survey.Question{
		{
			Name: resource,
			Prompt: &survey.Select{
				Message: fmt.Sprintf("Select %s :", resource),
				Options: options,
			},
		},
	}

	if err := survey.Ask(qs, &ans, opts.AskOpts); err != nil {
		return err
	}

	switch resource {
	case ResourceNamePipeline:
		opts.PipelineName = ans
	case ResourceNamePipelineRun:
		opts.PipelineRunName = strings.Fields(ans)[0]
	case ResourceNameTaskRun:
		opts.TaskrunName = strings.Fields(ans)[0]
	}

	return nil
}
