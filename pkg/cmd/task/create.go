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

package task

import (
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/helper/file"
	validate "github.com/tektoncd/cli/pkg/helper/validate"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

type createOptions struct {
	from string
}

func createCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("create")
	opts := &createOptions{from: ""}
	eg := `
# Create a Task defined by foo.yaml in namespace 'bar'
tkn task create -f foo.yaml -n bar
`

	c := &cobra.Command{
		Use:          "create",
		Short:        "Create a task in a namespace",
		Example:      eg,
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				In:  cmd.InOrStdin(),
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if err := validate.NamespaceExists(p); err != nil {
				return err
			}

			return createTask(s, p, opts.from)
		},
	}
	f.AddFlags(c)
	c.Flags().StringVarP(&opts.from, "from", "f", "", "local or remote filename to use to create the task")
	return c
}

func createTask(s *cli.Stream, p cli.Params, path string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	task, err := loadTask(p, path)
	if err != nil {
		return err
	}

	_, err = cs.Tekton.TektonV1alpha1().Tasks(p.Namespace()).Create(task)
	if err != nil {
		return fmt.Errorf("failed to create task %q: %s", task.Name, err)
	}

	fmt.Fprintf(s.Out, "Task created: %s\n", task.Name)
	return nil
}

func loadTask(p cli.Params, target string) (*v1alpha1.Task, error) {
	content, err := file.LoadFileContent(p, target, file.IsYamlFile(), fmt.Errorf("inavlid file format for %s: .yaml or .yml file extension and format required", target))
	if err != nil {
		return nil, err
	}

	var task v1alpha1.Task
	err = yaml.Unmarshal(content, &task)
	if err != nil {
		return nil, err
	}

	if task.Kind != "Task" {
		return nil, fmt.Errorf("provided kind %s instead of kind Task", task.Kind)
	}

	return &task, nil
}
