// Copyright 2022 The Witness Contributors
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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// DirHashSha256 is the "h1:" directory hash function, using SHA-256.
//
// DirHashSha256 returns a SHA-256 hash of a summary
// prepared as if by the Unix command:
//
//	sha256sum $(find . -type f | sort) | sha256sum
//
// More precisely, the hashed summary contains a single line for each file in the list,
// ordered by sort.Strings applied to the file names, where each line consists of
// the hexadecimal SHA-256 hash of the file content,
// two spaces (U+0020), the file name, and a newline (U+000A).
//
// File names with newlines (U+000A) are disallowed.
func DirhHashSha256(files []string, open func(string) (io.ReadCloser, error)) (string, error) {
	h := sha256.New()
	files = append([]string(nil), files...)
	sort.Strings(files)
	for _, file := range files {
		if strings.Contains(file, "\n") {
			return "", errors.New("dirhash: filenames with newlines are not supported")
		}
		r, err := open(file)
		if err != nil {
			return "", err
		}
		defer r.Close()

		// In case the opened file was a symlink, check if it points to a directory.
		// If so, skip it.
		var abstractedFile interface{} = r
		osFile, ok := abstractedFile.(*os.File)
		if !ok {
			return "", errors.New("dirhash: abstracted file is not an *os.File")
		}
		if info, err := osFile.Stat(); err == nil && info.IsDir() {
			continue
		}

		hf := sha256.New()
		_, err = io.Copy(hf, r)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%x  %s\n", hf.Sum(nil), file)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
