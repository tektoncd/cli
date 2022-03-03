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
	"strings"
)

// kinds defines the list of supported Tekton kind
var kinds = []string{
	"Task",
	"Pipeline",
}

// SupportedKinds returns list of supported Tekton kind
func SupportedKinds() []string {
	return kinds
}

// IsSupportedKind checks if passed kind is supported
func IsSupportedKind(kind string) bool {

	for _, k := range kinds {
		if strings.EqualFold(k, kind) {
			return true
		}
	}
	return false
}
