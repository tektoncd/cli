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
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/printer"
	validinput "github.com/tektoncd/cli/pkg/validate"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	versionedResource "github.com/tektoncd/pipeline/pkg/client/resource/clientset/versioned"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	msgNoPREsFound = "No pipelineresources found."
)

type ListOptions struct {
	Type string
}

func listCommand(p cli.Params) *cobra.Command {

	opts := &ListOptions{Type: ""}
	f := cliopts.NewPrintFlags("list")
	eg := `List all PipelineResources in a namespace 'foo':

    tkn pre list -n foo
`

	cmd := &cobra.Command{
		Use:          "list",
		Aliases:      []string{"ls"},
		Short:        "Lists pipeline resources in a namespace",
		Example:      eg,
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		Args: cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			if err := validinput.NamespaceExists(p); err != nil {
				return err
			}

			cs, err := p.Clients()
			if err != nil {
				return err
			}

			valid := false
			for _, allowed := range v1alpha1.AllResourceTypes {
				if string(allowed) == opts.Type || opts.Type == "" {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("failed to list pipelineresources. Invalid resource type %s", opts.Type)
			}

			pres, err := list(cs.Resource, p.Namespace(), opts.Type)
			stream := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to list pipelineresources from %s namespace \n", p.Namespace())
				return err
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				fmt.Fprint(os.Stderr, "Error: output option not set properly \n")
				return err
			}

			if output != "" {
				return printer.PrintObject(stream.Out, pres, f)
			}

			err = printFormatted(stream, pres)
			if err != nil {
				fmt.Fprint(os.Stderr, "Failed to print Pipelineresources \n")
				return err
			}
			return nil
		},
	}

	f.AddFlags(cmd)
	cmd.Flags().StringVarP(&opts.Type, "type", "t", "", "Pipeline resource type")

	return cmd
}

func list(client versionedResource.Interface, namespace string, resourceType string) (*v1alpha1.PipelineResourceList, error) {

	prec := client.TektonV1alpha1().PipelineResources(namespace)
	pres, err := prec.List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if resourceType != "" {
		pres.Items = filterByType(pres.Items, resourceType)
	}

	if len(pres.Items) > 0 {
		pres.Items = sortResourcesByTypeAndName(pres.Items)
	}

	// NOTE: this is required for -o json|yaml to work properly since
	// tektoncd go client fails to set these; probably a bug
	pres.GetObjectKind().SetGroupVersionKind(
		schema.GroupVersionKind{
			Version: "tekton.dev/v1alpha1",
			Kind:    "PipelineResourceList",
		})

	return pres, nil
}

func printFormatted(s *cli.Stream, pres *v1alpha1.PipelineResourceList) error {
	if len(pres.Items) == 0 {
		fmt.Fprintln(s.Err, msgNoPREsFound)
		return nil
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "NAME\tTYPE\tDETAILS")
	for _, pre := range pres.Items {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			pre.Name,
			pre.Spec.Type,
			details(pre),
		)
	}

	return w.Flush()
}

func details(pre v1alpha1.PipelineResource) string {
	var key = "url"
	if pre.Spec.Type == v1alpha1.PipelineResourceTypeStorage {
		key = "location"
	}
	if pre.Spec.Type == v1alpha1.PipelineResourceTypeCloudEvent {
		key = "targeturi"
	}

	for _, p := range pre.Spec.Params {
		if strings.ToLower(p.Name) == key {
			return p.Name + ": " + p.Value
		}
	}

	return "---"
}

func filterByType(resources []v1alpha1.PipelineResource, resourceType string) (ret []v1alpha1.PipelineResource) {
	for _, resource := range resources {
		if string(resource.Spec.Type) == resourceType {
			ret = append(ret, resource)
		}
	}
	return
}

// this will sort the Pipeline Resource by Type and then by Name
func sortResourcesByTypeAndName(pres []v1alpha1.PipelineResource) []v1alpha1.PipelineResource {
	sort.Slice(pres, func(i, j int) bool {
		if pres[j].Spec.Type < pres[i].Spec.Type {
			return false
		}

		if pres[j].Spec.Type > pres[i].Spec.Type {
			return true
		}

		return pres[j].Name > pres[i].Name
	})

	return pres
}
