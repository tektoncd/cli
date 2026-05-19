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

package parser

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/cmd/hub/git"
	"go.uber.org/zap"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

type fakeRepo struct {
	path         string
	head         string
	modifiedTime map[string]time.Time
}

var _ git.Repo = (*fakeRepo)(nil)

func (r fakeRepo) Path() string {
	return r.path
}

func (r fakeRepo) Head() string {
	return r.head
}

func (r fakeRepo) ModifiedTime(path string) (time.Time, error) {

	rp, _ := r.RelPath(path)
	return r.modifiedTime[rp], nil
}

func (r fakeRepo) RelPath(f string) (string, error) {
	return filepath.Rel(r.path, f)
}

func TestParse_NonExistentRepo(t *testing.T) {
	repo := fakeRepo{
		path: "./testdata/catalogs/non-existent",
	}

	p := ForCatalog(zap.NewNop().Sugar(), repo, "")
	_, result := p.Parse()
	assert.Equal(t, 1, len(result.Errors))
	assert.Equal(t, "no resources found in repo", result.Error())

}

func TestParse_ValidRepo(t *testing.T) {
	now := time.Now()
	repo := fakeRepo{
		path: "./testdata/catalogs/valid",
		modifiedTime: map[string]time.Time{
			"task/maven/0.1/maven.yaml": now,
		},
	}

	p := ForCatalog(zap.NewNop().Sugar(), repo, "")
	res, result := p.Parse()

	assert.Equal(t, 0, len(result.Errors))
	assert.Equal(t, "", result.Error())

	assert.Equal(t, 2, len(res))
	gitCLI := res[0]
	assert.Equal(t, "git-cli", gitCLI.Name)
	assert.Equal(t, 1, len(gitCLI.Versions))
	assert.Equal(t, "linux/s390x", gitCLI.Versions[0].Platforms[0])

	maven := res[1]
	assert.Equal(t, "maven", maven.Name)
	assert.Equal(t, 2, len(maven.Versions))
	assert.Equal(t, now, maven.Versions[0].ModifiedAt)
	assert.Equal(t, "linux/amd64", maven.Versions[0].Platforms[0])
	assert.Equal(t, "linux/ppc64le", maven.Versions[1].Platforms[0])
	assert.Equal(t, "linux/s390x", maven.Versions[1].Platforms[1])
}

func TestParse_InvalidTask(t *testing.T) {
	// invalid task is ignored but result must have the issue it found
	repo := fakeRepo{
		path: "./testdata/catalogs/invalid-task",
	}

	p := ForCatalog(zap.NewNop().Sugar(), repo, "")
	res, result := p.Parse()

	assert.Equal(t, 0, len(res))

	assert.Equal(t, 1, len(result.Errors))
	assert.Equal(t, "no resources found in repo", result.Error())

	assert.Equal(t, 1, len(result.Issues))
	issue := result.Issues[0]
	assert.Assert(t, cmp.Contains(issue.Message, "git-cli is missing mandatory version label"))
	assert.Equal(t, Critical, issue.Type)
}

func TestParse_InvalidFilename(t *testing.T) {
	// invalid task is ignored but result must have the issue it found
	repo := fakeRepo{
		path: "./testdata/catalogs/invalid-taskname",
	}

	p := ForCatalog(zap.NewNop().Sugar(), repo, "")
	res, result := p.Parse()

	assert.Equal(t, 2, len(res)) // one maven should be found and a git-clone

	assert.Equal(t, 0, len(result.Errors))

	assert.Equal(t, 2, len(result.Issues))
	gitCLI := result.Issues[0]
	assert.Assert(t, cmp.Contains(gitCLI.Message, "failed to find any resource matching"))
	assert.Equal(t, Critical, gitCLI.Type)

	gitClone := result.Issues[1]
	assert.Assert(t, cmp.Contains(gitClone.Message, "expected to find 2 versions but found only 1"))
	assert.Equal(t, Critical, gitClone.Type)
}
