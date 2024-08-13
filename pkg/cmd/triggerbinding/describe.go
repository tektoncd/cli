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
	"fmt"
	"text/tabwriter"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/triggerbinding"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const describeTemplate = `{{decorate "bold" "Name"}}:	{{ .TriggerBinding.Name }}
{{decorate "bold" "Namespace"}}:	{{ .TriggerBinding.Namespace }}

{{- if ne (len .TriggerBinding.Spec.Params) 0 }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}
 NAME	VALUE
{{- range $p := .TriggerBinding.Spec.Params }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value }}
{{- end }}
{{- end }}
`

func describeCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("describe")
	opts := &options.DescribeOptions{Params: p}
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
		SilenceUsage:      true,
		ValidArgsFunction: formatted.ParentCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			cs, err := p.Clients()
			if err != nil {
				return err
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}

			if len(args) == 0 {
				tb, err := triggerbinding.GetAllTriggerBindingNames(cs, p.Namespace())
				if err != nil {
					return err
				}
				if len(tb) == 1 {
					opts.TriggerBindingName = tb[0]
				} else {
					err = askTriggerBindingName(opts, tb)
					if err != nil {
						return err
					}
				}
			} else {
				opts.TriggerBindingName = args[0]
			}

			if output != "" {
				printer, err := f.ToPrinter()
				if err != nil {
					return err
				}
				return actions.PrintObject(triggerbindingGroupResource, opts.TriggerBindingName, cmd.OutOrStdout(), cs.Dynamic, cs.Triggers.Discovery(), printer, p.Namespace())
			}

			return printTriggerBindingDescription(s, p, opts.TriggerBindingName)
		},
	}

	f.AddFlags(c)
	return c
}

func printTriggerBindingDescription(s *cli.Stream, p cli.Params, tbName string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	tb, err := triggerbinding.Get(cs, tbName, metav1.GetOptions{}, p.Namespace())
	if err != nil {
		return fmt.Errorf("failed to get TriggerBinding %s from %s namespace: %v", tbName, p.Namespace(), err)
	}

	var data = struct {
		TriggerBinding *v1beta1.TriggerBinding
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

func askTriggerBindingName(opts *options.DescribeOptions, tb []string) error {
	if len(tb) == 0 {
		return fmt.Errorf("no TriggerBindings found")
	}
	err := opts.Ask(options.ResourceNameTriggerBinding, tb)
	if err != nil {
		return err
	}

	return nil
}
