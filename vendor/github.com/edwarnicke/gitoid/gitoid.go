// Copyright (c) 2022 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitoid

import (
	"bytes"
	"crypto/sha1" // #nosec G505
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

// GitObjectType type of git object - current values are "blob", "commit", "tag", "tree".
type GitObjectType string

const (
	BLOB   GitObjectType = "blob"
	COMMIT GitObjectType = "commit"
	TAG    GitObjectType = "tag"
	TREE   GitObjectType = "tree"
)

var ErrMayNotBeNil = errors.New("may not be nil")
var ErrInvalidGitOIDURI = errors.New("invalid uri in gitoid.FromURI")

type GitOID struct {
	gitObjectType GitObjectType
	hashName      string
	hashValue     []byte
}

// New - create a new GitOID
//       by default git object type is "blob" and hash is sha1
func New(reader io.Reader, opts ...Option) (*GitOID, error) {
	if reader == nil {
		return nil, fmt.Errorf("reader in gitoid.New: %w", ErrMayNotBeNil)
	}

	o := &option{
		gitObjectType: BLOB,
		/* #nosec G401 */
		h:             sha1.New(),
		hashName:      "sha1",
		contentLength: 0,
	}

	for _, opt := range opts {
		opt(o)
	}

	// If there is no declared o.contentLength, copy the entire reader into a buffer so we can compute
	// the contentLength
	if o.contentLength == 0 {
		buf := bytes.NewBuffer(nil)

		contentLength, err := io.Copy(buf, reader)
		if err != nil {
			return nil, fmt.Errorf("error copying reader to buffer in gitoid.New: %w", err)
		}

		reader = buf
		o.contentLength = contentLength
	}

	// Write the git object header
	o.h.Write(Header(o.gitObjectType, o.contentLength))

	// Copy the reader to the hash
	n, err := io.Copy(o.h, io.LimitReader(reader, o.contentLength))
	if err != nil {
		return nil, fmt.Errorf("error copying reader to hash.Hash.Writer in gitoid.New: %w", err)
	}

	if n < o.contentLength {
		return nil, fmt.Errorf("expected contentLength (%d) is less than actual contentLength (%d) in gitoid.New: %w", o.contentLength, n, io.ErrUnexpectedEOF)
	}

	return &GitOID{
		gitObjectType: o.gitObjectType,
		hashName:      o.hashName,
		hashValue:     o.h.Sum(nil),
	}, nil
}

// Header - returns the git object header from the gitObjectType and contentLength.
func Header(gitObjectType GitObjectType, contentLength int64) []byte {
	return []byte(fmt.Sprintf("%s %d\000", gitObjectType, contentLength))
}

// String - returns the gitoid in lowercase hex.
func (g *GitOID) String() string {
	return fmt.Sprintf("%x", g.hashValue)
}

// URI - returns the gitoid as a URI (https://www.iana.org/assignments/uri-schemes/prov/gitoid)
func (g *GitOID) URI() string {
	return fmt.Sprintf("gitoid:%s:%s:%s", g.gitObjectType, g.hashName, g)
}

func (g *GitOID) Bytes() []byte {
	if g == nil {
		return nil
	}

	return g.hashValue
}

// Equal - returns true of g == x.
func (g *GitOID) Equal(x *GitOID) bool {
	if g == x {
		return true
	}

	if g == nil || x == nil || g.hashName != x.hashName {
		return false
	}

	if len(g.Bytes()) != len(x.Bytes()) {
		return false
	}

	for i, v := range g.Bytes() {
		if x.Bytes()[i] != v {
			return false
		}
	}
	return true
}

// FromURI - returns a *GitOID from a gitoid uri string - see https://www.iana.org/assignments/uri-schemes/prov/gitoid
func FromURI(uri string) (*GitOID, error) {
	parts := strings.Split(uri, ":")
	if len(parts) != 4 || parts[0] != "gitoid" {
		return nil, fmt.Errorf("%w: %q in gitoid.FromURI", ErrInvalidGitOIDURI, uri)
	}
	hashValue, err := hex.DecodeString(parts[3])
	if err != nil {
		return nil, fmt.Errorf("error decoding hash value (%s) in gitoid.FromURI: %w", parts[3], err)
	}
	return &GitOID{
		gitObjectType: GitObjectType(parts[1]),
		hashName:      parts[2],
		hashValue:     hashValue,
	}, nil
}

// Match - returns true if contents of reader generates a GitOID equal to g.
func (g *GitOID) Match(reader io.Reader) bool {
	g2, err := New(reader, WithGitObjectType(g.gitObjectType))
	if err != nil {
		return false
	}
	return g.Equal(g2)
}

// Find - return the first fs.File in paths that Matches the *GitOID g.
func (g *GitOID) Find(paths ...fs.FS) fs.File {
	foundFiles := g.findN(1, paths...)
	if len(foundFiles) != 1 {
		return nil
	}
	return foundFiles[0]
}

// FindAll - return all fs.Files in paths that Matches the *GitOID g.
func (g *GitOID) FindAll(paths ...fs.FS) []fs.File {
	return g.findN(0, paths...)
}

func (g *GitOID) findN(n int, paths ...fs.FS) []fs.File {
	var foundFiles []fs.File
	for _, fsys := range paths {
		_ = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if d == nil || d.IsDir() || err != nil {
				//lint:ignore nilerr - returning non-nil error will stop the walk
				return nil
			}
			file, err := fsys.Open(path)
			defer func() { _ = file.Close() }()
			if err != nil {
				//lint:ignore nilerr - returning non-nil error will stop the walk
				return nil
			}
			if !g.Match(file) {
				return nil
			}
			foundFile, err := fsys.Open(path)
			if err == nil {
				foundFiles = append(foundFiles, foundFile)
			}
			if n > 0 && len(foundFiles) == n {
				return io.EOF
			}
			return nil
		})
	}
	return foundFiles
}
