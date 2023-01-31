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

package flags

import (
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"golang.org/x/term"
)

const (
	kubeConfig = "kubeconfig"
	context    = "context"
	namespace  = "namespace"
	nocolour   = "nocolour"
	nocolor    = "no-color"
)

// TektonOptions all global tekton options
type TektonOptions struct {
	KubeConfig, Context, Namespace string
	Nocolour                       bool
}

// AddTektonOptions amends command to add flags required to initialise a cli.Param
func AddTektonOptions(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP(
		kubeConfig, "k", "",
		"kubectl config file (default: $HOME/.kube/config)")

	cmd.PersistentFlags().StringP(
		context, "c", "",
		"name of the kubeconfig context to use (default: kubectl config current-context)")

	cmd.PersistentFlags().StringP(
		namespace, "n", "",
		"namespace to use (default: from $KUBECONFIG)")
	_ = cmd.RegisterFlagCompletionFunc(namespace,
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return formatted.BaseCompletion("namespace", args)
		},
	)

	cmd.PersistentFlags().BoolP(
		nocolour, "", false,
		"disable colouring (default: false)")

	// Since nocolour was the old name for the --no-color flag, mark
	// the flag as hidden to make the option backwards compatible while
	// only showing --no-color option in help output.
	_ = cmd.PersistentFlags().MarkHidden("nocolour")

	cmd.PersistentFlags().BoolP(
		"no-color", "C", false,
		"disable coloring (default: false)")
}

// GetTektonOptions get the global tekton Options that are not passed to a subcommands
func GetTektonOptions(cmd *cobra.Command) TektonOptions {
	kcPath, _ := cmd.Flags().GetString(kubeConfig)
	kubeContext, _ := cmd.Flags().GetString(context)
	ns, _ := cmd.Flags().GetString(namespace)
	nocolourFlag, _ := cmd.Flags().GetBool(nocolor)
	return TektonOptions{
		KubeConfig: kcPath,
		Context:    kubeContext,
		Namespace:  ns,
		Nocolour:   nocolourFlag,
	}
}

// InitParams initialises cli.Params based on flags defined in command
func InitParams(p cli.Params, cmd *cobra.Command) error {
	// NOTE: breaks symmetry with AddTektonOptions as this uses Flags instead of
	// PersistentFlags as it could be the sub command that is trying to access
	// the flags defined by the parent and hence need to use `Flag` instead
	// e.g. `list` accessing kubeconfig defined by `pipeline`
	kcPath, err := cmd.Flags().GetString(kubeConfig)
	if err != nil {
		return err
	}
	p.SetKubeConfigPath(kcPath)

	kubeContext, err := cmd.Flags().GetString(context)
	if err != nil {
		return err
	}
	p.SetKubeContext(kubeContext)

	// ensure that the config is valid by creating a client but skip for bundle cmd
	// as bundle cmd does not need k8s client and config
	// if this annotation is available on cmd and value is false then client
	// will not be initialized
	if cmd.Annotations["kubernetes"] != "false" {
		if _, err := p.Clients(); err != nil {
			return err
		}
	}

	ns, err := cmd.Flags().GetString(namespace)
	if err != nil {
		return err
	}
	if ns != "" {
		p.SetNamespace(ns)
	}

	nocolourFlag, err := cmd.Flags().GetBool(nocolor)
	if err != nil {
		return err
	}
	if !nocolourFlag {
		// Check to see if --nocolour option was passed instead of --no-color or -C
		nocolourFlag, err = cmd.Flags().GetBool(nocolour)
		if err != nil {
			return err
		}
	}
	p.SetNoColour(nocolourFlag)

	// Make sure we set as Nocolour if we don't have a terminal (ie redirection)
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		p.SetNoColour(true)
	}

	if runtime.GOOS == "windows" {
		p.SetNoColour(true)
	}

	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		p.SetNoColour(true)
	}

	return nil
}
