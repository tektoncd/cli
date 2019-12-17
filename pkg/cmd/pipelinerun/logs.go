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
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/helper/options"
	prhelper "github.com/tektoncd/cli/pkg/helper/pipelinerun"
	"github.com/tektoncd/cli/pkg/helper/pods"
	validate "github.com/tektoncd/cli/pkg/helper/validate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func logCommand(p cli.Params) *cobra.Command {
	opts := &options.LogOptions{Params: p}
	eg := `Show the logs of PipelineRun named 'foo' from namespace 'bar':

    tkn pipelinerun logs foo -n bar

Show the logs of PipelineRun named 'microservice-1' for task 'build' only from namespace 'bar':

    tkn pr logs microservice-1 -t build -n bar

Show the logs of PipelineRun named 'microservice-1' for all tasks and steps (including init steps) from namespace 'foo':

    tkn pr logs microservice-1 -a -n foo
   `

	c := &cobra.Command{
		Use:                   "logs",
		DisableFlagsInUseLine: true,
		Short:                 "Show the logs of PipelineRun",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Example: eg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				opts.PipelineRunName = args[0]
			}

			opts.Stream = &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if err := validate.NamespaceExists(p); err != nil {
				return err
			}

			return Run(opts)
		},
	}

	c.Flags().BoolVarP(&opts.AllSteps, "all", "a", false, "show all logs including init steps injected by tekton")
	c.Flags().BoolVarP(&opts.Follow, "follow", "f", false, "stream live logs")
	c.Flags().StringSliceVarP(&opts.Tasks, "only-tasks", "t", []string{}, "show logs for mentioned tasks only")
	c.Flags().IntVarP(&opts.Limit, "limit", "", 5, "lists number of pipelineruns")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_pipelinerun")
	return c
}

func Run(opts *options.LogOptions) error {
	if opts.PipelineRunName == "" {
		if err := askRunName(opts); err != nil {
			return err
		}
	}
	streamer := pods.NewStream
	if opts.Streamer != nil {
		streamer = opts.Streamer
	}
	cs, err := opts.Params.Clients()
	if err != nil {
		return err
	}

	lr := &LogReader{
		Run:      opts.PipelineRunName,
		Ns:       opts.Params.Namespace(),
		Clients:  cs,
		Streamer: streamer,
		Stream:   opts.Stream,
		Follow:   opts.Follow,
		AllSteps: opts.AllSteps,
		Tasks:    opts.Tasks,
	}

	logC, errC, err := lr.Read()
	if err != nil {
		return err
	}

	NewLogWriter().Write(opts.Stream, logC, errC)

	return nil
}

func askRunName(opts *options.LogOptions) error {
	lOpts := metav1.ListOptions{}

	prs, err := prhelper.GetAllPipelineRuns(opts.Params, lOpts, opts.Limit)
	if err != nil {
		return err
	}

	if len(prs) == 0 {
		return fmt.Errorf("No pipelineruns found")
	}

	if len(prs) == 1 {
		opts.PipelineRunName = strings.Fields(prs[0])[0]
		return nil
	}

	return opts.Ask(options.ResourceNamePipelineRun, prs)
}
