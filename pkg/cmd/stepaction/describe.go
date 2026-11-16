// Copyright © 2026 The Tekton Authors.
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

package stepaction

import (
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	describeTemplate = `{{decorate "bold" "Name"}}:	{{ .StepAction.Name }}
{{decorate "bold" "Namespace"}}:	{{ .StepAction.Namespace }}
{{- if ne .StepAction.Spec.Description "" }}
{{decorate "bold" "Description"}}:	{{ .StepAction.Spec.Description }}
{{- end }}
{{- if ne .StepAction.Spec.Image "" }}
{{decorate "bold" "Image"}}:	{{ .StepAction.Spec.Image }}
{{- end }}
{{- if ne (len .StepAction.Spec.Command) 0 }}
{{decorate "bold" "Command"}}:	{{ .StepAction.Spec.Command }}
{{- end }}
{{- if ne (len .StepAction.Spec.Args) 0 }}
{{decorate "bold" "Args"}}:	{{ .StepAction.Spec.Args }}
{{- end }}
{{- if ne .StepAction.Spec.WorkingDir "" }}
{{decorate "bold" "Working Dir"}}:	{{ .StepAction.Spec.WorkingDir }}
{{- end }}
{{- if ne .StepAction.Spec.Script "" }}
{{decorate "bold" "Script"}}:	{{ "set" }}
{{- end }}
{{- $l := len .StepAction.Labels }}{{ if ne $l 0 }}
{{decorate "bold" "Labels"}}:
{{- range $k, $v := .StepAction.Labels }}
 {{ $k }}={{ $v }}
{{- end }}
{{- end }}
{{- $annotations := removeLastAppliedConfig .StepAction.Annotations -}}
{{- if $annotations }}
{{decorate "bold" "Annotations"}}:
{{- range $k, $v := $annotations }}
 {{ $k }}={{ $v }}
{{- end }}
{{- end }}

{{- if ne (len .StepAction.Spec.Params) 0 }}

{{decorate "params" ""}}{{decorate "underline bold" "Params\n"}}
 NAME	TYPE	DESCRIPTION	DEFAULT VALUE
{{- range $p := .StepAction.Spec.Params }}
{{- if not $p.Default }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ formatDesc $p.Description }}	{{ "---" }}
{{- else }}
{{- if eq $p.Type "string" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ formatDesc $p.Description }}	{{ $p.Default.StringVal }}
{{- else if eq $p.Type "array" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ formatDesc $p.Description }}	{{ $p.Default.ArrayVal }}
{{- else }}
 {{decorate "bullet" $p.Name }}	{{ $p.Type }}	{{ formatDesc $p.Description }}	{{ $p.Default.ObjectVal }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{- if ne (len .StepAction.Spec.Results) 0 }}

{{decorate "results" ""}}{{decorate "underline bold" "Results\n"}}
 NAME	TYPE	DESCRIPTION
{{- range $result := .StepAction.Spec.Results }}
 {{ decorate "bullet" $result.Name }}	{{ resultType $result.Type }}	{{ formatDesc $result.Description }}
{{- end }}
{{- end }}
`
	defaultStepActionDescribeLimit = 5
)

func describeCommand(p cli.Params) *cobra.Command {
	opts := &options.DescribeOptions{Params: p}
	f := cliopts.NewPrintFlags("describe")
	eg := `Describe a StepAction of name 'foo' in namespace 'bar':

    tkn stepaction describe foo -n bar

or

    tkn stepaction desc foo -n bar

Describe a StepAction as JSON:

    tkn stepaction describe foo -n bar -o json

Describe a StepAction as YAML:

    tkn stepaction describe foo -n bar -o yaml
`

	c := &cobra.Command{
		Use:          "describe",
		Aliases:      []string{"desc"},
		Short:        "Describe a StepAction in a namespace",
		Example:      eg,
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		ValidArgsFunction: formatted.ParentCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}

			if !opts.Fzf {
				if _, ok := os.LookupEnv("TKN_USE_FZF"); ok {
					opts.Fzf = true
				}
			}

			cs, err := p.Clients()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				names, err := getAllStepActionNames(cs, p.Namespace(), opts.Limit)
				if err != nil {
					return err
				}
				if len(names) == 1 {
					opts.StepActionName = names[0]
				} else if err := askStepActionName(opts, names); err != nil {
					return err
				}
			} else {
				opts.StepActionName = args[0]
			}

			if output != "" {
				printer, err := f.ToPrinter()
				if err != nil {
					return err
				}
				return actions.PrintObjectV1(stepactionGroupResource, opts.StepActionName, cmd.OutOrStdout(), cs, printer, p.Namespace())
			}

			return printStepActionDescription(cmd.OutOrStdout(), cs, p.Namespace(), opts.StepActionName)
		},
	}

	c.Flags().IntVarP(&opts.Limit, "limit", "", defaultStepActionDescribeLimit, "lists number of StepActions when selecting a StepAction to describe")
	c.Flags().BoolVarP(&opts.Fzf, "fzf", "F", false, "use fzf to select a StepAction to describe")

	f.AddFlags(c)

	return c
}

func askStepActionName(opts *options.DescribeOptions, names []string) error {
	if err := opts.ValidateOpts(); err != nil {
		return err
	}
	if len(names) == 0 {
		return fmt.Errorf("no StepActions found")
	}
	if opts.Fzf {
		return opts.FuzzyAsk(options.ResourceNameStepAction, names)
	}
	return opts.Ask(options.ResourceNameStepAction, names)
}

// getAllStepActionNames returns the StepAction names in a namespace in sorted
// order, keeping at most limit entries when limit is positive
func getAllStepActionNames(c *cli.Clients, ns string, limit int) ([]string, error) {
	var sas *v1beta1.StepActionList
	if err := actions.ListV1(stepactionGroupResource, c, metav1.ListOptions{}, ns, &sas); err != nil {
		return nil, fmt.Errorf("failed to list StepActions from namespace %s: %w", ns, err)
	}

	names := make([]string, 0, len(sas.Items))
	for _, sa := range sas.Items {
		names = append(names, sa.Name)
	}
	sort.Strings(names)
	if limit > 0 && len(names) > limit {
		names = names[:limit]
	}
	return names, nil
}

func getStepAction(c *cli.Clients, name, ns string) (*v1beta1.StepAction, error) {
	var sa v1beta1.StepAction
	if err := actions.GetV1(stepactionGroupResource, c, name, ns, metav1.GetOptions{}, &sa); err != nil {
		return nil, err
	}
	return &sa, nil
}

func printStepActionDescription(out io.Writer, c *cli.Clients, ns, name string) error {
	sa, err := getStepAction(c, name, ns)
	if err != nil {
		return fmt.Errorf("failed to get StepAction %s: %w", name, err)
	}

	data := struct {
		StepAction *v1beta1.StepAction
	}{
		StepAction: sa,
	}

	funcMap := template.FuncMap{
		"decorate":                formatted.DecorateAttr,
		"formatDesc":              formatted.FormatDesc,
		"removeLastAppliedConfig": formatted.RemoveLastAppliedConfig,
		"resultType":              resultType,
	}

	w := tabwriter.NewWriter(out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Describe StepAction").Funcs(funcMap).Parse(describeTemplate))
	if err := t.Execute(w, data); err != nil {
		return err
	}
	return w.Flush()
}

// resultType shows the result type, which defaults to string when not set
func resultType(t interface{}) string {
	s := fmt.Sprint(t)
	if s == "" {
		return "string"
	}
	return s
}
