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
	"sort"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const invalidParam = "invalid input format for param parameter: "

var paramByType = map[string]v1beta1.ParamType{}

func MergeParam(p []v1beta1.Param, optPar []string) ([]v1beta1.Param, error) {
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
	sort.Slice(p, func(i, j int) bool { return p[i].Name < p[j].Name })
	return p, nil
}

func parseParam(p []string) (map[string]v1beta1.Param, error) {
	params := map[string]v1beta1.Param{}
	for _, v := range p {
		r := strings.SplitN(v, "=", 2)
		if len(r) != 2 {
			return nil, errors.New(invalidParam + v)
		}

		if _, ok := paramByType[r[0]]; !ok {
			return nil, fmt.Errorf("param '%s' not present in spec", r[0])
		}

		param := v1beta1.Param{
			Name: r[0],
			Value: v1beta1.ParamValue{
				Type: paramByType[r[0]],
			},
		}

		if paramByType[r[0]] == "string" {
			param.Value.StringVal = r[1]
		}

		if paramByType[r[0]] == "array" {
			if len(r[1]) == 0 {
				param.Value.ArrayVal = make([]string, 0)
			} else {
				param.Value.ArrayVal = strings.Split(r[1], ",")
			}
		}

		if paramByType[r[0]] == "object" {
			fields := strings.Split(r[1], ",")
			object := map[string]string{}
			for _, field := range fields {
				r := strings.SplitN(field, ":", 2)
				if len(r) != 2 {
					return nil, errors.New(invalidParam + v)
				}
				object[strings.TrimSpace(r[0])] = strings.TrimSpace(r[1])
			}
			param.Value.ObjectVal = object
		}
		params[r[0]] = param
	}
	return params, nil
}

func FilterParamsByType(params []v1beta1.ParamSpec) {
	for _, p := range params {
		if p.Type == "string" {
			paramByType[p.Name] = v1beta1.ParamTypeString
			continue
		}
		if p.Type == "array" {
			paramByType[p.Name] = v1beta1.ParamTypeArray
			continue
		}
		paramByType[p.Name] = v1beta1.ParamTypeObject
	}
}

// ParseParams parse the params and return as map
func ParseParams(params []string) (map[string]string, error) {
	parsedParams := make(map[string]string)
	for _, p := range params {
		r := strings.SplitN(p, "=", 2)
		if len(r) != 2 {
			return nil, errors.New(invalidParam + p)
		}
		parsedParams[r[0]] = r[1]
	}
	return parsedParams, nil
}
