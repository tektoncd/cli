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
	desc = `Use to generate argument completion script 
for one of the supported shells below. Next, invoke the script, and 
then have interactive tab completion. 

Supported Shells:
	- bash
	- zsh
	- fish
	- powershell
`
	examples = `
Bash:
# Execute the following once:
$ source <(tkn completion bash)

# To load completions for each session, execute once:
Linux:
  $ tkn completion bash > /etc/bash_completion.d/tkn

MacOS:
  $ tkn completion bash > /usr/local/etc/bash_completion.d/tkn

Zsh:

# Execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ tkn completion zsh > "${fpath[1]}/_tkn"

# Restart shell for this setup to take effect.

Fish:
# Execute the following once:
$ tkn completion fish | source

# To load completions for each session, execute once:
$ tkn completion fish > ~/.config/fish/completions/tkn.fish

PowerShell:
 
# Example 1: 
PS> ./tkn completion powershell | Out-String | Invoke-Expression

#Example 2: 
# To load completions for every new session, run:
PS> ./tkn completion powershell > Register-TektonArgumentCompleter.ps1

# and source this file from your PowerShell profile.
`
)

func Command() *cobra.Command {
	var cmd = &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate argument completion script",
		Long:      desc,
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Example:   examples,
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
				_ = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
	return cmd
}
