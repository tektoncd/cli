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

package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func CacheDir() string {
	tmpDir, err := os.UserCacheDir()
	if err != nil {
		tmpDir = os.TempDir()
	}
	tmpDir, err = contractHome(tmpDir)
	if err != nil {
		tmpDir = os.TempDir()
	}
	return filepath.Join(tmpDir, "tkn")
}

func ConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	configDir, err = contractHome(configDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "tkn"), err
}

// opposite of homedir.Expand()
func contractHome(path string) (string, error) {
	if len(path) == 0 {
		return path, nil
	}

	if path[0] == '~' {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(path, homeDir) {
		return "", errors.New("cannot contract user-specific home dir")
	}

	return filepath.Join("~", path[len(homeDir):]), nil
}
