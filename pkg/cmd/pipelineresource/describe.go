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
	"context"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/printer"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const templ = `{{decorate "bold" "Name"}}:	{{ .PipelineResource.Name }}
{{decorate "bold" "Namespace"}}:	{{ .PipelineResource.Namespace }}
{{decorate "bold" "PipelineResource Type"}}:	{{ .PipelineResource.Spec.Type }}
{{- if ne .PipelineResource.Spec.Description "" }}
{{decorate "bold" "Description"}}:	{{ .PipelineResource.Spec.Description }}
{{- end }}

{{- if ne (len .PipelineResource.Spec.Params) 0 }}

{{decorate "underline bold" "Params\n"}}
 NAME	VALUE
{{- range $i, $p := .PipelineResource.Spec.Params }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value }}
{{- end }}
{{- end }}

{{- if ne (len .PipelineResource.Spec.SecretParams) 0 }}

{{decorate "underline bold" "Secret Params\n"}}
 FIELDNAME	SECRETNAME
{{- range $i, $p := .PipelineResource.Spec.SecretParams }}
 {{decorate "bullet" $p.FieldName }}	{{ $p.SecretName }}
{{- end }}
{{- end }}
`

func describeCommand(p cli.Params) *cobra.Command {
	opts := &options.DescribeOptions{Params: p}
	f := cliopts.NewPrintFlags("describe")
	eg := `Describe a PipelineResource of name 'foo' in namespace 'bar':

    tkn resource describe foo -n bar

or

    tkn res desc foo -n bar
`

	c := &cobra.Command{
		Use:          "describe",
		Aliases:      []string{"desc"},
		Short:        "Describes a pipeline resource in a namespace",
		Example:      eg,
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		ValidArgsFunction: formatted.ParentCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				fmt.Fprint(os.Stderr, "Error: output option not set properly \n")
				return err
			}

			if len(args) == 0 {
				err = askPipelineResourceName(opts, p)
				if err != nil {
					return err
				}
			} else {
				opts.PipelineResourceName = args[0]
			}

			if output != "" {
				return describePipelineResourceOutput(cmd.OutOrStdout(), p, f, opts.PipelineResourceName)
			}

			return printPipelineResourceDescription(s, p, opts.PipelineResourceName)
		},
	}

	f.AddFlags(c)
	c.Deprecated = "PipelineResource commands are deprecated, they will be removed soon as it get removed from API."

	return c
}

func describePipelineResourceOutput(w io.Writer, p cli.Params, f *cliopts.PrintFlags, name string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	c := cs.Resource.TektonV1alpha1().PipelineResources(p.Namespace())

	pipelineresource, err := c.Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// NOTE: this is required for -o json|yaml to work properly since
	// tektoncd go client fails to set these; probably a bug
	pipelineresource.GetObjectKind().SetGroupVersionKind(
		schema.GroupVersionKind{
			Version: "tekton.dev/v1alpha1",
			Kind:    "PipelineResource",
		})

	return printer.PrintObject(w, pipelineresource, f)
}

func printPipelineResourceDescription(s *cli.Stream, p cli.Params, preName string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	pre, err := cs.Resource.TektonV1alpha1().PipelineResources(p.Namespace()).Get(context.Background(), preName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to find pipelineresource %q", preName)
	}

	var data = struct {
		PipelineResource *v1alpha1.PipelineResource
		Params           cli.Params
	}{
		PipelineResource: pre,
		Params:           p,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	FuncMap := template.FuncMap{
		"decorate": formatted.DecorateAttr,
	}
	t := template.Must(template.New("Describe PipelineResource").Funcs(FuncMap).Parse(templ))

	err = t.Execute(w, data)
	if err != nil {
		fmt.Fprintf(s.Err, "Failed to execute template")
		return err
	}
	return w.Flush()
}

func askPipelineResourceName(opts *options.DescribeOptions, p cli.Params) error {
	cliClients, err := p.Clients()
	if err != nil {
		return err
	}

	// the `allPipelineResourceNames( )` func is in delete.go, in the pipelineresource package.
	pipelineResourceNames, err := allPipelineResourceNames(p, cliClients)
	if err != nil {
		return err
	}
	if len(pipelineResourceNames) == 0 {
		return fmt.Errorf("no pipelineresources found")
	}

	err = opts.Ask(options.ResourceNamePipelineResource, pipelineResourceNames)
	if err != nil {
		return err
	}

	return nil
}
