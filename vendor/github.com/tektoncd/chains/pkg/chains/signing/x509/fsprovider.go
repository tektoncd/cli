//
// Copyright 2023 The Sigstore Authors.
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

package x509

import (
	"context"
	"os"

	"github.com/sigstore/cosign/v2/pkg/providers"
	"github.com/sigstore/cosign/v2/pkg/providers/filesystem"
)

const (
	fsCustomTokenPathProvider        = "filesystem-custom-path"
	fsDefaultCosignTokenPathProvider = "filesystem"
)

func init() {
	providers.Register(fsCustomTokenPathProvider, &filesystemCustomPath{})
}

type filesystemCustomPath struct{}

var _ providers.Interface = (*filesystemCustomPath)(nil)

var (
	// FilesystemTokenPath is the path to where we read an OIDC
	// token from the filesystem. This is the hard-coded value from cosign.
	// If identity.token.file is configured, this variable will be updated to match.
	// nolint
	FilesystemTokenPath = filesystem.FilesystemTokenPath
)

// Enabled implements providers.Interface
func (ga *filesystemCustomPath) Enabled(ctx context.Context) bool {
	// If we can stat the file without error then this is enabled.
	_, err := os.Stat(FilesystemTokenPath)
	return err == nil
}

// Provide implements providers.Interface
func (ga *filesystemCustomPath) Provide(ctx context.Context, audience string) (string, error) {
	b, err := os.ReadFile(FilesystemTokenPath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
