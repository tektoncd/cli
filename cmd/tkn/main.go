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
	"os/exec"
	"path/filepath"
	"syscall"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

const (
	pluginDirEnv = "TKN_PLUGINS_DIR"
	pluginDir    = "~/.config/tkn/plugins"
)

func main() {
	tp := &cli.TektonParams{}
	tkn := cmd.Root(tp)

	args := os.Args[1:]
	cmd, _, _ := tkn.Find(args)
	if cmd != nil && cmd == tkn && len(args) > 0 {
		pluginCmd := "tkn-" + os.Args[1]
		pluginDir, err := getPluginDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting plugin folder: %v", err)
		}
		exCmd, err := findBinaryPluginDir(pluginDir, pluginCmd)
		if err != nil {
			// If we can't find the plugin in the plugin dir, try in the user
			// PATH
			exCmd, err = exec.LookPath(pluginCmd)
			if err != nil {
				goto CoreTkn
			}
		}

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

func getPluginDir() (string, error) {
	dir := os.Getenv(pluginDirEnv)
	// if TKN_PLUGINS_DIR is set, follow it
	if dir != "" {
		return dir, nil
	}
	// Respect XDG_CONFIG_HOME if set
	if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
		return filepath.Join(xdgHome, "tkn", "plugins"), nil
	}
	// Fallback to default pluginDir (~/.config/tkn/plugins)
	return homedir.Expand(pluginDir)
}

// Find a binary in Plugin Directory
func findBinaryPluginDir(dir, cmd string) (string, error) {
	path := filepath.Join(dir, cmd)
	_, err := os.Stat(path)
	if err == nil {
		// Found in dir
		return path, nil
	}
	if !os.IsNotExist(err) {
		return "", errors.Wrap(err, fmt.Sprintf("i/o error while reading %s", path))
	}
	return "", err
}
