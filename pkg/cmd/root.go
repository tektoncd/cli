// Copyright © 2019 The Tekton Authors.
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

package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cli/prerun"
	"github.com/tektoncd/cli/pkg/cmd/bundle"
	"github.com/tektoncd/cli/pkg/cmd/clustertriggerbinding"
	"github.com/tektoncd/cli/pkg/cmd/completion"
	"github.com/tektoncd/cli/pkg/cmd/customrun"
	"github.com/tektoncd/cli/pkg/cmd/eventlistener"
	"github.com/tektoncd/cli/pkg/exitcode"
	hubApp "github.com/tektoncd/cli/pkg/cmd/hub/app"
	hub "github.com/tektoncd/cli/pkg/cmd/hub/cmd"
	"github.com/tektoncd/cli/pkg/cmd/pipeline"
	"github.com/tektoncd/cli/pkg/cmd/pipelinerun"
	"github.com/tektoncd/cli/pkg/cmd/task"
	"github.com/tektoncd/cli/pkg/cmd/taskrun"
	"github.com/tektoncd/cli/pkg/cmd/triggerbinding"
	"github.com/tektoncd/cli/pkg/cmd/triggertemplate"
	"github.com/tektoncd/cli/pkg/cmd/version"
	"github.com/tektoncd/cli/pkg/plugins"
	"github.com/tektoncd/cli/pkg/suggestion"
)

const usageTemplate = `Usage:{{if .Runnable}}
{{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
{{.CommandPath}} [command]{{end}}
{{- if isExperimental .}}

EXPERIMENTAL:
  {{.CommandPath}} is an experimental feature.
  Experimental features provide early access to the project functionality. These
  features may change between releases without warning, or can be removed from a
  future release.{{- end}}
{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{if HasMainSubCommands .}}

Available Commands:{{range .Commands}}{{if (eq .Annotations.commandType "main")}}
  {{rpad (commandName .) .NamePadding }} {{.Short}}{{if isExperimental .}} (experimental){{end}}{{end}}{{end}}{{end}}{{if HasUtilitySubCommands .}}

Other Commands:{{range .Commands}}{{if (eq .Annotations.commandType "utility")}}
  {{rpad (commandName .) .NamePadding }} {{.Short}}{{if isExperimental .}} (experimental){{end}}{{end}}{{end}}{{end}}{{end}}{{if gt (len pluginList) 0}}
{{- if not .HasParent}}

Available Plugins:

{{- range pluginList}}
  {{.}}
{{- end}}
{{- end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
{{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

func Root(p cli.Params) *cobra.Command {
	// Reset CommandLine so we don't get the flags from the libraries, i.e:
	// azure library adding --azure-container-registry-config
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	cmd := &cobra.Command{
		Use:           "tkn",
		Short:         "CLI for tekton pipelines",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cobra.AddTemplateFunc("HasMainSubCommands", hasMainSubCommands)
	cobra.AddTemplateFunc("HasUtilitySubCommands", hasUtilitySubCommands)
	cmd.SetUsageTemplate(usageTemplate)

	cmd.AddCommand(
		bundle.Command(p),
		clustertriggerbinding.Command(p),
		completion.Command(),
		eventlistener.Command(p),
		pipeline.Command(p),
		pipelinerun.Command(p),
		task.Command(p),
		taskrun.Command(p),
		customrun.Command(p),
		triggerbinding.Command(p),
		triggertemplate.Command(p),
		version.Command(p),
		hub.Root(hubApp.New()),
	)
	visitCommands(cmd, reconfigureCmdWithSubcmd)
	addPluginsToHelp()
	cobra.AddTemplateFunc("isExperimental", prerun.IsExperimental)
	cobra.AddTemplateFunc("commandName", commandName)

	return cmd
}

// PrintError writes err to errW.  When the resolved --output flag on cmd is
// "json", the error is serialised as {"error":"<message>","code":<n>}.
// Otherwise the standard "Error: <message>\n" format is used.
func PrintError(cmd *cobra.Command, err error, errW *os.File) {
	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		payload := struct {
			Error string `json:"error"`
			Code  int    `json:"code"`
		}{
			Error: err.Error(),
			Code:  exitcode.CodeFrom(err),
		}
		b, _ := json.Marshal(payload)
		fmt.Fprintf(errW, "%s\n", b)
	} else {
		fmt.Fprintf(errW, "Error: %s\n", err)
	}
}

func commandName(cmd *cobra.Command) string {
	if prerun.IsExperimental(cmd) {
		return fmt.Sprintf("%s*", cmd.Name())
	}
	return cmd.Name()
}

func addPluginsToHelp() {
	pluginList := plugins.GetAllTknPluginFromPaths()
	cobra.AddTemplateFunc("pluginList", func() []string { return pluginList })
}

func hasMainSubCommands(cmd *cobra.Command) bool {
	return len(subCommands(cmd, "main")) > 0
}

func hasUtilitySubCommands(cmd *cobra.Command) bool {
	return len(subCommands(cmd, "utility")) > 0
}

func subCommands(cmd *cobra.Command, annotation string) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if sub.IsAvailableCommand() && sub.Annotations["commandType"] == annotation {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func reconfigureCmdWithSubcmd(cmd *cobra.Command) {
	if len(cmd.Commands()) == 0 {
		return
	}

	if cmd.Args == nil {
		cmd.Args = cobra.ArbitraryArgs
	}

	if cmd.RunE == nil {
		cmd.RunE = suggestion.SubcommandsRequiredWithSuggestions
	}
}

func visitCommands(cmd *cobra.Command, f func(*cobra.Command)) {
	f(cmd)
	for _, child := range cmd.Commands() {
		visitCommands(child, f)
	}
}
