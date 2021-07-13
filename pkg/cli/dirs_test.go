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
	"os"
	"path/filepath"
	"testing"
)

func Test_contractHome_base(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Errorf("Failed to get home: %w", err)
	}
	expected := "~"
	got, err := contractHome(homeDir)
	if err != nil {
		t.Errorf("Failed to contract home: %w", err)
	}
	if got != expected {
		t.Errorf("Result should be '%s' != '%s'", expected, got)
	}
}

func Test_contractHome_suffix(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Errorf("Failed to get home: %w", err)
	}
	suffix := "abc"
	expected := filepath.Join("~", suffix)
	got, err := contractHome(filepath.Join(homeDir, suffix))
	if err != nil {
		t.Errorf("Failed to contract home: %w", err)
	}
	if got != expected {
		t.Errorf("Result should be '%s' != '%s'", expected, got)
	}
}

func Test_contractHome_empty(t *testing.T) {
	expected := ""
	got, err := contractHome("")
	if err != nil {
		t.Errorf("Failed to contract home: %w", err)
	}
	if got != expected {
		t.Errorf("Result should be empty")
	}
}

func Test_contractHome_root(t *testing.T) {
	got, err := contractHome("/")
	if err == nil {
		t.Errorf("Error expected here, got this result: '%s'", got)
	}
}
