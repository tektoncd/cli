// Copyright © 2021 The Tekton Authors.
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

package bundle

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cli/prerun"
	"github.com/tektoncd/cli/pkg/flags"
)

var (
	errInvalidRef = errors.New("first argument must be a valid docker-like Tekton bundle reference")
)

func Command(p cli.Params) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bundle",
		Aliases: []string{"tkb", "bundles"},
		Short:   "Manage Tekton Bundles",
		Annotations: map[string]string{
			"commandType":  "main",
			"experimental": "",
			"kubernetes":   "false",
		},
		PersistentPreRunE: prerun.PersistentPreRunE(p),
	}

	flags.AddTektonOptions(cmd)
	_ = cmd.PersistentFlags().MarkHidden("context")
	_ = cmd.PersistentFlags().MarkHidden("kubeconfig")
	_ = cmd.PersistentFlags().MarkHidden("namespace")
	cmd.AddCommand(
		listCommand(p),
		pushCommand(p),
	)
	return cmd
}
