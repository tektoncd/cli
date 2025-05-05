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
	"github.com/spf13/cobra"
	"github.com/tektoncd/hub/api/pkg/cli/hub"
	"github.com/tektoncd/hub/api/pkg/cli/printer"
)

type taskOptions struct {
	*options
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
		RunE: func(cmd *cobra.Command, args []string) error {
			taskOpts.kind = kind
			taskOpts.args = args
			return taskOpts.run()
		},
	}

	return cmd
}

func (opts *taskOptions) run() error {

	if err := opts.validate(); err != nil {
		return err
	}

	opts.hubClient = opts.cli.Hub()
	var err error

	name, err := opts.GetResourceInfo()
	if err != nil {
		return err
	}

	resource := opts.hubClient.GetResourceYaml(hub.ResourceOption{
		Name:    name,
		Catalog: opts.from,
		Kind:    opts.kind,
		Version: opts.version,
	})

	data, err := resource.ResourceYaml()
	if err != nil {
		return err
	}

	out := opts.cli.Stream().Out
	return printer.New(out).Raw([]byte(data), nil)
}
