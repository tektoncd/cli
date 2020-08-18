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

package eventlistener

import (
	"fmt"
	"io"
	"os"
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
	"sigs.k8s.io/yaml"
)

const describeTemplate = `{{decorate "bold" "Name"}}:	{{ .EventListener.Name }}
{{decorate "bold" "Namespace"}}:	{{ .EventListener.Namespace }}

{{- if ne .EventListener.Spec.ServiceAccountName "" }}
{{decorate "bold" "Service Account"}}:	{{ .EventListener.Spec.ServiceAccountName }}
{{- end }}

{{- if ne .EventListener.Spec.ServiceType "" }}
{{decorate "bold" "Service Type"}}:	{{ .EventListener.Spec.ServiceType }}
{{- end }}

{{- $value := getURL .EventListener }}
{{-  if ne $value "" }}
{{decorate "bold" "URL"}}:	{{ $value }}
{{- end }}

{{- $value := getEventListenerName .EventListener }}
{{-  if ne $value "" }}
{{decorate "bold" "EventListnerServiceName"}}:	{{ $value }}
{{- end }}

{{- if eq (len .EventListener.Spec.Triggers) 0 }}

 No EventListenerTriggers
{{- else }}
{{ " " }}
{{- range $v := .EventListener.Spec.Triggers }}
{{decorate "bold" "EventListenerTriggers\n"}}

{{- if eq $v.Name "" }}
{{- else }}
 NAME
 {{decorate "bullet" $v.Name }}
{{ " " }}
{{- end }}

{{- if ne (len $v.Bindings) 0 }}
 {{decorate "" "BINDINGS\n"}}
{{- if isBindingRefExist $v.Bindings }}
  REF	KIND	APIVERSION
{{- range $b := $v.Bindings }}
{{- if ne $b.Ref "" }}
  {{ decorate "bullet" $b.Ref }}	{{ $b.Kind }}	{{ $b.APIVersion }}
{{- end }}
{{- end }}
{{ " " }}
{{- end }}

{{- if isBindingSpecExist $v.Bindings }}
  SPEC
{{- range $b := $v.Bindings }}
{{- if eq $b.Ref "" }}
  {{ decorate "bullet" "Params" }}
{{- range $p := $b.Spec.Params }}
    {{ $p.Name }}	{{ $p.Value }}
{{- end }}
{{- end }}
{{- end }}
{{ " " }}
{{- end }}
{{- end }}
 TEMPLATE NAME	APIVERSION
 {{ decorate "bullet" $v.Template.Name }}	{{ $v.Template.APIVersion }}
{{- if not $v.ServiceAccount }}
{{- else }}
{{ " " }}
 SERVICE ACCOUNT NAME	NAMESPACE
 {{ decorate "bullet" $v.ServiceAccount.Name }}	{{ $v.ServiceAccount.Namespace }}
{{- end }}
{{ " " }}
{{- if ne (len $v.Interceptors) 0 }}
 INTERCEPTORS
{{- $b := getInterceptors $v.Interceptors }}
{{ $b }}
{{- end }}
{{- end }}
{{- end }}
`

func describeCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("describe")
	eg := `Describe an EventListener of name 'foo' in namespace 'bar':

    tkn eventlistener describe foo -n bar

or

   tkn el desc foo -n bar
`

	c := &cobra.Command{
		Use:     "describe",
		Aliases: []string{"desc"},
		Short:   "Describe EventListener in a namespace",
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

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				fmt.Fprint(os.Stderr, "Error: output option not set properly \n")
				return err
			}

			if output != "" {
				return describeEventListenerOutput(cmd.OutOrStdout(), p, f, args[0])
			}

			return printEventListenerDescription(s, p, args[0])
		},
	}

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_eventlistener")
	f.AddFlags(c)
	return c
}

func describeEventListenerOutput(w io.Writer, p cli.Params, f *cliopts.PrintFlags, name string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	el, err := cs.Triggers.TriggersV1alpha1().EventListeners(p.Namespace()).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// NOTE: this is required for -o json|yaml to work properly since
	// tektoncd go client fails to set these; probably a bug
	el.GetObjectKind().SetGroupVersionKind(
		schema.GroupVersionKind{
			Version: "triggers.tekton.dev",
			Kind:    "EventListener",
		})

	return printer.PrintObject(w, el, f)
}

func printEventListenerDescription(s *cli.Stream, p cli.Params, elName string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	el, err := cs.Triggers.TriggersV1alpha1().EventListeners(p.Namespace()).Get(elName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get eventlistener %s: %v", elName, err)
	}

	var data = struct {
		EventListener *v1alpha1.EventListener
	}{
		EventListener: el,
	}

	funcMap := template.FuncMap{
		"decorate":             formatted.DecorateAttr,
		"getInterceptors":      getInterceptors,
		"getURL":               getURL,
		"getEventListenerName": getEventListenerName,
		"isBindingRefExist":    isBindingRefExist,
		"isBindingSpecExist":   isBindingSpecExist,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	tparsed := template.Must(template.New("Describe EventListener").Funcs(funcMap).Parse(describeTemplate))
	if err = tparsed.Execute(w, data); err != nil {
		fmt.Fprintf(s.Err, "Failed to execute template \n")
		return err
	}
	return w.Flush()
}

func getInterceptors(interceptors []*v1alpha1.EventInterceptor) string {
	resourceTemplate, err := yaml.Marshal(interceptors)
	if err != nil {
		return "yaml marshal failed with error: " + err.Error()
	}
	return string(resourceTemplate)
}

func getURL(listener v1alpha1.EventListener) string {
	if listener.Status.AddressStatus.Address == nil {
		return ""
	}
	return listener.Status.Address.URL.String()
}

func getEventListenerName(listener v1alpha1.EventListener) string {
	if listener.Status.Configuration.GeneratedResourceName == "" {
		return ""
	}
	return listener.Status.Configuration.GeneratedResourceName
}

func isBindingRefExist(bindings []*v1alpha1.EventListenerBinding) bool {
	refExist := false
	for _, j := range bindings {
		if j.Ref != "" {
			refExist = true
		}
	}
	return refExist
}

func isBindingSpecExist(bindings []*v1alpha1.EventListenerBinding) bool {
	specExist := false
	for _, j := range bindings {
		if j.Spec != nil {
			specExist = true
		}
	}
	return specExist
}
