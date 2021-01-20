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

package completion

import (
	"os"

	"github.com/spf13/cobra"
)

const (
	desc = `This command prints shell completion code which must be evaluated to provide
interactive completion

Supported Shells:
	- bash
	- zsh
	- fish
	- powershell
`
	eg = `To load completions:

Bash:

$ source <(tkn completion bash)

# To load completions for each session, execute once:
Linux:
  $ tkn completion bash > /etc/bash_completion.d/tkn

MacOS:
  $ tkn completion bash > /usr/local/etc/bash_completion.d/tkn

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ tkn completion zsh > "${fpath[1]}/_tkn"

# You will need to start a new shell for this setup to take effect.

Fish:

$ tkn completion fish | source

# To load completions for each session, execute once:
$ tkn completion fish > ~/.config/fish/completions/tkn.fish
`
)

func Command() *cobra.Command {
	var cmd = &cobra.Command{
		Use:       "completion [SHELL]",
		Short:     "Prints shell completion scripts",
		Long:      desc,
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Example:   eg,
		Annotations: map[string]string{
			"commandType": "utility",
		},
		Args: cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				_ = cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				_ = cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				_ = cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				_ = cmd.Root().GenPowerShellCompletion(os.Stdout)
			}

			return nil
		},
	}
	return cmd
}
