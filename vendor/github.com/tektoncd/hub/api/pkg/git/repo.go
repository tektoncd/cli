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
	"time"
)

type Repo interface {
	Path() string
	Head() string
	ModifiedTime(path string) (time.Time, error)
	RelPath(path string) (string, error)
}

type LocalRepo struct {
	path string
	head string
}

func (r LocalRepo) Path() string {
	return r.path
}

func (r LocalRepo) Head() string {
	if r.head == "" {
		head, _ := rawGit("", "rev-parse", "HEAD")
		r.head = strings.TrimSuffix(head, "\n")
	}
	return r.head
}

// ModifiedTime returns the modified (commited/changed) time of the file by
// querying the git history.
func (r LocalRepo) ModifiedTime(path string) (time.Time, error) {
	gitPath, err := r.RelPath(path)
	if err != nil {
		return time.Time{}, err
	}

	commitedAt, err := rawGit(r.path, "log", "-1", "--pretty=format:%cI", gitPath)
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339, commitedAt)
}

// RelPath returns the path of file relative to repo's path
func (r LocalRepo) RelPath(file string) (string, error) {
	return filepath.Rel(r.path, file)
}
