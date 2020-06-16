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
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	tp := &cli.TektonParams{}
	tkn := cmd.Root(tp)

	args := os.Args[1:]
	_, _, err := tkn.Find(args)

	if err == nil {
		if err := tkn.Execute(); err != nil {
			os.Exit(1)
		}
		return
	}

	pluginCmd := "tkn-" + os.Args[1]

	exCmd, err := exec.LookPath(pluginCmd)

	if err != nil {
		fmt.Fprintf(os.Stderr, `
		============================================
		looked for: %s
		error:
		%s
		_____________________________________________
		`, pluginCmd, err)

		if err := tkn.Execute(); err != nil {
			os.Exit(1)
		}
	}

	fmt.Fprintf(os.Stderr, `
	============================================
	external binary: %s
	error:
	%s
	_____________________________________________
	`, exCmd, err)

	errX := syscall.Exec(exCmd, append([]string{exCmd}, os.Args[2:]...), os.Environ())
	log.Printf("Command finished with error: %v", errX)

}
