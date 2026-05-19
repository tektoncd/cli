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

package git

import (
	"path/filepath"
	"strings"
)

// FetchSpec describes how to initialize and fetch from a Git repository.
type FetchSpec struct {
	URL         string
	SSHUrl      string
	Revision    string
	Path        string
	Depth       uint
	SSLVerify   bool
	CatalogName string
}

func (f *FetchSpec) sanitize() {
	f.URL = strings.TrimSpace(f.URL)
	f.SSHUrl = strings.TrimSpace(f.SSHUrl)
	f.Path = strings.TrimSpace(f.Path)
	f.Revision = strings.TrimSpace(f.Revision)
	f.CatalogName = strings.TrimSpace(f.CatalogName)
}

func (f *FetchSpec) clonePath() string {
	f.sanitize()
	return filepath.Join(f.Path, f.CatalogName)
}
