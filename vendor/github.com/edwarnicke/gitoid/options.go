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
	"crypto/sha256"
	"hash"
)

type option struct {
	gitObjectType GitObjectType
	h             hash.Hash
	hashName      string
	contentLength int64
}

// Option - option for GitOID creation.
type Option func(o *option)

// WithSha256 - use sha256 for computing gitoids instead of the default sha1.
func WithSha256() Option {
	return func(o *option) {
		o.hashName = "sha256"
		o.h = sha256.New()
	}
}

// WithGitObjectType - set the GitOobjectType to a value different than the default gitoid.BLOB type.
func WithGitObjectType(gitObjectType GitObjectType) Option {
	return func(o *option) {
		o.gitObjectType = gitObjectType
	}
}

// WithContentLength - allows the assertion of a contentLength to be read from the provided reader
//                     only the first contentLength of data will be read from the reader
//                     if contentLength bytes are unavailable from the reader, an error will be returned.
func WithContentLength(contentLength int64) Option {
	return func(o *option) {
		o.contentLength = contentLength
	}
}
