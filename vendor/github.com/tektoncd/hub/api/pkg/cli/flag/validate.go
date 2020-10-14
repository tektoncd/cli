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
	"fmt"
	"regexp"
	"strings"
)

// InList validates if a value of a flag is in the array passed to it.
func InList(option, val string, list []string) error {
	val = strings.ToLower(val)

	for _, v := range list {
		if v == val {
			return nil
		}
	}
	return fmt.Errorf("invalid value %q set for option %s. Valid options: [%s]",
		val, option, strings.Join(list, ", "))
}

// TrimArray Splits the array by `,` & ' '(space) and returns an array
// eg. [abc,def mno xyz] -> [abc def mno xyz]
func TrimArray(arr []string) []string {
	input := strings.Trim(fmt.Sprint(arr), "[]")
	return strings.FieldsFunc(input, func(r rune) bool { return r == ' ' || r == ',' })
}

// AllEmpty checks if all the passed arrays are empty
func AllEmpty(arr ...[]string) bool {
	for _, a := range arr {
		if len(a) != 0 {
			return false
		}
	}
	return true
}

// ValidateVersion validates version format
func ValidateVersion(version string) error {
	if version == "" {
		return nil
	}
	var re = regexp.MustCompile(`^(\d+\.)?(\d+\.)?(\*|\d+)$`)
	if !re.MatchString(version) {
		return fmt.Errorf("invalid value %q set for option version. valid eg. 0.1, 1.2.1", version)
	}
	return nil
}
