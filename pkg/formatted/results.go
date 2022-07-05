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

package formatted

import (
	"encoding/json"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// Result will format a given result value
func Result(value v1beta1.ArrayOrString) string {
	switch value.Type {
	case v1beta1.ParamTypeString:
		// remove trailing new-line from value
		return strings.TrimSuffix(value.StringVal, "\n")
	case v1beta1.ParamTypeArray:
		return strings.Join(value.ArrayVal, ", ")
	case v1beta1.ParamTypeObject:
		// FIXME: do not ignore the error
		v, _ := json.Marshal(value.ObjectVal)
		return string(v)
	}
	return "<invalid result type>"
}

// ResultString will format a given result value.
// This is a deprecated function, that should get removed once all results
// are using ArrayOrString.
// Deprecated
func ResultString(value string) string {
	// remove trailing new-line from value
	return strings.TrimSuffix(value, "\n")
}
