// Copyright © 2020 The Tekton Authors.
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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/hub/api/pkg/cli/app"
	"github.com/tektoncd/hub/api/pkg/cli/cmd/check_upgrade"
	"github.com/tektoncd/hub/api/pkg/cli/cmd/downgrade"
	"github.com/tektoncd/hub/api/pkg/cli/cmd/get"
	"github.com/tektoncd/hub/api/pkg/cli/cmd/info"
	"github.com/tektoncd/hub/api/pkg/cli/cmd/install"
	"github.com/tektoncd/hub/api/pkg/cli/cmd/reinstall"
	"github.com/tektoncd/hub/api/pkg/cli/cmd/search"
	"github.com/tektoncd/hub/api/pkg/cli/cmd/upgrade"
	"github.com/tektoncd/hub/api/pkg/cli/hub"
)

// Root represents the base command when called without any subcommands
func Root(cli app.CLI) *cobra.Command {
	apiURL := ""
	hubType := ""

	cmd := &cobra.Command{
		Use: "hub",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Short:        "Interact with tekton hub",
		Long:         ``,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if hubType != hub.ArtifactHubType && hubType != hub.TektonHubType {
				return fmt.Errorf("invalid hub type: %s, expecting artifact or tekton", hubType)
			}
			if err := cli.SetHub(hubType); err != nil {
				return err
			}
			return cli.Hub().SetURL(apiURL)
		},
	}

	cli.SetStream(cmd.OutOrStdout(), cmd.OutOrStderr())

	cmd.AddCommand(
		downgrade.Command(cli),
		get.Command(cli),
		info.Command(cli),
		install.Command(cli),
		reinstall.Command(cli),
		search.Command(cli),
		upgrade.Command(cli),
		check_upgrade.Command(cli),
	)

	cmd.PersistentFlags().StringVar(&apiURL, "api-server", "", "Hub API Server URL (default 'https://api.hub.tekton.dev' for 'tekton' type; default 'https://artifacthub.io' for 'artifact' type).\nURL can also be defined in a file '$HOME/.tekton/hub-config' with a variable 'TEKTON_HUB_API_SERVER'/'ARTIFACT_HUB_API_SERVER'.")
	cmd.PersistentFlags().StringVar(&hubType, "type", "tekton", "The type of Hub from where to pull the resource. Either 'artifact' or 'tekton'")

	return cmd
}
