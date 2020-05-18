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
	"testing"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"gotest.tools/v3/assert"
)

func TestParam(t *testing.T) {
	paramSpec := []v1beta1.ParamSpec{
		{
			Name:    "foo",
			Type:    v1beta1.ParamTypeString,
			Default: &v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "astring"},
		},
		{
			Name:    "bar",
			Type:    v1beta1.ParamTypeString,
			Default: &v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "bstring"},
		},
		{
			Name:    "foo-array-1",
			Type:    v1beta1.ParamTypeArray,
			Default: &v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"a1", "a2"}},
		},
		{
			Name:    "foo-array-2",
			Type:    v1beta1.ParamTypeArray,
			Default: &v1beta1.ArrayOrString{Type: v1beta1.ParamTypeArray, ArrayVal: []string{"b1", "b2"}},
		},
		{
			Name: "no-def-val",
			Type: v1beta1.ParamTypeString,
		},
		{
			Name: "no-def-val-1",
			Type: v1beta1.ParamTypeArray,
		},
	}
	p := []v1beta1.Param{
		{
			Name: "foo",
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: "bar",
			},
		},
		{
			Name: "foo-1",
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: "bar-1",
			},
		},
	}
	p1 := []v1beta1.Param{
		{
			Name: "foo",
			Value: v1beta1.ArrayOrString{
				Type:     v1beta1.ParamTypeArray,
				ArrayVal: []string{"v1", "v2"},
			},
		},
		{
			Name: "foo-bar",
			Value: v1beta1.ArrayOrString{
				Type:     v1beta1.ParamTypeArray,
				ArrayVal: []string{"v3", "v4", "v5"},
			},
		},
	}
	p2 := []v1beta1.Param{
		{
			Name: "foo",
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: "$(params.foo)",
			},
		},
		{
			Name: "foo-array",
			Value: v1beta1.ArrayOrString{
				Type:     v1beta1.ParamTypeArray,
				ArrayVal: []string{"$(params.foo-array-1)", "$(params.foo-array-2)", "last"},
			},
		},
	}
	p3 := []v1beta1.Param{
		{
			Name: "foo",
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: "$(no-def-val)",
			},
		},
		{
			Name: "foo-array",
			Value: v1beta1.ArrayOrString{
				Type:     v1beta1.ParamTypeArray,
				ArrayVal: []string{"$(no-def-val)", "$(no-def-val-1)", "last"},
			},
		},
	}

	str := Param(nil, paramSpec) // No Param are defined for task
	assert.Equal(t, str, "---")

	str = Param(p, paramSpec) // Param has a string value
	assert.Equal(t, str, "foo: bar, foo-1: bar-1")

	str = Param(p1, paramSpec) // Param has array values
	assert.Equal(t, str, "foo: [ v1, v2 ], foo-bar: [ v3, v4, v5 ]")

	str = Param(p2, paramSpec) // Param value is not defined, show default value
	assert.Equal(t, str, "foo: astring, foo-array: [ a1 a2, b1 b2, last ]")

	str = Param(p3, paramSpec)                                              // Param value and default is not defined
	assert.Equal(t, str, "foo: string, foo-array: [ string, array, last ]") // show param type
}
