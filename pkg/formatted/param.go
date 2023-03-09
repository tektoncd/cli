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
	"fmt"
	"strings"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

// Param returns params with their values. If user value is not defined then returns default value,
// if default value is not defined then returns param's type
func Param(params []v1.Param, paramSpec []v1.ParamSpec) string {
	if len(params) == 0 {
		return "---"
	}
	var str string
	for i, param := range params {
		paramValue := ""
		switch param.Value.Type {
		case "string":
			paramValue = CheckParamDefaultValue(param.Value.StringVal, paramSpec)
		case "array":
			paramValue = "["
			for j, pv := range param.Value.ArrayVal {
				pv = CheckParamDefaultValue(pv, paramSpec)
				if j == len(param.Value.ArrayVal)-1 {
					paramValue += " " + pv + " ]"
				} else {
					paramValue += " " + pv + ","
				}
			}
		default:
			paramValue = "{"
			j := 0
			for k, v := range param.Value.ObjectVal {
				pv := CheckParamDefaultValue(v, paramSpec)
				if j == len(param.Value.ObjectVal)-1 {
					paramValue += fmt.Sprintf(" %s: %s }", k, pv)
				} else {
					paramValue += fmt.Sprintf(" %s: %s ,", k, pv)
				}
				j++
			}
		}
		if i == len(params)-1 {
			str += fmt.Sprintf("%s: %s", param.Name, paramValue)
		} else {
			str += fmt.Sprintf("%s: %s, ", param.Name, paramValue)
		}
	}
	return str
}

// CheckParamDefaultValue returns param's value if defined, if not then checks for default value
// If default value is not defined then returns param's type
func CheckParamDefaultValue(param string, paramSpec []v1.ParamSpec) string {
	if strings.ContainsAny(param, "$") {
		paramValue := ""
		replacer := strings.NewReplacer("$", "", "(", "", ")", "", "params.", "")
		paramName := replacer.Replace(param)
		for _, spec := range paramSpec {
			if spec.Name == paramName {
				if spec.Default == nil {
					paramValue = string(spec.Type)
					break
				}
				switch spec.Default.Type {
				case "string":
					paramValue = spec.Default.StringVal
				case "array":
					pv := ""
					for k, val := range spec.Default.ArrayVal {
						if k == 0 {
							pv += val
						} else {
							pv += " " + val
						}
					}
					paramValue = pv
				default:
					pv := "{"
					first := true
					for k, val := range spec.Default.ObjectVal {
						if !first {
							pv += ","
						}
						pv += k + ":" + val
						first = false
					}
					paramValue = pv + "}"
				}
				break
			}
		}
		return paramValue
	}
	return param
}
