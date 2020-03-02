// Copyright © 2019 The Tekton Authors.
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
	"errors"
	"fmt"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

const invalidParam = "invalid input format for param parameter: "

var paramByType = map[string]v1alpha1.ParamType{}

func MergeParam(p []v1alpha1.Param, optPar []string) ([]v1alpha1.Param, error) {
	params, err := parseParam(optPar)
	if err != nil {
		return nil, err
	}

	if len(params) == 0 {
		return p, nil
	}

	for i := range p {
		if v, ok := params[p[i].Name]; ok {
			p[i] = v
			delete(params, v.Name)
		}
	}

	for _, v := range params {
		p = append(p, v)
	}

	return p, nil
}

func parseParam(p []string) (map[string]v1alpha1.Param, error) {
	params := map[string]v1alpha1.Param{}
	for _, v := range p {
		r := strings.SplitN(v, "=", 2)
		if len(r) != 2 {
			return nil, errors.New(invalidParam + v)
		}

		if _, ok := paramByType[r[0]]; !ok {
			return nil, fmt.Errorf("param '%s' not present in spec", r[0])
		}

		param := v1alpha1.Param{
			Name: r[0],
			Value: v1alpha1.ArrayOrString{
				Type: paramByType[r[0]],
			},
		}

		if paramByType[r[0]] == "string" {
			param.Value.StringVal = r[1]
		}

		if paramByType[r[0]] == "array" {
			param.Value.ArrayVal = strings.Split(r[1], ",")
		}
		params[r[0]] = param
	}
	return params, nil
}

func FilterParamsByType(params []v1alpha1.ParamSpec) {
	for _, p := range params {
		if p.Type == "string" {
			paramByType[p.Name] = v1alpha1.ParamTypeString
			continue
		}
		paramByType[p.Name] = v1alpha1.ParamTypeArray
	}
}
