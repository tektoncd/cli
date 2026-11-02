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
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cli/prerun"
	"github.com/tektoncd/cli/pkg/cmd/bundle"
	"github.com/tektoncd/cli/pkg/cmd/clustertriggerbinding"
	"github.com/tektoncd/cli/pkg/cmd/completion"
	"github.com/tektoncd/cli/pkg/cmd/customrun"
	"github.com/tektoncd/cli/pkg/cmd/eventlistener"
	hubApp "github.com/tektoncd/cli/pkg/cmd/hub/app"
	hub "github.com/tektoncd/cli/pkg/cmd/hub/cmd"
	"github.com/tektoncd/cli/pkg/cmd/pipeline"
	"github.com/tektoncd/cli/pkg/cmd/pipelinerun"
	"github.com/tektoncd/cli/pkg/cmd/task"
	"github.com/tektoncd/cli/pkg/cmd/taskrun"
	"github.com/tektoncd/cli/pkg/cmd/triggerbinding"
	"github.com/tektoncd/cli/pkg/cmd/triggertemplate"
	"github.com/tektoncd/cli/pkg/cmd/version"
	"github.com/tektoncd/cli/pkg/exitcode"
	"github.com/tektoncd/cli/pkg/formatted"
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
		Use:   "tkn",
		Short: "CLI for tekton pipelines",
		Long:  ``,
		// Silence Cobra's own error and usage output so Execute is the single
		// place that renders failures.  Execute formats the error (plain text
		// or structured JSON for --output json) and derives the exit code; if
		// Cobra also printed "Error: ..." it would duplicate the message and
		// corrupt the JSON on stderr.
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
	// Classify Cobra flag-parsing failures (unknown or malformed flags) as
	// InvalidInput so they exit with the documented code.  The func is set on
	// the root command and inherited by every subcommand.
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return exitcode.New(exitcode.InvalidInput, "%s", err)
	})
	// Classify positional-argument validation failures as InvalidInput too.
	visitCommands(cmd, classifyArgErrors)
	addPluginsToHelp()
	cobra.AddTemplateFunc("isExperimental", prerun.IsExperimental)
	cobra.AddTemplateFunc("commandName", commandName)

	return cmd
}

// printError writes err to errW.  When asJSON is true the error is serialised
// as {"error":"<message>","code":<n>}; otherwise the standard
// "Error: <message>\n" format is used.
func printError(err error, errW io.Writer, asJSON bool) {
	if asJSON {
		payload := struct {
			Error string `json:"error"`
			Code  int    `json:"code"`
		}{
			Error: err.Error(),
			Code:  exitcode.CodeFrom(err),
		}
		b, _ := json.Marshal(payload)
		fmt.Fprintf(errW, "%s\n", b)
		return
	}
	fmt.Fprintf(errW, "Error: %s\n", err)
}

// Execute runs the tkn command tree with args and returns the process exit
// code.  The output format is resolved from the raw arguments up front, before
// Cobra parses flags: Cobra stops at the first invalid flag, so
// "tkn ... --limit=abc --output json" would otherwise lose the --output value
// and misclassify the error format.  In JSON error mode the executed command's
// advisory output (command and flag deprecation notices, hook warnings) is
// suppressed — Cobra prints it before SilenceErrors can cover it — so a failure
// emits only the JSON error, keeping stderr valid JSON.
func Execute(root *cobra.Command, args []string, errW io.Writer) int {
	root.SetArgs(args)

	asJSON := false
	if target, _, ferr := root.Find(args); ferr == nil && target != nil {
		if asJSON = wantsJSONError(target, args); asJSON {
			suppressWarnings(target)
		}
	}

	if _, err := root.ExecuteC(); err != nil {
		printError(err, errW, asJSON)
		return exitcode.CodeFrom(err)
	}
	return exitcode.Success
}

// suppressWarnings silences the advisory text Cobra and pflag print before a
// command runs, which in JSON error mode would precede the JSON error on stderr
// and make it unparseable.  Command and flag deprecation notices are written to
// the out writer, which is shared with real output, so they are cleared at the
// source; hook warnings use the error writer, which is redirected to
// io.Discard.  The command tree is rebuilt per process, so the mutation is safe.
func suppressWarnings(cmd *cobra.Command) {
	cmd.Deprecated = ""
	cmd.SetErr(io.Discard)
	clearDeprecated := func(f *pflag.Flag) { f.Deprecated = "" }
	cmd.InheritedFlags().VisitAll(clearDeprecated)
	cmd.Flags().VisitAll(clearDeprecated)
}

// wantsJSONError reports whether the resolved --output value selects the JSON
// error format.
func wantsJSONError(cmd *cobra.Command, args []string) bool {
	return formatted.NormalizeOutput(resolveOutputFlag(cmd, args)) == formatted.OutputJSON
}

// resolveOutputFlag returns the effective --output value.  The raw command line
// is the source of truth: Cobra stops at the first invalid flag, so a later
// --output may never be stored, and pflag's last-occurrence semantics must be
// honored even when an earlier --output was parsed.  The parsed value (which
// includes any non-empty default) is used only when no --output appears in the
// arguments.
func resolveOutputFlag(cmd *cobra.Command, args []string) string {
	outputFlag, _ := cmd.Flags().GetString("output")
	if cmd.Flags().Lookup("output") != nil {
		if raw, ok := outputFromArgs(cmd.Flags(), args); ok {
			outputFlag = raw
		}
	}
	return outputFlag
}

