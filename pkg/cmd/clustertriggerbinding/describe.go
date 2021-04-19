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

package clustertriggerbinding

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/clustertriggerbinding"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/printer"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const describeTemplate = `{{decorate "bold" "Name"}}:	{{ .ClusterTriggerBinding.Name }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}

{{- if eq (len .ClusterTriggerBinding.Spec.Params) 0 }}
 No params
{{- else }}
 NAME	VALUE
{{- range $p := .ClusterTriggerBinding.Spec.Params }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value }}
{{- end }}
{{- end }}
`

func describeCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("describe")
	opts := &options.DescribeOptions{Params: p}
	eg := `Describe a ClusterTriggerBinding of name 'foo':

    tkn clustertriggerbinding describe foo

or

    tkn ctb desc foo
`

	c := &cobra.Command{
		Use:               "describe",
		Aliases:           []string{"desc"},
		Short:             "Describes a ClusterTriggerBinding",
		Example:           eg,
		ValidArgsFunction: formatted.ParentCompletion,
		Annotations: map[string]string{
			"commandType": "main",
		},
		SilenceUsage: true,
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
				ctb, err := clustertriggerbinding.GetAllClusterTriggerBindingNames(cs.Triggers, p.Namespace())
				if err != nil {
					return err
				}
				if len(ctb) == 1 {
					opts.ClusterTriggerBindingName = ctb[0]
				} else {
					err = askClusterTriggerBindingName(opts, ctb)
					if err != nil {
						return err
					}
				}
			} else {
				opts.ClusterTriggerBindingName = args[0]
			}

			if output != "" {
				return describeClusterTriggerBindingOutput(cmd.OutOrStdout(), p, f, args[0])
			}

			return printClusterTriggerBindingDescription(s, p, opts.ClusterTriggerBindingName)
		},
	}

	f.AddFlags(c)
	return c
}

func describeClusterTriggerBindingOutput(w io.Writer, p cli.Params, f *cliopts.PrintFlags, name string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	ctb, err := cs.Triggers.TriggersV1alpha1().ClusterTriggerBindings().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// NOTE: this is required for -o json|yaml to work properly since
	// tektoncd go client fails to set these; probably a bug
	ctb.GetObjectKind().SetGroupVersionKind(
		schema.GroupVersionKind{
			Version: "triggers.tekton.dev/v1alpha1",
			Kind:    "ClusterTriggerBinding",
		})

	return printer.PrintObject(w, ctb, f)
}

func printClusterTriggerBindingDescription(s *cli.Stream, p cli.Params, ctbName string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	ctb, err := cs.Triggers.TriggersV1alpha1().ClusterTriggerBindings().Get(context.Background(), ctbName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ClusterTriggerBinding %s: %v", ctbName, err)
	}

	var data = struct {
		ClusterTriggerBinding *v1alpha1.ClusterTriggerBinding
	}{
		ClusterTriggerBinding: ctb,
	}

	funcMap := template.FuncMap{
		"decorate": formatted.DecorateAttr,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	tparsed := template.Must(template.New("Describe ClusterTriggerbinding").Funcs(funcMap).Parse(describeTemplate))
	if err = tparsed.Execute(w, data); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}
	return w.Flush()
}

func askClusterTriggerBindingName(opts *options.DescribeOptions, ctb []string) error {
	if len(ctb) == 0 {
		return fmt.Errorf("no ClusterTriggerBindings found")
	}
	err := opts.Ask(options.ResourceNameClusterTriggerBinding, ctb)
	if err != nil {
		return err
	}

	return nil
}
