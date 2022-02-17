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

package pipelinerun

import (
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/export"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/pipelinerun"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func exportCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("export")

	eg := `Export PipelineRun Definition:

	tkn pipelinerun export will export a pipelinerun definition as yaml to be easily
	imported or modified.

	Example: export a PipelineRun named 'pipelinerun' in namespace 'foo' and recreate
	it in the namespace 'bar':

    tkn pr export pipelinerun -n foo|kubectl create -f- -n bar
`

	c := &cobra.Command{
		Use:     "export",
		Short:   "Export PipelineRun",
		Example: eg,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &options.DescribeOptions{Params: p}
			if len(args) == 0 {
				pipelineRunNames, err := pipelinerun.GetAllPipelineRuns(p, metav1.ListOptions{}, 5)
				if err != nil {
					return err
				}
				if len(pipelineRunNames) == 1 {
					opts.PipelineRunName = strings.Fields(pipelineRunNames[0])[0]
				} else {
					err = askPipelineRunName(opts, pipelineRunNames)
					if err != nil {
						return err
					}
				}
			} else {
				opts.PipelineRunName = args[0]
			}

			err := exportPipelineRun(cmd.OutOrStdout(), p, opts.PipelineRunName)
			if err != nil {
				return err
			}
			return nil
		},
	}
	f.AddFlags(c)
	return c
}

func exportPipelineRun(out io.Writer, p cli.Params, pname string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	pipelinerun, err := pipelinerun.Get(cs, pname, metav1.GetOptions{}, p.Namespace())
	if err != nil {
		return err
	}
	exported, err := export.TektonResourceToYaml(pipelinerun)
	if err != nil {
		return err
	}
	_, err = out.Write([]byte(exported))
	return err
}