// outputFromArgs scans raw command-line arguments for the --output/-o flag and
// returns its value.  It is used as a fallback when flag parsing failed before
// reaching --output, so error formatting does not depend on flag order.  The
// last occurrence up to a bare "--" wins, matching pflag's semantics for
// repeated string flags; the boolean return reports whether any value was seen.
//
// The scan is flag-aware: it consults flags so that a value consumed by another
// option (for example "--component -ojson" or "-n foo") is skipped rather than
// mistaken for the output flag.
func outputFromArgs(flags *pflag.FlagSet, args []string) (string, bool) {
	value := ""
	found := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			// Everything after a bare "--" is a positional argument, so stop
			// looking for the flag.
			return value, found
		case strings.HasPrefix(a, "--"):
			name := a[len("--"):]
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				if name[:eq] == "output" {
					value, found = name[eq+1:], true
				}
				continue
			}
			if name == "output" {
				if i+1 < len(args) {
					value, found = args[i+1], true
					i++
				}
				continue
			}
			// A long flag that takes a value consumes the next argument, which
			// must not be interpreted as a flag.
			if i+1 < len(args) && flagTakesValue(flags.Lookup(name)) {
				i++
			}
		case strings.HasPrefix(a, "-") && len(a) > 1:
			// Short-flag cluster, e.g. "-o", "-oyaml", "-n foo" or "-abc".
			v, isOutput, consumedNext := outputFromShortCluster(flags, a[1:], args, i)
			if isOutput {
				value, found = v, true
			}
			if consumedNext {
				i++
			}
		}
	}
	return value, found
}

// outputFromShortCluster inspects a short-flag cluster (the characters after a
// single "-").  It returns the output value when -o is found, otherwise it
// reports whether the following argument is consumed as another flag's value.
func outputFromShortCluster(flags *pflag.FlagSet, cluster string, args []string, i int) (value string, found, consumedNext bool) {
	for j := 0; j < len(cluster); j++ {
		short := cluster[j]
		rest := cluster[j+1:]
		fl := flags.ShorthandLookup(string(short))
		// Only treat -o as the output flag when the command registers "o" as
		// the shorthand for --output.  Some commands (for example "task start")
		// define --output without a shorthand, so a stray -o is unknown to
		// Cobra and must not override a valid --output value.
		if short == 'o' && fl != nil && fl.Name == "output" {
			switch {
			case strings.HasPrefix(rest, "="):
				return rest[len("="):], true, false
			case rest != "":
				return rest, true, false
			case i+1 < len(args):
				// The value is the next argument; report it as consumed so the
				// caller does not rescan it.
				return args[i+1], true, true
			default:
				// A trailing "-o" with no following value cannot select a
				// format; report nothing found so an earlier --output value is
				// retained, matching the long "--output" handling.
				return "", false, false
			}
		}
		// An unknown shorthand makes the whole cluster invalid to pflag, which
		// aborts parsing it; mirror that so a stray cluster such as "-zoyaml"
		// cannot reach a later "o" and override an earlier --output value.
		if fl == nil {
			return "", false, false
		}
		// A short flag given an explicit value with "-x=value" ends the
		// cluster: the remainder is that flag's value (even for NoOptDefVal
		// booleans such as "-C=false"), not more flags, so stop scanning.
		if strings.HasPrefix(rest, "=") {
			return "", false, false
		}
		// A short flag that takes a value consumes the rest of the cluster or,
		// when it is the last character, the next argument.  Either way no more
		// flags follow in this cluster.
		if flagTakesValue(fl) {
			return "", false, rest == ""
		}
	}
	return "", false, false
}

// flagTakesValue reports whether fl requires an argument.  Boolean-style flags
// set NoOptDefVal and may appear without one; an unknown (nil) flag is assumed
// not to consume a value.
func flagTakesValue(fl *pflag.Flag) bool {
	return fl != nil && fl.NoOptDefVal == ""
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

// classifyArgErrors wraps a command's argument validator so that positional
// argument validation failures (for example a missing resource name) are
// reported as InvalidInput, giving them the documented exit code instead of
// the generic error code.  Validators that already return a classified error
// (for example the start commands, which flag an invalid --output value as
// InvalidInput) are left untouched, so only unclassified validation failures
// are converted.  Operational work that can fail for non-validation reasons
// (client setup, API lookups) lives in PreRunE rather than Args, so such
// failures keep their real exit code and are never seen here.
func classifyArgErrors(cmd *cobra.Command) {
	orig := cmd.Args
	if orig == nil {
		return
	}
	cmd.Args = func(c *cobra.Command, args []string) error {
		err := orig(c, args)
		if err == nil {
			return nil
		}
		if exitcode.CodeFrom(err) != exitcode.GeneralError {
			return err
		}
		return exitcode.New(exitcode.InvalidInput, "%s", err)
	}
}

func visitCommands(cmd *cobra.Command, f func(*cobra.Command)) {
	f(cmd)
	for _, child := range cmd.Commands() {
		visitCommands(child, f)
	}
}
