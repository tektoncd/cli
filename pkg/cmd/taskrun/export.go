// Copyright © 2022 The Tekton Authors.
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
	"io"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/export"
	"github.com/tektoncd/cli/pkg/options"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/yaml"
)

func exportCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("export")

	eg := `Export Taskrun Definition:

	"tkn taskrun export" will export a taskrun definition as yaml to be easily
	imported or modified.

	Example: export a TaskRun named 'taskrun' in namespace 'foo' and recreate
	it in the namespace 'bar':

    tkn taskrun export taskrun -n foo|kubectl create -f- -n bar
`

	cmd := &cobra.Command{
		Use:     "export",
		Short:   "Export TaskRun",
		Example: eg,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &options.LogOptions{Params: p}
			if len(args) == 0 {
				err := askRunName(opts)
				if err != nil {
					return err
				}
			} else {
				opts.TaskrunName = args[0]
			}
			err := exportTaskRun(cmd.OutOrStdout(), p, opts.TaskrunName)
			if err != nil {
				return err
			}
			return nil
		},
	}
	f.AddFlags(cmd)
	return cmd
}

func exportTaskRun(out io.Writer, p cli.Params, trName string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	obj, err := actions.GetUnstructured(taskrunGroupResource, cs, trName, p.Namespace(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	err = export.RemoveFieldForExport(obj)
	if err != nil {
		return err
	}

	obj.SetKind("TaskRun")

	data, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	_, err = out.Write(data)
	return err
}
