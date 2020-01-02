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

package pipelineresource

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	validateinput "github.com/tektoncd/cli/pkg/helper/validate"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

type applyOptions struct {
	from string
}

func applyCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("apply")
	opts := &applyOptions{from: ""}
	eg := `Create a pipeline resource or update already existing pipeline resource defined by foo.yaml in namespace 'bar':

    tkn resource apply -f foo.yaml -n bar
`

	c := &cobra.Command{
		Use:          "apply",
		Short:        "Create or update a pipeline resource in a namespace",
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

			if err := validateinput.NamespaceExists(p); err != nil {
				return err
			}

			return applyPipelineResource(s, p, opts.from)
		},
	}
	f.AddFlags(c)
	c.Flags().StringVarP(&opts.from, "from", "f", "", "local or remote filename to use to create or update a pipeline resource")
	return c
}

func applyPipelineResource(s *cli.Stream, p cli.Params, path string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	//loadResource defined in create.go
	resource, err := loadResource(p, path)
	if err != nil {
		return err
	}

	create := false
	//check if PipelineResource already exists
	resourceExistCheck, err := cs.Tekton.TektonV1alpha1().PipelineResources(p.Namespace()).Get(resource.Name, v1.GetOptions{})
	if err != nil {
		if err.Error() == "pipelineresources.tekton.dev \""+resource.Name+"\" not found" {
			create = true
		} else {
			return err
		}
	}

	//PipelineResource does not exist
	if create {
		_, err = cs.Tekton.TektonV1alpha1().PipelineResources(p.Namespace()).Create(resource)
		if err != nil {
			return fmt.Errorf("failed to create PipelineResource %q: %s", resource.Name, err)
		}

		fmt.Fprintf(s.Out, "PipelineResource created: %s\n", resource.Name)
		return nil
	}

	//PipelineResource exists
	//Update PipelineResource's resourceVersion based on already existing PipelineResource
	resource.ResourceVersion = resourceExistCheck.ResourceVersion
	_, err = cs.Tekton.TektonV1alpha1().PipelineResources(p.Namespace()).Update(resource)
	if err != nil {
		return fmt.Errorf("failed to update PipelineResource %q: %s", resource.Name, err)
	}

	fmt.Fprintf(s.Out, "PipelineResource updated: %s\n", resource.Name)
	return nil
}
