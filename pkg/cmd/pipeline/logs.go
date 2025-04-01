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
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cmd/pipelinerun"
	"github.com/tektoncd/cli/pkg/flags"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	pipelinepkg "github.com/tektoncd/cli/pkg/pipeline"
	pipelinerunpkg "github.com/tektoncd/cli/pkg/pipelinerun"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func nameArg(args []string, p cli.Params) error {
	if len(args) == 1 {
		c, err := p.Clients()
		if err != nil {
			return err
		}
		name, ns := args[0], p.Namespace()
		var pipeline *v1.Pipeline
		if err = actions.GetV1(pipelineGroupResource, c, name, ns, metav1.GetOptions{}, &pipeline); err != nil {
			return err
		}
	}
	return nil
}

func logCommand(p cli.Params) *cobra.Command {
	opts := options.NewLogOptions(p)

	eg := `
Interactive mode: shows logs of the selected PipelineRun:

    tkn pipeline logs -n namespace

Interactive mode: shows logs of the selected PipelineRun of the given Pipeline:

    tkn pipeline logs pipeline -n namespace

Show logs of given Pipeline for last run:

    tkn pipeline logs pipeline -n namespace --last

Show logs for given Pipeline and PipelineRun:

    tkn pipeline logs pipeline run -n namespace
`
	c := &cobra.Command{
		Use:                   "logs",
		DisableFlagsInUseLine: true,
		Short:                 "Show Pipeline logs",
		Example:               eg,
		SilenceUsage:          true,

		Annotations: map[string]string{
			"commandType": "main",
		},
		ValidArgsFunction: formatted.ParentCompletion,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := flags.InitParams(p, cmd); err != nil {
				return err
			}
			return nameArg(args, p)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Stream = &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			return run(opts, args)
		},
	}
	c.Flags().BoolVarP(&opts.Last, "last", "L", false, "show logs for last PipelineRun")
	c.Flags().BoolVarP(&opts.AllSteps, "all", "a", false, "show all logs including init steps injected by tekton")
	c.Flags().BoolVarP(&opts.Follow, "follow", "f", false, "stream live logs")
	c.Flags().BoolVarP(&opts.Timestamps, "timestamps", "t", false, "show logs with timestamp")
	c.Flags().IntVarP(&opts.Limit, "limit", "", 5, "lists number of PipelineRuns")
	c.Flags().BoolVarP(&opts.Prefixing, "prefix", "", true, "prefix each log line with the log source (task name and step name)")

	return c
}

func run(opts *options.LogOptions, args []string) error {
	if err := initOpts(opts, args); err != nil {
		return err
	}

	if opts.PipelineName == "" || opts.PipelineRunName == "" {
		return nil
	}

	return pipelinerun.Run(opts)
}

func initOpts(opts *options.LogOptions, args []string) error {
	// ensure the client is properly initialized
	if _, err := opts.Params.Clients(); err != nil {
		return err
	}

	switch len(args) {
	case 0: // no inputs
		return getAllInputs(opts)

	case 1: // pipeline name provided
		opts.PipelineName = args[0]
		return askRunName(opts)

	case 2: // both pipeline and run provided
		opts.PipelineName = args[0]
		opts.PipelineRunName = args[1]

	default:
		return fmt.Errorf("too many arguments")
	}
	return nil
}

func getAllInputs(opts *options.LogOptions) error {
	if err := opts.ValidateOpts(); err != nil {
		return err
	}

	cs, err := opts.Params.Clients()
	if err != nil {
		return err
	}

	ps, err := pipelinepkg.GetAllPipelineNames(pipelineGroupResource, cs, opts.Params.Namespace())
	if err != nil {
		return err
	}

	if len(ps) == 0 {
		fmt.Fprintf(opts.Stream.Out, "No Pipelines found in namespace %s", opts.Params.Namespace())
		return nil
	}

	if len(ps) == 1 {
		opts.PipelineName = strings.Fields(ps[0])[0]
	} else if err := opts.Ask(options.ResourceNamePipeline, ps); err != nil {
		return err
	}

	return askRunName(opts)
}

func askRunName(opts *options.LogOptions) error {
	if err := opts.ValidateOpts(); err != nil {
		return err
	}

	cs, err := opts.Params.Clients()
	if err != nil {
		return err
	}

	if opts.Last {
		name, err := pipelinepkg.LastRunName(cs, opts.PipelineName, opts.Params.Namespace())
		if err != nil {
			return err
		}
		opts.PipelineRunName = name
		return nil
	}

	lOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/pipeline=%s", opts.PipelineName),
	}

	prs, err := pipelinerunpkg.GetAllPipelineRuns(pipelineRunGroupResource, lOpts, cs, opts.Params.Namespace(), opts.Limit, opts.Params.Time())
	if err != nil {
		return err
	}

	if len(prs) == 0 {
		fmt.Fprintf(opts.Stream.Out, "No PipelineRuns found for Pipeline %s", opts.PipelineName)
		return nil
	}

	if len(prs) == 1 {
		opts.PipelineRunName = strings.Fields(prs[0])[0]
		return nil
	}

	return opts.Ask(options.ResourceNamePipelineRun, prs)
}
