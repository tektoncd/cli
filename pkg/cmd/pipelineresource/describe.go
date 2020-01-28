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
	"io"
	"os"
	"text/tabwriter"
	"text/template"

	"github.com/tektoncd/cli/pkg/formatted"
	validateinput "github.com/tektoncd/cli/pkg/helper/validate"
	"github.com/tektoncd/cli/pkg/printer"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const templ = `{{decorate "bold" "Name"}}:	{{ .PipelineResource.Name }}
{{decorate "bold" "Namespace"}}:	{{ .PipelineResource.Namespace }}
{{decorate "bold" "PipelineResource Type"}}:	{{ .PipelineResource.Spec.Type }}

{{decorate "underline bold" "Params\n"}}
{{- $l := len .PipelineResource.Spec.Params }}{{ if eq $l 0 }}
 No params
{{- else }}
 NAME	VALUE
{{- range $i, $p := .PipelineResource.Spec.Params }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value }}
{{- end }}
{{- end }}

{{decorate "underline bold" "Secret Params\n"}}
{{- $l := len .PipelineResource.Spec.SecretParams }}{{ if eq $l 0 }}
 No secret params
{{- else }}
 FIELDNAME	SECRETNAME
{{- range $i, $p := .PipelineResource.Spec.SecretParams }}
 {{decorate "bullet" $p.FieldName }}	{{ $p.SecretName }}
{{- end }}
{{- end }}
`

func describeCommand(p cli.Params) *cobra.Command {
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
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if err := validateinput.NamespaceExists(p); err != nil {
				return err
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				fmt.Fprint(os.Stderr, "Error: output option not set properly \n")
				return err
			}

			if output != "" {
				return describePipelineResourceOutput(cmd.OutOrStdout(), p, f, args[0])
			}

			return printPipelineResourceDescription(s, p, args[0])
		},
	}

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_pipelineresource")
	f.AddFlags(c)
	return c
}

func describePipelineResourceOutput(w io.Writer, p cli.Params, f *cliopts.PrintFlags, name string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	c := cs.Tekton.TektonV1alpha1().PipelineResources(p.Namespace())

	pipelineresource, err := c.Get(name, metav1.GetOptions{})
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

	pre, err := cs.Tekton.TektonV1alpha1().PipelineResources(p.Namespace()).Get(preName, metav1.GetOptions{})
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
