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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"text/tabwriter"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/printer"
	"github.com/tektoncd/cli/pkg/validate"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const describeTemplate = `{{decorate "bold" "Name"}}:	{{ .TriggerTemplate.Name }}
{{decorate "bold" "Namespace"}}:	{{ .TriggerTemplate.Namespace }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}

{{- if eq (len .TriggerTemplate.Spec.Params) 0 }}
 No params
{{- else }}
 NAME	TYPE	DESCRIPTION	DEFAULT VALUE
{{- range $p := .TriggerTemplate.Spec.Params }}
{{- if not $p.Default }}
{{- if eq $p.Type "" }}
 {{decorate "bullet" $p.Name }}	{{ "string" }}	{{ $p.Description }}	{{ "---" }}
{{- else }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ $p.Description }}	{{ "---" }} 
{{ end }}
{{- else }}
{{- if eq $p.Default.Type "string" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Default.Type }}	{{ $p.Description }}	{{ $p.Default.StringVal }}
{{- else }}
 {{decorate "bullet" $p.Name }}	{{ $p.Default.Type }}	{{ $p.Description }}	{{ $p.Default.ArrayVal }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{decorate "resources" ""}}{{decorate "underline bold" "ResourceTemplates\n"}}

{{- $value := getResourceTemplate .TriggerTemplate }}
{{-  if ne $value "" }}
{{ $value }}
{{- end }}
`

func describeCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("describe")
	eg := `Describe a TriggerTemplate of name 'foo' in namespace 'bar':

    tkn triggertemplate describe foo -n bar

or

   tkn tt desc foo -n bar
`

	c := &cobra.Command{
		Use:     "describe",
		Aliases: []string{"desc"},
		Short:   "Describes a triggertemplate in a namespace",
		Example: eg,
		Annotations: map[string]string{
			"commandType": "main",
		},
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if err := validate.NamespaceExists(p); err != nil {
				return err
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				fmt.Fprint(os.Stderr, "Error: output option not set properly \n")
				return err
			}

			if output != "" {
				return describeTriggerTemplateOutput(cmd.OutOrStdout(), p, f, args[0])
			}

			return printTriggerTemplateDescription(s, p, args[0])
		},
	}

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_triggertemplate")
	f.AddFlags(c)
	return c
}

func describeTriggerTemplateOutput(w io.Writer, p cli.Params, f *cliopts.PrintFlags, name string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	tt, err := cs.Triggers.TriggersV1alpha1().TriggerTemplates(p.Namespace()).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// NOTE: this is required for -o json|yaml to work properly since
	// tektoncd go client fails to set these; probably a bug
	tt.GetObjectKind().SetGroupVersionKind(
		schema.GroupVersionKind{
			Version: "triggers.tekton.dev",
			Kind:    "TriggerTemplate",
		})

	return printer.PrintObject(w, tt, f)
}

func printTriggerTemplateDescription(s *cli.Stream, p cli.Params, ttname string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	tt, err := cs.Triggers.TriggersV1alpha1().TriggerTemplates(p.Namespace()).Get(ttname, metav1.GetOptions{})
	if err != nil {
		fmt.Fprintf(s.Err, "failed to get triggertemplate %s\n", ttname)
		return err
	}

	var data = struct {
		TriggerTemplate *v1alpha1.TriggerTemplate
	}{
		TriggerTemplate: tt,
	}

	funcMap := template.FuncMap{
		"decorate":            formatted.DecorateAttr,
		"getResourceTemplate": getResourceTemplate,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	tparsed := template.Must(template.New("Describe TriggerTemplate").Funcs(funcMap).Parse(describeTemplate))
	if err = tparsed.Execute(w, data); err != nil {
		fmt.Fprintf(s.Err, "Failed to execute template \n")
		return err
	}
	return w.Flush()
}

func getResourceTemplate(triggerTemplate v1alpha1.TriggerTemplate) string {
	var out bytes.Buffer
	resourceTemplate, err := json.Marshal(triggerTemplate.Spec.ResourceTemplates)
	if err != nil {
		log.Printf("json marshal failed with error: %v", err)
		return ""
	}
	_ = json.Indent(&out, resourceTemplate, "", "    ")
	return out.String()
}
