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

package get

import (
	"bytes"

	"github.com/spf13/cobra"
	"github.com/tektoncd/hub/api/pkg/cli/hub"
	"github.com/tektoncd/hub/api/pkg/cli/printer"
)

type taskOptions struct {
	*options
	clusterTask bool
}

func taskCommand(opts *options) *cobra.Command {
	kind := "task"
	taskOpts := &taskOptions{options: opts}

	cmd := &cobra.Command{
		Use:          kind,
		Short:        "Get Task by name, catalog and version",
		Long:         ``,
		SilenceUsage: true,
		Example:      examples(kind),
		Annotations: map[string]string{
			"commandType": "main",
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskOpts.kind = kind
			taskOpts.args = args
			return taskOpts.run()
		},
	}

	cmd.Flags().BoolVar(&taskOpts.clusterTask, "as-clustertask", false, "Get the Task as ClusterTask")

	return cmd
}

func (opts *taskOptions) run() error {

	if err := opts.validate(); err != nil {
		return err
	}

	hubClient := opts.cli.Hub()
	resource := hubClient.GetResource(hub.ResourceOption{
		Name:    opts.name(),
		Catalog: opts.from,
		Kind:    opts.kind,
		Version: opts.version,
	})

	data, err := resource.Manifest()
	if err != nil {
		return err
	}

	if opts.clusterTask {
		data = taskToClusterTask(data)
	}

	out := opts.cli.Stream().Out
	return printer.New(out).Raw(data, nil)
}

func taskToClusterTask(data []byte) []byte {
	return bytes.Replace(data, []byte("kind: Task"), []byte("kind: ClusterTask"), 1)
}
