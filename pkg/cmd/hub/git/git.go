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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/mitchellh/go-homedir"
	"go.uber.org/zap"
)

type Client interface {
	Fetch(spec FetchSpec) (Repo, error)
}

type client struct {
	log *zap.SugaredLogger
}

func New(log *zap.SugaredLogger) Client {
	return &client{log: log}
}

// Fetch fetches the specified git repository at the revision into path.
func (c *client) Fetch(spec FetchSpec) (Repo, error) {
	spec.sanitize()
	log := c.log.With("name", "git")
	if err := ensureHomeEnv(log); err != nil {
		return nil, err
	}

	log.With("path", spec.clonePath()).Info("cloning")

	repo, err := c.initRepo(spec)
	if err != nil {
		os.RemoveAll(spec.clonePath())
		return nil, err
	}

	fetchArgs := []string{"fetch", "--recurse-submodules=yes", "origin", spec.Revision}

	if _, err := git(log, "", fetchArgs...); err != nil {
		// Fetch can fail if an old commit id was used so try git pull, performing regardless of error
		// as no guarantee that the same error is returned by all git servers gitlab, github etc...
		if _, err := git(log, "", "pull", "--recurse-submodules=yes", "origin"); err != nil {
			log.Info("Failed to pull origin", "err", err)
		}
		if _, err := git(log, "", "checkout", spec.Revision); err != nil {
			return nil, err
		}
	} else if _, err := git(log, "", "reset", "--hard", "FETCH_HEAD"); err != nil {
		return nil, err
	}
	log.With("url", spec.URL, "revision", spec.Revision, "path", repo.path).Info("successfully cloned")

	return repo, nil
}

func (c *client) initRepo(spec FetchSpec) (*LocalRepo, error) {

	cloneUrl := spec.URL

	if spec.SSHUrl != "" {
		cloneUrl = spec.SSHUrl
	}

	log := c.log.With("name", "repo").With("url", cloneUrl)

	clonePath := spec.clonePath()
	repo := &LocalRepo{path: clonePath}

	// if already cloned, cd to the cloned path
	if _, err := os.Stat(clonePath); err == nil {
		if err := os.Chdir(clonePath); err != nil {
			return nil, fmt.Errorf("failed to change directory with path %s; err: %w", clonePath, err)
		}
		return repo, nil
	}

	if _, err := git(log, "", "init", clonePath); err != nil {
		return nil, err
	}

	if err := os.Chdir(clonePath); err != nil {
		return nil, fmt.Errorf("failed to change directory with path %s; err: %w", spec.Path, err)
	}

	if _, err := git(log, "", "remote", "add", "origin", cloneUrl); err != nil {
		return nil, err
	}

	if _, err := git(log, "", "config", "http.sslVerify", strconv.FormatBool(spec.SSLVerify)); err != nil {
		log.Error(err, "failed to set http.sslVerify in git configs")
		return nil, err
	}
	return repo, nil
}

func ensureHomeEnv(log *zap.SugaredLogger) error {
	// HACK: This is to get git+ssh to work since ssh doesn't respect the HOME
	// env variable.
	homepath, err := homedir.Dir()
	if err != nil {
		log.Error(err, "Unexpected error: getting the user home directory")
		return err
	}
	homeenv := os.Getenv("HOME")
	euid := os.Geteuid()
	// Special case the root user/directory
	if euid == 0 {
		if err := os.Symlink(homeenv+"/.ssh", "/root/.ssh"); err != nil {
			// Only do a warning, in case we don't have a real home
			// directory writable in our image
			log.Error(err, "Unexpected error: creating symlink")
		}
	} else if homeenv != "" && homeenv != homepath {
		if _, err := os.Stat(homepath + "/.ssh"); os.IsNotExist(err) {
			if err := os.Symlink(homeenv+"/.ssh", homepath+"/.ssh"); err != nil {
				// Only do a warning, in case we don't have a real home
				// directory writable in our image
				log.Error(err, "Unexpected error: creating symlink: %v", err)
			}
		}
	}
	return nil
}

func git(log *zap.SugaredLogger, kind string, args ...string) (string, error) {
	output, err := rawGit(kind, args...)

	if err != nil {
		log.Errorw(
			"executedCommand", fmt.Sprintf("git %s", strings.Join(args, " ")),
			"executedCommandOutput", output,
			err,
		)
		return output, err
	}
	return output, nil
}

func rawGit(dir string, args ...string) (string, error) {
	c := exec.Command("git", args...)
	var output bytes.Buffer
	c.Stderr = &output
	c.Stdout = &output
	// This is the optional working directory. If not set, it defaults to the current
	// working directory of the process.
	if dir != "" {
		c.Dir = dir
	}
	err := c.Run()
	return output.String(), err
}
