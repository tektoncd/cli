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
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/eventlistener"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/yaml"
)

const describeTemplate = `{{decorate "bold" "Name"}}:	{{ .EventListener.Name }}
{{decorate "bold" "Namespace"}}:	{{ .EventListener.Namespace }}

{{- if ne .EventListener.Spec.ServiceAccountName "" }}
{{decorate "bold" "Service Account"}}:	{{ .EventListener.Spec.ServiceAccountName }}
{{- end }}

{{- if not .EventListener.Spec.Resources.KubernetesResource }}
{{- else }}
{{- if ne .EventListener.Spec.Resources.KubernetesResource.ServiceType "" }}
{{decorate "bold" "Service Type"}}:	{{ .EventListener.Spec.Resources.KubernetesResource.ServiceType }}
{{- end }}
{{- end }}

{{- $value := getURL .EventListener }}
{{-  if ne $value "" }}
{{decorate "bold" "URL"}}:	{{ $value }}
{{- end }}

{{- $value := getEventListenerName .EventListener }}
{{-  if ne $value "" }}
{{decorate "bold" "EventListnerServiceName"}}:	{{ $value }}
{{- end }}

{{- if ne (len .EventListener.Spec.Triggers) 0 }}
{{ "" }}
{{- range $v := .EventListener.Spec.Triggers }}
{{decorate "bold" "EventListenerTriggers\n"}}

{{- if eq $v.Name "" }}
{{- else }}
 NAME
 {{decorate "bullet" $v.Name }}
{{ "" }}
{{- end }}

{{- if not $v.Bindings }}
{{- else }}
{{- if ne (len $v.Bindings) 0 }}
 {{decorate "" "BINDINGS\n"}}
{{- if isBindingRefExist $v.Bindings }}
  REF	KIND	APIVERSION
{{- range $b := $v.Bindings }}
{{- if ne $b.Ref "" }}
  {{ decorate "bullet" $b.Ref }}	{{ $b.Kind }}	{{ $b.APIVersion }}
{{- end }}
{{- end }}
{{ "" }}
{{- end }}
{{- if isBindingNameExist $v.Bindings }}
  NAME	VALUE
{{- range $b := $v.Bindings }}
{{- if ne $b.Name "" }}
  {{ decorate "bullet" $b.Name }}	{{ $b.Value }}
{{- end }}
{{- end }}
{{ "" }}
{{- end }}

{{- end }}
{{- end }}
{{- if not $v.Template }}
{{- else }}
{{- if isTemplateRefExist $v.Template }}
 TEMPLATE REF	APIVERSION
 {{ decorate "bullet" $v.Template.Ref }}	{{ $v.Template.APIVersion }}
 {{ "" }}
{{- end }}
{{- end }}
{{- if eq $v.TriggerRef "" }}
{{- else }}
 TRIGGER REF
 {{ decorate "bullet" $v.TriggerRef }}
 {{ "" }}
{{- end }}
{{- if eq $v.ServiceAccountName "" }}
{{- else }}
 SERVICE ACCOUNT NAME
 {{ decorate "bullet" $v.ServiceAccountName }}
 {{ "" }}
{{- end }}
{{- if not $v.Interceptors }}
{{- else }}
{{- if ne (len $v.Interceptors) 0 }}
 INTERCEPTORS
{{- $b := getInterceptors $v.Interceptors }}
{{ $b }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
`

func describeCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("describe")
	opts := &options.DescribeOptions{Params: p}
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
				fmt.Fprint(os.Stderr, "Error: output option not set properly \n")
				return err
			}

			if len(args) == 0 {
				eventListenerNames, err := eventlistener.GetAllEventListenerNames(cs, p.Namespace())
				if err != nil {
					return err
				}
				if len(eventListenerNames) == 1 {
					opts.EventListenerName = eventListenerNames[0]
				} else {
					err = askEventListenerName(opts, eventListenerNames)
					if err != nil {
						return err
					}
				}
			} else {
				opts.EventListenerName = args[0]
			}

			if output != "" {
				if strings.ToLower(output) == "url" {
					return describeEventListenerOutputURL(cmd.OutOrStdout(), p, args[0])
				}
				printer, err := f.ToPrinter()
				if err != nil {
					return err
				}
				return actions.PrintObject(eventlistenerGroupResource, opts.EventListenerName, cmd.OutOrStdout(), cs.Dynamic, cs.Triggers.Discovery(), printer, p.Namespace())
			}
			return printEventListenerDescription(s, p, opts.EventListenerName)
		},
	}

	f.AddFlags(c)
	return c
}

func describeEventListenerOutputURL(w io.Writer, p cli.Params, name string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	el, err := eventlistener.Get(cs, name, metav1.GetOptions{}, p.Namespace())
	if err != nil {
		return err
	}

	if getURL(*el) == "" {
		return fmt.Errorf("url of EventListener %s not available yet", el.Name)
	}
	fmt.Fprintf(w, "%s\n", getURL(*el))
	return nil
}

func printEventListenerDescription(s *cli.Stream, p cli.Params, elName string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	el, err := eventlistener.Get(cs, elName, metav1.GetOptions{}, p.Namespace())
	if err != nil {
		return fmt.Errorf("failed to get EventListener %s: %v", elName, err)
	}

	var data = struct {
		EventListener *v1beta1.EventListener
	}{
		EventListener: el,
	}

	funcMap := template.FuncMap{
		"decorate":             formatted.DecorateAttr,
		"getInterceptors":      getInterceptors,
		"getURL":               getURL,
		"getEventListenerName": getEventListenerName,
		"isBindingRefExist":    isBindingRefExist,
		"isBindingNameExist":   isBindingNameExist,
		"isTemplateRefExist":   isTemplateRefExist,
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	tparsed := template.Must(template.New("Describe EventListener").Funcs(funcMap).Parse(describeTemplate))
	if err = tparsed.Execute(w, data); err != nil {
		fmt.Fprintf(s.Err, "Failed to execute template \n")
		return err
	}
	return w.Flush()
}

func getInterceptors(interceptors []*v1beta1.EventInterceptor) string {
	resourceTemplate, err := yaml.Marshal(interceptors)
	if err != nil {
		return "yaml marshal failed with error: " + err.Error()
	}
	return string(resourceTemplate)
}

func getURL(listener v1beta1.EventListener) string {
	if listener.Status.AddressStatus.Address == nil {
		return ""
	}
	return listener.Status.Address.URL.String()
}

func getEventListenerName(listener v1beta1.EventListener) string {
	if listener.Status.Configuration.GeneratedResourceName == "" {
		return ""
	}
	return listener.Status.Configuration.GeneratedResourceName
}

func isBindingRefExist(bindings []*v1beta1.EventListenerBinding) bool {
	refExist := false
	for _, j := range bindings {
		if j.Ref != "" {
			refExist = true
		}
	}
	return refExist
}

func isBindingNameExist(bindings []*v1beta1.EventListenerBinding) bool {
	nameExist := false
	for _, j := range bindings {
		if j.Name != "" {
			return true
		}
	}
	return nameExist
}

func isTemplateRefExist(templates *v1beta1.EventListenerTemplate) bool {
	return templates.Ref != nil
}

func askEventListenerName(opts *options.DescribeOptions, eventListenerNames []string) error {
	if len(eventListenerNames) == 0 {
		return fmt.Errorf("no EventListeners found")
	}
	err := opts.Ask(options.ResourceNameEventListener, eventListenerNames)
	if err != nil {
		return err
	}

	return nil
}
