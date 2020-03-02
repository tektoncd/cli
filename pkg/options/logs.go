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
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/fatih/color"
	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/tektoncd/cli/pkg/cli"
	prdesc "github.com/tektoncd/cli/pkg/helper/pipelinerun/description"
	"github.com/tektoncd/cli/pkg/pods/stream"
	trdesc "github.com/tektoncd/cli/pkg/helper/taskrun/description"
)

const (
	ResourceNamePipeline    = "pipeline"
	ResourceNamePipelineRun = "pipelinerun"
	ResourceNameTask        = "task"
	ResourceNameTaskRun     = "taskrun"
)

type LogOptions struct {
	AllSteps        bool
	Follow          bool
	Params          cli.Params
	PipelineName    string
	PipelineRunName string
	TaskName        string
	TaskrunName     string
	Stream          *cli.Stream
	Streamer        stream.NewStreamerFunc
	Tasks           []string
	Steps           []string
	Last            bool
	Limit           int
	AskOpts         survey.AskOpt
	Fzf             bool
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
				Message: fmt.Sprintf("Select %s:", resource),
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
	case ResourceNameTask:
		opts.TaskName = ans
	case ResourceNameTaskRun:
		opts.TaskrunName = strings.Fields(ans)[0]
	}

	return nil
}

func (opts *LogOptions) FuzzyAsk(resource string, options []string) error {
	chosencolouring := color.NoColor
	defer func() {
		color.NoColor = chosencolouring
	}()
	// Remove colors as fuzzyfinder doesn't support it!
	color.NoColor = true

	idx, err := fuzzyfinder.FindMulti(options,
		func(i int) string {
			return strings.Fields(options[i])[0]
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}

			buf := new(bytes.Buffer)
			s := cli.Stream{
				Out: buf,
			}

			bname := strings.Fields(options[i])[0]
			switch resource {
			case ResourceNameTaskRun:
				err := trdesc.PrintTaskRunDescription(&s, bname, opts.Params)
				if err != nil {
					return fmt.Sprintf("Cannot get taskrun description for %s: %s", bname, err.Error())
				}
			case ResourceNamePipelineRun:
				err := prdesc.PrintPipelineRunDescription(&s, bname, opts.Params)
				if err != nil {
					return fmt.Sprintf("Cannot get pipelinerun description for %s: %s", bname, err.Error())
				}
			}
			return buf.String()
		}))
	if err != nil {
		return err
	}
	ans := options[idx[0]]
	switch resource {
	case ResourceNamePipeline:
		opts.PipelineName = ans
	case ResourceNamePipelineRun:
		opts.PipelineRunName = strings.Fields(ans)[0]
	case ResourceNameTask:
		opts.TaskName = ans
	case ResourceNameTaskRun:
		opts.TaskrunName = strings.Fields(ans)[0]
	}

	return nil
}
