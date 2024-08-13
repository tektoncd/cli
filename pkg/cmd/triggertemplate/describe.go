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

package triggertemplate

import (
	"fmt"
	"text/tabwriter"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/triggertemplate"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const describeTemplate = `{{decorate "bold" "Name"}}:	{{ .TriggerTemplate.Name }}
{{decorate "bold" "Namespace"}}:	{{ .TriggerTemplate.Namespace }}

{{- if ne (len .TriggerTemplate.Spec.Params) 0 }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}
 NAME	DESCRIPTION	DEFAULT VALUE
{{- range $p := .TriggerTemplate.Spec.Params }}
{{- if not $p.Default }}
 {{decorate "bullet" $p.Name }}	{{ formatDesc $p.Description }}	{{ "---" }}
{{- else }}
 {{decorate "bullet" $p.Name }}	{{ formatDesc $p.Description }}	{{ $p.Default }}
{{- end }}
{{- end }}
{{- end }}

{{- $errValue := checkError .TriggerTemplate.Spec.ResourceTemplates }}
{{- if ne $errValue "" }}

 {{ $errValue }}
{{- else }}

{{decorate "resources" ""}}{{decorate "underline bold" "ResourceTemplates\n"}}
 NAME	GENERATENAME	KIND	APIVERSION
{{- range $p := .TriggerTemplate.Spec.ResourceTemplates }}
{{- $value := getResourceTemplate $p }}
 {{ format $value.GetName | decorate "bullet" }}	{{ format $value.GetGenerateName }}	{{ $value.GetKind }}	{{ $value.GetAPIVersion }}
{{- end }}
{{- end }}
`

func describeCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("describe")
	opts := &options.DescribeOptions{Params: p}
	eg := `Describe a TriggerTemplate of name 'foo' in namespace 'bar':

    tkn triggertemplate describe foo -n bar

or

   tkn tt desc foo -n bar
`

	c := &cobra.Command{
		Use:     "describe",
		Aliases: []string{"desc"},
		Short:   "Describes a TriggerTemplate in a namespace",
		Example: eg,
		Annotations: map[string]string{
			"commandType": "main",
		},
		ValidArgsFunction: formatted.ParentCompletion,
		SilenceUsage:      true,
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
				tt, err := triggertemplate.GetAllTriggerTemplateNames(cs, p.Namespace())
				if err != nil {
					return err
				}
				if len(tt) == 1 {
					opts.TriggerTemplateName = tt[0]
				} else {
					err = askTriggerTemplateName(opts, tt)
					if err != nil {
						return err
					}
				}
			} else {
				opts.TriggerTemplateName = args[0]
			}

			if output != "" {
				printer, err := f.ToPrinter()
				if err != nil {
					return err
				}
				return actions.PrintObject(triggertemplateGroupResource, opts.TriggerTemplateName, cmd.OutOrStdout(), cs.Dynamic, cs.Triggers.Discovery(), printer, p.Namespace())
			}

			return printTriggerTemplateDescription(s, p, opts.TriggerTemplateName)
		},
	}

	f.AddFlags(c)
	return c
}

func printTriggerTemplateDescription(s *cli.Stream, p cli.Params, ttname string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	tt, err := triggertemplate.Get(cs, ttname, metav1.GetOptions{}, p.Namespace())
	if err != nil {
		return fmt.Errorf("failed to get TriggerTemplate %s from %s namespace: %v", ttname, p.Namespace(), err)
	}

	var data = struct {
		TriggerTemplate *v1beta1.TriggerTemplate
	}{
		TriggerTemplate: tt,
	}

	funcMap := template.FuncMap{
		"decorate":            formatted.DecorateAttr,
		"formatDesc":          formatted.FormatDesc,
		"checkError":          checkError,
		"getResourceTemplate": getResourceTemplate,
		"format":              format,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	tparsed := template.Must(template.New("Describe TriggerTemplate").Funcs(funcMap).Parse(describeTemplate))
	if err = tparsed.Execute(w, data); err != nil {
		fmt.Fprintf(s.Err, "Failed to execute template \n")
		return err
	}
	return w.Flush()
}

func checkError(resourceTemplate []v1beta1.TriggerResourceTemplate) string {
	errValue := ""
	for i := range resourceTemplate {
		if _, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&resourceTemplate[i]); err != nil {
			errValue = err.Error()
		}
	}
	return errValue
}

func getResourceTemplate(resourceTemplate v1beta1.TriggerResourceTemplate) *unstructured.Unstructured {
	d, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&resourceTemplate)
	return &unstructured.Unstructured{Object: d}
}

func format(name string) string {
	if name == "" {
		return "---"
	}
	return name
}

func askTriggerTemplateName(opts *options.DescribeOptions, tt []string) error {
	if len(tt) == 0 {
		return fmt.Errorf("no TriggerTemplate found")
	}

	err := opts.Ask(options.ResourceNameTriggerTemplate, tt)
	if err != nil {
		return err
	}

	return nil
}
