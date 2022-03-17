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

package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cmd"
	"github.com/tektoncd/cli/pkg/plugins"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	tp := &cli.TektonParams{}
	tkn := cmd.Root(tp)

	args := os.Args[1:]
	cmd, _, _ := tkn.Find(args)
	if cmd != nil && cmd == tkn && len(args) > 0 {
		exCmd, err := plugins.FindPlugin(os.Args[1])
		// if we can't find command then execute the normal tkn command.
		if err != nil {
			goto CoreTkn
		}

		// if we have found the plugin then sysexec it by replacing current process.
		if err := syscall.Exec(exCmd, append([]string{exCmd}, os.Args[2:]...), os.Environ()); err != nil {
			fmt.Fprintf(os.Stderr, "Command finished with error: %v", err)
			os.Exit(127)
		}
		return
	}

CoreTkn:
	if err := tkn.Execute(); err != nil {
		os.Exit(1)
	}
}
