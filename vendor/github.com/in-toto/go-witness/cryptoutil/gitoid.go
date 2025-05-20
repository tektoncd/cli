// Copyright 2023 The Witness Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cryptoutil

import (
	"bytes"
	"crypto"
	"encoding/hex"
	"fmt"

	"github.com/edwarnicke/gitoid"
)

// gitoidHasher implements io.Writer so we can generate gitoids with our CalculateDigestSet function.
// CalculateDigestSet takes in an io.Reader pointing to some data we want to hash, and writes it to a
// MultiWriter that forwards it to writers for each hash we wish to calculate.
// This is a bit hacky -- it maintains an internal buffer and then when asked for the Sum, it calculates
// the gitoid. We may be able to contribute to the gitoid library to make this smoother
type gitoidHasher struct {
	buf  *bytes.Buffer
	hash crypto.Hash
}

// Write implments the io.Writer interface, and writes to the internal buffer
func (gh *gitoidHasher) Write(p []byte) (n int, err error) {
	return gh.buf.Write(p)
}

// Sum appends the current hash to b and returns the resulting slice.
// It does not change the underlying hash state.
func (gh *gitoidHasher) Sum(b []byte) []byte {
	opts := []gitoid.Option{}
	if gh.hash == crypto.SHA256 {
		opts = append(opts, gitoid.WithSha256())
	}

	g, err := gitoid.New(gh.buf, opts...)
	if err != nil {
		return []byte{}
	}

	return append(b, []byte(g.URI())...)
}

// Reset resets the Hash to its initial state.
func (gh *gitoidHasher) Reset() {
	gh.buf = &bytes.Buffer{}
}

// Size returns the number of bytes Sum will return.
func (gh *gitoidHasher) Size() int {
	hashName, err := HashToString(gh.hash)
	if err != nil {
		return 0
	}

	// this is somewhat fragile and knows too much about the internals of the gitoid code...
	// we're assuming that the default gitoid content type will remain BLOB, and that our
	// string representations of hash functions will remain consistent with their...
	// and that the URI format will remain consistent.
	// this should probably be changed, and this entire thing could maybe be upstreamed to the
	// gitoid library.
	return len(fmt.Sprintf("gitoid:%s:%s:", gitoid.BLOB, hashName)) + hex.EncodedLen(gh.hash.Size())
}

// BlockSize returns the hash's underlying block size.
// The Write method must be able to accept any amount
// of data, but it may operate more efficiently if all writes
// are a multiple of the block size.
func (gh *gitoidHasher) BlockSize() int {
	hf := gh.hash.New()
	return hf.BlockSize()
}
