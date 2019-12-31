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

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	validate "github.com/tektoncd/cli/pkg/helper/validate"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

type applyOptions struct {
	from string
}

func applyCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("apply")
	opts := &applyOptions{from: ""}
	eg := `Create a pipeline or update already existing pipeline defined by foo.yaml in namespace 'bar':
    tkn pipeline apply -f foo.yaml -n bar
`

	c := &cobra.Command{
		Use:          "apply",
		Short:        "Create or update a pipeline in a namespace",
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

			return applyPipeline(s, p, opts.from)
		},
	}
	f.AddFlags(c)
	c.Flags().StringVarP(&opts.from, "from", "f", "", "local or remote filename to use to create or update a pipeline")
	return c
}

func applyPipeline(s *cli.Stream, p cli.Params, path string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	//loadPipeline defined in create.go
	pipeline, err := loadPipeline(p, path)
	if err != nil {
		return err
	}

	create := false
	//check if Pipeline already exists
	pipelineExistCheck, err := cs.Tekton.TektonV1alpha1().Pipelines(p.Namespace()).Get(pipeline.Name, v1.GetOptions{})
	if err != nil {
		if err.Error() == "pipelines.tekton.dev \""+pipeline.Name+"\" not found" {
			create = true
		} else {
			return err
		}
	}

	//Pipeline does not exist
	if create {
		_, err = cs.Tekton.TektonV1alpha1().Pipelines(p.Namespace()).Create(pipeline)
		if err != nil {
			return fmt.Errorf("failed to create Pipeline %q: %s", pipeline.Name, err)
		}

		fmt.Fprintf(s.Out, "Pipeline created: %s\n", pipeline.Name)
		return nil
	}

	//Pipeline exists
	//Update Pipeline's resourceVersion based on already existing Pipeline
	pipeline.ResourceVersion = pipelineExistCheck.ResourceVersion
	_, err = cs.Tekton.TektonV1alpha1().Pipelines(p.Namespace()).Update(pipeline)
	if err != nil {
		return fmt.Errorf("failed to update Pipeline %q: %s", pipeline.Name, err)
	}

	fmt.Fprintf(s.Out, "Pipeline updated: %s\n", pipeline.Name)
	return nil
}
