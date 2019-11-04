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
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

const invalidParam = "invalid input format for param parameter: "

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
		values := strings.Split(r[1], ",")
		if len(values) == 1 {
			params[r[0]] = v1alpha1.Param{
				Name: r[0],
				Value: v1alpha1.ArrayOrString{
					Type:      v1alpha1.ParamTypeString,
					StringVal: r[1],
				},
			}
		}
		if len(values) > 1 {
			params[r[0]] = v1alpha1.Param{
				Name: r[0],
				Value: v1alpha1.ArrayOrString{
					Type:     v1alpha1.ParamTypeArray,
					ArrayVal: values,
				},
			}
		}
	}
	return params, nil
}
