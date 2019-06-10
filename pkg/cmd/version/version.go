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

package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NOTE: use go build -ldflags "-X github.com/tektoncd/cli/pkg/cmd/version.clientVersion=$(git describe)"
var clientVersion = "dev"

// Command returns version command
func Command() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "version",
		Short: "Prints version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "Client version: %s\n", clientVersion)
		},
	}
	return cmd
}
