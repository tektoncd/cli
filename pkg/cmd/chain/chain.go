// Copyright © 2022 The Tekton Authors.
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

package chain

import (
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cli/prerun"
	"github.com/tektoncd/cli/pkg/flags"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	x509Keypair string = "k8s://%s/signing-secrets"
)

var taskrunGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}

func Command(p cli.Params) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "chain",
		Aliases: []string{"chains"},
		Short:   "Manage Chains",
		Annotations: map[string]string{
			"commandType": "main",
		},
		PersistentPreRunE: prerun.PersistentPreRunE(p),
	}

	flags.AddTektonOptions(cmd)
	cmd.PersistentFlags().String(
		"chains-namespace", "tekton-chains",
		"namespace in which chains is installed")
	cmd.AddCommand(
		payloadCommand(p),
		signatureCommand(p),
	)
	return cmd
}
