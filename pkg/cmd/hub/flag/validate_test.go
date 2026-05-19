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

package flag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInList(t *testing.T) {

	// Valid Case
	err := InList("match", "abc", []string{"abc", "def"})
	assert.NoError(t, err)

	// Invalid Case
	err = InList("match", "xyz", []string{"abc", "def"})
	assert.EqualError(t, err, "invalid value \"xyz\" set for option match. Valid options: [abc, def]")
}

func TestTrimArray(t *testing.T) {

	input := []string{"a,b", "abc", "xyz,mno"}
	res := TrimArray(input)
	want := []string{"a", "b", "abc", "xyz", "mno"}
	assert.Equal(t, want, res)
}

func TestValidateVersion(t *testing.T) {

	// Valid Case
	err := ValidateVersion("0.1")
	assert.NoError(t, err)

	err = ValidateVersion("0.1.1")
	assert.NoError(t, err)

	// Invalid Case
	err = ValidateVersion("abc")
	assert.EqualError(t, err, "invalid value \"abc\" set for option version. valid eg. 0.1, 1.2.1")
}
