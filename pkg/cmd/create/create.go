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

package create

import (
	"log"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// runCmd is suitable for use with cobra.Command's Run field
type runCmd func(*cobra.Command, []string)

func Command() *cobra.Command {
	eg := `Create resources defined in foo.yaml in namespace 'bar':

	tkn create -f foo.yaml -n bar
`

	c := &cobra.Command{
		Use:          "create",
		Short:        "Uses kubectl create to create Kubernetes resources",
		Example:      eg,
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		Run: invokeCommand("kubectl"),
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
	}

	return c
}

// returns a runCmd that passes CLI arguments
// through to a binary named command
func invokeCommand(command string) runCmd {
	return func(_ *cobra.Command, _ []string) {

		if !isKubectlAvailable() {
			log.Print("error: kubectl is not available. kubectl must be installed to use tkn create.")
			return
		}
		// Start building a command line invocation by passing
		// through arguments to command
		cmd := exec.Command(command, os.Args[1:]...)

		// Pass through environment
		cmd.Env = os.Environ()
		// Pass through stdfoo
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin

		// Run command
		if err := cmd.Run(); err != nil {
			log.Printf("error executing %q command with args: %v; %v", command, os.Args[1:], err)
		}
	}
}

// check if kubectl is installed
func isKubectlAvailable() bool {
	cmd := exec.Command("kubectl")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
