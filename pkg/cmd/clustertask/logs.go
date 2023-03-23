// Copyright © 2021 The Tekton Authors.
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

package clustertask

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/clustertask"
	"github.com/tektoncd/cli/pkg/cmd/taskrun"
	"github.com/tektoncd/cli/pkg/flags"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/task"
	taskrunpkg "github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func nameArg(args []string, p cli.Params) error {
	if len(args) == 1 {
		c, err := p.Clients()
		if err != nil {
			return err
		}
		name := args[0]
		var clustertask *v1beta1.ClusterTask
		if err = actions.GetV1(clustertaskGroupResource, c, name, "", metav1.GetOptions{}, &clustertask); err != nil {
			return err
		}
	}
	return nil
}

func logCommand(p cli.Params) *cobra.Command {
	opts := options.NewLogOptions(p)

	eg := `Interactive mode: shows logs of the selected TaskRun:

    tkn clustertask logs -n namespace

Interactive mode: shows logs of the selected TaskRun of the given ClusterTask:

    tkn clustertask logs clustertask -n namespace

Show logs of given ClusterTask for last TaskRun:

    tkn clustertask logs clustertask -n namespace --last

Show logs for given ClusterTask and associated TaskRun:

    tkn clustertask logs clustertask taskrun -n namespace
`
	c := &cobra.Command{
		Use:                   "logs",
		DisableFlagsInUseLine: true,
		Short:                 "Show ClusterTask logs",
		Example:               eg,
		SilenceUsage:          true,
		ValidArgsFunction:     formatted.ParentCompletion,
		Annotations: map[string]string{
			"commandType": "main",
		},

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
	c.Flags().BoolVarP(&opts.Last, "last", "L", false, "show logs for last TaskRun")
	c.Flags().BoolVarP(&opts.AllSteps, "all", "a", false, "show all logs including init steps injected by tekton")
	c.Flags().BoolVarP(&opts.Follow, "follow", "f", false, "stream live logs")
	c.Flags().BoolVarP(&opts.Timestamps, "timestamps", "t", false, "show logs with timestamp")
	c.Flags().IntVarP(&opts.Limit, "limit", "", 5, "lists number of TaskRuns")
	c.Deprecated = "ClusterTasks are deprecated, this command will be removed in future releases."
	return c
}

func run(opts *options.LogOptions, args []string) error {
	if err := initOpts(opts, args); err != nil {
		return err
	}

	if opts.ClusterTaskName == "" || opts.TaskrunName == "" {
		return nil
	}

	return taskrun.Run(opts)
}

func initOpts(opts *options.LogOptions, args []string) error {
	// ensure the client is properly initialized
	if _, err := opts.Params.Clients(); err != nil {
		return err
	}

	if err := opts.ValidateOpts(); err != nil {
		return err
	}

	switch len(args) {
	case 0: // no inputs
		return getAllInputs(opts)

	case 1: // clustertask name provided
		opts.ClusterTaskName = args[0]
		return askRunName(opts)

	case 2: // both clustertask and run provided
		opts.ClusterTaskName = args[0]
		opts.TaskrunName = args[1]

	default:
		return fmt.Errorf("too many arguments")
	}
	return nil
}

func getAllInputs(opts *options.LogOptions) error {
	cs, err := opts.Params.Clients()
	if err != nil {
		return err
	}

	cts, err := clustertask.GetAllClusterTaskNames(clustertaskGroupResource, cs)
	if err != nil {
		return err
	}

	if len(cts) == 0 {
		return fmt.Errorf("no ClusterTasks found")
	}

	if len(cts) == 1 {
		opts.ClusterTaskName = strings.Fields(cts[0])[0]
	} else if err := opts.Ask(options.ResourceNameClusterTask, cts); err != nil {
		return err
	}

	return askRunName(opts)
}

func askRunName(opts *options.LogOptions) error {
	cs, err := opts.Params.Clients()
	if err != nil {
		return err
	}

	if opts.Last {
		name, err := initLastRunName(cs, opts.ClusterTaskName, opts.Params.Namespace())
		if err != nil {
			return err
		}
		opts.TaskrunName = name
		return nil
	}

	lOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/clusterTask=%s", opts.ClusterTaskName),
	}

	trs, err := taskrunpkg.GetAllTaskRuns(taskrunGroupResource, lOpts, cs, opts.Params.Namespace(), opts.Limit, opts.Params.Time())
	if err != nil {
		return err
	}

	if len(trs) == 0 {
		return fmt.Errorf("no TaskRuns found for ClusterTask %s", opts.ClusterTaskName)
	}

	if len(trs) == 1 {
		opts.TaskrunName = strings.Fields(trs[0])[0]
		return nil
	}

	return opts.Ask(options.ResourceNameTaskRun, trs)
}

func initLastRunName(cs *cli.Clients, name, namespace string) (string, error) {
	lastrun, err := task.LastRun(cs, name, namespace, "ClusterTask")
	if err != nil {
		return "", err
	}
	return lastrun.Name, nil
}
