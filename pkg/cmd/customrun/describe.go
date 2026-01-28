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

package customrun

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	describeTemplate = `{{decorate "bold" "Name"}}:	{{ .CustomRun.Name }}
{{decorate "bold" "Namespace"}}:	{{ .CustomRun.Namespace }}
{{- $customRef := customRefExists .CustomRun.Spec }}{{- if ne $customRef "" }}
{{decorate "bold" "Custom Ref"}}:	{{ $customRef }}
{{- end }}
{{- if ne .CustomRun.Spec.ServiceAccountName "" }}
{{decorate "bold" "Service Account"}}:	{{ .CustomRun.Spec.ServiceAccountName }}
{{- end }}

{{- $timeout := getTimeout .CustomRun -}}
{{- if and (ne $timeout "") (ne $timeout "0s") }}
{{decorate "bold" "Timeout"}}:	{{ .CustomRun.Spec.Timeout.Duration.String }}
{{- end }}
{{- $l := len .CustomRun.Labels }}{{ if eq $l 0 }}
{{- else }}
{{decorate "bold" "Labels"}}:
{{- range $k, $v := .CustomRun.Labels }}
 {{ $k }}={{ $v }}
{{- end }}
{{- end }}
{{- $annotations := removeLastAppliedConfig .CustomRun.Annotations -}}
{{- if $annotations }}
{{decorate "bold" "Annotations"}}:
{{- range $k, $v := $annotations }}
 {{ $k }}={{ $v }}
{{- end }}
{{- end }}

{{decorate "status" ""}}{{decorate "underline bold" "Status"}}

STARTED	DURATION	STATUS
{{ formatAge .CustomRun.Status.StartTime  .Time }}	{{ formatDuration .CustomRun.Status.StartTime .CustomRun.Status.CompletionTime }}	{{ formatCondition .CustomRun.Status.Conditions }}
{{- $msg := hasFailed .CustomRun -}}
{{-  if ne $msg "" }}

{{decorate "underline bold" "Message"}}

{{ $msg }}
{{- end }}

{{- if ne (len .CustomRun.Spec.Params) 0 }}

{{decorate "params" ""}}{{decorate "underline bold" "Params"}}

 NAME	VALUE
{{- range $i, $p := .CustomRun.Spec.Params }}
{{- if eq $p.Value.Type "string" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value.StringVal }}
{{- else if eq $p.Value.Type "array" }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value.ArrayVal }}
{{- else }}
 {{decorate "bullet" $p.Name }}	{{ $p.Value.ObjectVal }}
{{- end }}
{{- end }}
{{- end }}

{{- if ne (len .CustomRun.Status.Results) 0 }}

{{decorate "results" ""}}{{decorate "underline bold" "Results"}}

 NAME	VALUE
{{- range $result := .CustomRun.Status.Results }}
 {{decorate "bullet" $result.Name }}	{{ $result.Value }}
{{- end }}
{{- end }}

{{- if ne (len .CustomRun.Spec.Workspaces) 0 }}

{{decorate "workspaces" ""}}{{decorate "underline bold" "Workspaces"}}

 NAME	SUB PATH	WORKSPACE BINDING
{{- range $workspace := .CustomRun.Spec.Workspaces }}
{{- if not $workspace.SubPath }}
 {{ decorate "bullet" $workspace.Name }}	{{ "---" }}	{{ formatWorkspace $workspace }}
{{- else }}
 {{ decorate "bullet" $workspace.Name }}	{{ $workspace.SubPath }}	{{ formatWorkspace $workspace }}
{{- end }}
{{- end }}
{{- end }}
`
	defaultCustomRunDescribeLimit = 5
)

func describeCommand(p cli.Params) *cobra.Command {
	opts := &options.DescribeOptions{Params: p}
	f := cliopts.NewPrintFlags("describe")
	eg := `Describe a CustomRun of name 'foo' in namespace 'bar':

    tkn customrun describe foo -n bar

or

    tkn cr desc foo -n bar
`

	c := &cobra.Command{
		Use:          "describe",
		Aliases:      []string{"desc"},
		Short:        "Describe a CustomRun in a namespace",
		Example:      eg,
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
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
				lOpts := metav1.ListOptions{}
				if !opts.Last {
					crs, err := GetAllCustomRuns(customrunGroupResource, lOpts, cs, p.Namespace(), opts.Limit, p.Time())
					if err != nil {
						return err
					}
					if len(crs) == 1 {
						opts.CustomRunName = strings.Fields(crs[0])[0]
					} else {
						err = askCustomRunName(opts, crs)
						if err != nil {
							return err
						}
					}
				} else {
					crs, err := GetAllCustomRuns(customrunGroupResource, lOpts, cs, p.Namespace(), 1, p.Time())
					if err != nil {
						return err
					}
					if len(crs) == 0 {
						fmt.Fprintf(s.Out, "No CustomRuns present in namespace %s\n", opts.Params.Namespace())
						return nil
					}
					opts.CustomRunName = strings.Fields(crs[0])[0]
				}
			} else {
				opts.CustomRunName = args[0]
			}

			if output != "" {
				printer, err := f.ToPrinter()
				if err != nil {
					return err
				}
				return actions.PrintObjectV1(customrunGroupResource, opts.CustomRunName, cmd.OutOrStdout(), cs, printer, p.Namespace())
			}

			return PrintCustomRunDescription(s.Out, cs, opts.Params.Namespace(), opts.CustomRunName, opts.Params.Time())
		},
	}

	c.Flags().BoolVarP(&opts.Last, "last", "L", false, "show description for last CustomRun")
	c.Flags().IntVarP(&opts.Limit, "limit", "", defaultCustomRunDescribeLimit, "lists number of CustomRuns when selecting a CustomRun to describe")
	c.Flags().BoolVarP(&opts.Fzf, "fzf", "F", false, "use fzf to select a CustomRun to describe")

	f.AddFlags(c)

	return c
}

