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

package condition

import (
	"context"
	"fmt"
	"text/tabwriter"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/printer"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const templ = `{{decorate "bold" "Name"}}:	{{ .Condition.Name }}
{{decorate "bold" "Namespace"}}:	{{ .Condition.Namespace }}
{{- if ne .Condition.Spec.Description "" }}
{{decorate "bold" "Description"}}:	{{ .Condition.Spec.Description }}
{{- end }}

{{ decorate "check" ""}}{{decorate "underline bold" "Check\n"}}
 {{decorate "bold" "Name"}}:	{{ .Condition.Spec.Check.Name }}
 {{decorate "bold" "Image"}}:	{{ .Condition.Spec.Check.Image }}
 {{- if .Condition.Spec.Check.Command }}
 {{decorate "bold" "Command"}}:	{{ .Condition.Spec.Check.Command }}
 {{- end }}
 {{- if .Condition.Spec.Check.Args }}
 {{decorate "bold" "Args"}}:	{{ .Condition.Spec.Check.Args }}
 {{- end }}
 {{- if ne .Condition.Spec.Check.Script "" }}
 {{decorate "bold" "Script"}}:	{{ .Condition.Spec.Check.Script }}
 {{- end }}

{{decorate "resources" ""}}{{decorate "underline bold" "Resources\n"}}
{{- $rl := len .Condition.Spec.Resources }}{{ if eq $rl 0 }}
 No resources
{{- else }}
 NAME	TYPE
{{- range $r := .Condition.Spec.Resources }}
 {{decorate "bullet" $r.Name }}	{{ $r.Type }}
{{- end }}
{{- end }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}
{{- $l := len .Condition.Spec.Params }}{{ if eq $l 0 }}
 No params
{{- else }}
 NAME	TYPE	DESCRIPTION	DEFAULT VALUE
{{- range $p := .Condition.Spec.Params }}
{{- if not $p.Default }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ formatDesc $p.Description }}	{{ "---" }}
{{- else }}
{{- if eq $p.Type "string" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ formatDesc $p.Description }}	{{ $p.Default.StringVal }}
{{- else }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ formatDesc $p.Description }}	{{ $p.Default.ArrayVal }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
`

func describeCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("describe")
	eg := `Describe a Condition of name 'foo' in namespace 'bar':

    tkn condition describe foo -n bar

or

    tkn cond desc foo -n bar
`

	c := &cobra.Command{
		Use:               "describe",
		Aliases:           []string{"desc"},
		Short:             "Describe Conditions in a namespace",
		Example:           eg,
		ValidArgsFunction: formatted.ParentCompletion,
		Args:              cobra.MinimumNArgs(1),
		SilenceUsage:      true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}

			condition, err := getCondition(p, args[0])
			if err != nil {
				return err
			}

			if output != "" {
				if err := printer.PrintObject(cmd.OutOrStdout(), condition, f); err != nil {
					return err
				}
				return nil
			}

			return printConditionDescription(s, p, condition)
		},
	}

	f.AddFlags(c)
	return c
}

func getCondition(p cli.Params, name string) (*v1alpha1.Condition, error) {
	cs, err := p.Clients()
	if err != nil {
		return nil, fmt.Errorf("failed to create tekton client")
	}

	condition, err := cs.Tekton.TektonV1alpha1().Conditions(p.Namespace()).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to find Condition %q", name)
	}

	// NOTE: this is required for -o json|yaml to work properly since
	// tektoncd go client fails to set these; probably a bug
	condition.GetObjectKind().SetGroupVersionKind(
		schema.GroupVersionKind{
			Version: "tekton.dev/v1alpha1",
			Kind:    "Condition",
		})

	return condition, nil
}

func printConditionDescription(s *cli.Stream, p cli.Params, condition *v1alpha1.Condition) error {
	var data = struct {
		Condition *v1alpha1.Condition
		Params    cli.Params
	}{
		Condition: condition,
		Params:    p,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	FuncMap := template.FuncMap{
		"decorate":   formatted.DecorateAttr,
		"formatDesc": formatted.FormatDesc,
	}
	t := template.Must(template.New("Describe Condition").Funcs(FuncMap).Parse(templ))

	if err := t.Execute(w, data); err != nil {
		fmt.Fprintf(s.Err, "Failed to execute template")
		return err
	}
	return w.Flush()
}
