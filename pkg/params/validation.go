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

package params

import (
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/util/sets"
)

var allowedParamTypes = sets.NewString("string", "array", "object")

func ValidateParamType(params []v1beta1.ParamSpec) error {
	paramsWithInvalidType := make([]v1beta1.ParamSpec, 0)
	for _, param := range params {
		if !allowedParamTypes.Has(string(param.Type)) {
			paramsWithInvalidType = append(paramsWithInvalidType, param)
		}
	}

	if len(paramsWithInvalidType) > 0 {
		errString := "params does not have a valid type -"
		for _, param := range paramsWithInvalidType {
			errString += fmt.Sprintf(" '%s'", param.Name)
		}
		return fmt.Errorf(errString)
	}

	return nil
}
