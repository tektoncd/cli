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

package pipeline

import (
	"io"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/export"
	"github.com/tektoncd/cli/pkg/options"
	pipelinepkg "github.com/tektoncd/cli/pkg/pipeline"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/yaml"
)

func exportCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("export")

	eg := `Export Pipeline Definition:

	tkn pipeline export will export a pipeline definition as yaml to be easily
	reimported or modified.

	Example: export a Pipeline named 'pipeline' in namespace 'foo' and recreate
	it in the namespace 'bar':

    tkn p export pipeline -n foo|kubectl create -f- -n bar
`

	c := &cobra.Command{
		Use:     "export",
		Short:   "Export Pipeline",
		Example: eg,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &options.DescribeOptions{Params: p}
			cs, err := p.Clients()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				pipelineNames, err := pipelinepkg.GetAllPipelineNames(pipelineGroupResource, cs, p.Namespace())
				if err != nil {
					return err
				}
				if len(pipelineNames) == 1 {
					opts.PipelineName = pipelineNames[0]
				} else {
					err = askPipelineName(opts, pipelineNames)
					if err != nil {
						return err
					}
				}
			} else {
				opts.PipelineName = args[0]
			}

			err = exportPipeline(cmd.OutOrStdout(), cs, p.Namespace(), opts.PipelineName)
			if err != nil {
				return err
			}
			return nil
		},
	}
	f.AddFlags(c)
	return c
}

func exportPipeline(out io.Writer, c *cli.Clients, ns string, pName string) error {
	obj, err := actions.GetUnstructured(pipelineGroupResource, c, pName, ns, metav1.GetOptions{})
	if err != nil {
		return err
	}

	err = export.RemoveFieldForExport(obj)
	if err != nil {
		return err
	}

	obj.SetKind("Pipeline")

	data, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	_, err = out.Write(data)
	return err
}
