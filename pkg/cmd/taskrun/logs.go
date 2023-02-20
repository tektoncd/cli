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

package taskrun

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/log"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/taskrun"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultLimit = 5
)

func logCommand(p cli.Params) *cobra.Command {
	opts := &options.LogOptions{Params: p}
	eg := `
Show the logs of TaskRun named 'foo' from the namespace 'bar':

    tkn taskrun logs foo -n bar

Show the live logs of TaskRun named 'foo' from namespace 'bar':

    tkn taskrun logs -f foo -n bar

Show the logs of TaskRun named 'microservice-1' for step 'build' only from namespace 'bar':

    tkn tr logs microservice-1 -s build -n bar
`
	c := &cobra.Command{
		Use:          "logs",
		Short:        "Show TaskRuns logs",
		Example:      eg,
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		ValidArgsFunction: formatted.ParentCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				opts.TaskrunName = args[0]
			}

			if !opts.Fzf {
				if _, ok := os.LookupEnv("TKN_USE_FZF"); ok {
					opts.Fzf = true
				}
			}

			opts.Stream = &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if len(opts.Steps) > 0 && opts.AllSteps {
				return fmt.Errorf("option --all and option --step are not compatible")
			}

			return Run(opts)
		},
	}

	c.Flags().BoolVarP(&opts.Last, "last", "L", false, "show logs for last TaskRun")
	c.Flags().BoolVarP(&opts.AllSteps, "all", "a", false, "show all logs including init steps injected by tekton")
	c.Flags().BoolVarP(&opts.Follow, "follow", "f", false, "stream live logs")
	c.Flags().BoolVarP(&opts.Timestamps, "timestamps", "t", false, "show logs with timestamp")
	c.Flags().BoolVarP(&opts.Prefixing, "prefix", "", true, "prefix each log line with the log source (step name)")
	c.Flags().IntVarP(&opts.Limit, "limit", "", defaultLimit, "lists number of TaskRuns")
	c.Flags().BoolVarP(&opts.Fzf, "fzf", "F", false, "use fzf to select a TaskRun")
	c.Flags().StringSliceVarP(&opts.Steps, "step", "s", []string{}, "show logs for mentioned steps only")

	return c
}

func Run(opts *options.LogOptions) error {
	if opts.TaskrunName == "" {
		if err := opts.ValidateOpts(); err != nil {
			return err
		}
		if err := askRunName(opts); err != nil {
			return err
		}
	}

	lr, err := log.NewReader(log.LogTypeTask, opts)
	if err != nil {
		return err
	}

	logC, errC, err := lr.Read()
	if err != nil {
		return err
	}

	log.NewWriter(log.LogTypeTask, opts.Prefixing).Write(opts.Stream, logC, errC)
	return nil
}

func askRunName(opts *options.LogOptions) error {
	lOpts := metav1.ListOptions{}

	// We are able to show much more than the default 5 with fzf, so let
	// increase that limit limited to 100
	if opts.Fzf && opts.Limit == defaultLimit {
		opts.Limit = 100
	}

	clients, err := opts.Params.Clients()
	if err != nil {
		return err
	}

	trs, err := taskrun.GetAllTaskRuns(taskrunGroupResource, lOpts, clients, opts.Params.Namespace(), opts.Limit, opts.Params.Time())
	if err != nil {
		return err
	}

	if len(trs) == 0 {
		return fmt.Errorf("No TaskRuns found")
	}

	if len(trs) == 1 || opts.Last {
		opts.TaskrunName = strings.Fields(trs[0])[0]
		return nil
	}

	if opts.Fzf {
		return opts.FuzzyAsk(options.ResourceNameTaskRun, trs)
	}
	return opts.Ask(options.ResourceNameTaskRun, trs)
}