func askCustomRunName(opts *options.DescribeOptions, crs []string) error {
	err := opts.ValidateOpts()
	if err != nil {
		return err
	}

	if len(crs) == 0 {
		return fmt.Errorf("no CustomRuns found")
	}

	if opts.Fzf {
		err = opts.FuzzyAsk(options.ResourceNameCustomRun, crs)
	} else {
		err = opts.Ask(options.ResourceNameCustomRun, crs)
	}
	if err != nil {
		return err
	}

	return nil
}

// GetAllCustomRuns returns all CustomRuns as a formatted string slice
func GetAllCustomRuns(gr schema.GroupVersionResource, opts metav1.ListOptions, c *cli.Clients, ns string, limit int, time clockwork.Clock) ([]string, error) {
	var customruns *v1beta1.CustomRunList
	if err := actions.ListV1(gr, c, opts, ns, &customruns); err != nil {
		return nil, fmt.Errorf("failed to list CustomRuns from namespace %s: %v", ns, err)
	}

	var ret []string
	for i, cr := range customruns.Items {
		if limit > 0 && i >= limit {
			break
		}
		ret = append(ret, fmt.Sprintf("%s\tStarted: %s\tDuration: %s\tStatus: %s",
			cr.Name,
			formatted.Age(cr.Status.StartTime, time),
			formatted.Duration(cr.Status.StartTime, cr.Status.CompletionTime),
			formatted.Condition(cr.Status.Conditions)))
	}
	return ret, nil
}

// GetCustomRun retrieves a CustomRun by name
func GetCustomRun(gr schema.GroupVersionResource, c *cli.Clients, crName, ns string) (*v1beta1.CustomRun, error) {
	var customrun v1beta1.CustomRun
	err := actions.GetV1(gr, c, crName, ns, metav1.GetOptions{}, &customrun)
	if err != nil {
		return nil, err
	}
	return &customrun, nil
}

// PrintCustomRunDescription prints the description of a CustomRun
func PrintCustomRunDescription(out io.Writer, c *cli.Clients, ns string, crName string, time clockwork.Clock) error {
	cr, err := GetCustomRun(customrunGroupResource, c, crName, ns)
	if err != nil {
		return fmt.Errorf("failed to get CustomRun %s: %v", crName, err)
	}

	var data = struct {
		CustomRun *v1beta1.CustomRun
		Time      clockwork.Clock
	}{
		CustomRun: cr,
		Time:      time,
	}

	funcMap := template.FuncMap{
		"formatAge":               formatted.Age,
		"formatDuration":          formatted.Duration,
		"formatCondition":         formatted.Condition,
		"formatWorkspace":         formatted.Workspace,
		"hasFailed":               hasFailed,
		"customRefExists":         customRefExists,
		"decorate":                formatted.DecorateAttr,
		"getTimeout":              getTimeoutValue,
		"removeLastAppliedConfig": formatted.RemoveLastAppliedConfig,
	}

	w := tabwriter.NewWriter(out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Describe CustomRun").Funcs(funcMap).Parse(describeTemplate))

	err = t.Execute(w, data)
	if err != nil {
		return err
	}
	return w.Flush()
}

func hasFailed(cr *v1beta1.CustomRun) string {
	if len(cr.Status.Conditions) == 0 {
		return ""
	}

	if cr.Status.Conditions[0].Status == corev1.ConditionFalse {
		return cr.Status.Conditions[0].Message
	}

	return ""
}

func getTimeoutValue(cr *v1beta1.CustomRun) string {
	if cr.Spec.Timeout != nil {
		return cr.Spec.Timeout.Duration.String()
	}
	return ""
}

func customRefExists(spec v1beta1.CustomRunSpec) string {
	if spec.CustomRef != nil && spec.CustomRef.Name != "" {
		return spec.CustomRef.Name
	}
	if spec.CustomSpec != nil {
		return fmt.Sprintf("%s/%s", spec.CustomSpec.APIVersion, spec.CustomSpec.Kind)
	}
	return ""
}
