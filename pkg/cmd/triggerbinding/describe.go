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

package triggerbinding

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/printer"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const describeTemplate = `{{decorate "bold" "Name"}}:	{{ .TriggerBinding.Name }}
{{decorate "bold" "Namespace"}}:	{{ .TriggerBinding.Namespace }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}

{{- if eq (len .TriggerBinding.Spec.Params) 0 }}
 No params
{{- else }}
 NAME	VALUE
{{- range $p := .TriggerBinding.Spec.Params }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value }}
{{- end }}
{{- end }}
`

func describeCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("describe")
	eg := `Describe a TriggerBinding of name 'foo' in namespace 'bar':

    tkn triggerbinding describe foo -n bar

or

    tkn tb desc foo -n bar
`

	c := &cobra.Command{
		Use:     "describe",
		Aliases: []string{"desc"},
		Short:   "Describes a TriggerBinding in a namespace",
		Example: eg,
		Annotations: map[string]string{
			"commandType": "main",
		},
		Args:              cobra.MinimumNArgs(1),
		SilenceUsage:      true,
		ValidArgsFunction: formatted.ParentCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}

			if output != "" {
				return describeTriggerBindingOutput(cmd.OutOrStdout(), p, f, args[0])
			}

			return printTriggerBindingDescription(s, p, args[0])
		},
	}

	f.AddFlags(c)
	return c
}

func describeTriggerBindingOutput(w io.Writer, p cli.Params, f *cliopts.PrintFlags, name string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	tb, err := cs.Triggers.TriggersV1alpha1().TriggerBindings(p.Namespace()).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// NOTE: this is required for -o json|yaml to work properly since
	// tektoncd go client fails to set these; probably a bug
	tb.GetObjectKind().SetGroupVersionKind(
		schema.GroupVersionKind{
			Version: "triggers.tekton.dev/v1alpha1",
			Kind:    "TriggerBinding",
		})

	return printer.PrintObject(w, tb, f)
}

func printTriggerBindingDescription(s *cli.Stream, p cli.Params, tbName string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	tb, err := cs.Triggers.TriggersV1alpha1().TriggerBindings(p.Namespace()).Get(context.Background(), tbName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get TriggerBinding %s from %s namespace: %v", tbName, p.Namespace(), err)
	}

	var data = struct {
		TriggerBinding *v1alpha1.TriggerBinding
	}{
		TriggerBinding: tb,
	}

	funcMap := template.FuncMap{
		"decorate": formatted.DecorateAttr,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	tparsed := template.Must(template.New("Describe Triggerbinding").Funcs(funcMap).Parse(describeTemplate))
	if err = tparsed.Execute(w, data); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}
	return w.Flush()
}
