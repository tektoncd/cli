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
	"github.com/spf13/pflag"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cmd/completion"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	kubeConfig = "kubeconfig"
	context    = "context"
	namespace  = "namespace"
	nocolour   = "nocolour"
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

	cmd.PersistentFlags().BoolP(
		nocolour, "C", false,
		"disable colouring (default: false)")

	// Add custom completion for that command as specified in
	// bashCompletionFlags map
	for name, completion := range completion.ShellCompletionMap {
		pflag := cmd.PersistentFlags().Lookup(name)
		AddShellCompletion(pflag, completion)
	}
}

// GetTektonOptions get the global tekton Options that are not passed to a subcommands
func GetTektonOptions(cmd *cobra.Command) TektonOptions {
	kcPath, _ := cmd.Flags().GetString(kubeConfig)
	kContext, _ := cmd.Flags().GetString(context)
	ns, _ := cmd.Flags().GetString(namespace)
	nocolourFlag, _ := cmd.Flags().GetBool(nocolour)
	return TektonOptions{
		KubeConfig: kcPath,
		Context:    kContext,
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

	kContext, err := cmd.Flags().GetString(context)
	if err != nil {
		return err
	}
	p.SetKubeContext(kContext)

	// ensure that the config is valid by creating a client
	if _, err := p.Clients(); err != nil {
		return err
	}

	ns, err := cmd.Flags().GetString(namespace)
	if err != nil {
		return err
	}
	if ns != "" {
		p.SetNamespace(ns)
	}

	nocolourFlag, err := cmd.Flags().GetBool(nocolour)
	if err != nil {
		return err
	}
	p.SetNoColour(nocolourFlag)

	// Make sure we set as Nocolour if we don't have a terminal (ie redirection)
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		p.SetNoColour(true)
	}

	if runtime.GOOS == "windows" {
		p.SetNoColour(true)
	}

	return nil
}

// AddShellCompletion add a hint to the cobra flag annotation for how to do a completion
func AddShellCompletion(pflag *pflag.Flag, shellfunction string) {
	if pflag.Annotations == nil {
		pflag.Annotations = map[string][]string{}
	}
	pflag.Annotations[cobra.BashCompCustom] = append(pflag.Annotations[cobra.BashCompCustom], shellfunction)
}
