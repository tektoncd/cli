// Copyright © 2019 The Knative Authors.
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
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
)

const (
	kubeConfig = "kubeconfig"
	namespace  = "namespace"
)

// Options is a set of configurations used to configure cobra.Command
type Options struct {
	Required bool // Required when set marks the flag as mandatory
}

// Initializer accepts a cli.Param and sets/initializes it based on flags in cobra.Command
type Initializer func(p cli.Params, cmd *cobra.Command) error

// RunE is the type for cobra.RunE
type RunE func(cmd *cobra.Command, args []string) error

// InitParams initialises cli.Params
func InitParams(p cli.Params, ilist ...Initializer) RunE {
	return func(cmd *cobra.Command, args []string) error {
		for _, fn := range ilist {
			if err := fn(p, cmd); err != nil {
				return err
			}
		}
		return nil
	}
}

// FromKubeConfig adds an optional --kubeconfig to cmd
func FromKubeConfig(cmd *cobra.Command) Initializer {
	cmd.PersistentFlags().StringP(
		kubeConfig, "k", "",
		"kubectl config file (default: $HOME/.kube/config)")
	return kcInitializer
}

func kcInitializer(p cli.Params, cmd *cobra.Command) error {
	kcPath, err := cmd.Flags().GetString(kubeConfig)
	if err != nil {
		return err
	}
	p.SetKubeConfigPath(kcPath)

	// ensure that the config is valid by creating a client
	_, err = p.Clientset()
	return err
}

// FromNamespace add a mandatory namespace flag to cmd
func FromNamespace(cmd *cobra.Command, o Options) Initializer {
	cmd.PersistentFlags().StringP(
		namespace, "n", "",
		"namespace to use (mandatory)")

	if o.Required {
		cmd.MarkPersistentFlagRequired(namespace)
	}
	return nsInitializer
}

func nsInitializer(p cli.Params, cmd *cobra.Command) error {
	ns, err := cmd.Flags().GetString(namespace)
	if err != nil {
		return err
	}
	p.SetNamespace(ns)
	return nil
}
