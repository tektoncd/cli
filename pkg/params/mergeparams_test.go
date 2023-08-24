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
	"reflect"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func Test_MergeParam_String(t *testing.T) {
	params := []v1beta1.Param{
		{
			Name: "key1",
			Value: v1beta1.ParamValue{
				Type:      v1beta1.ParamTypeString,
				StringVal: "value1",
			},
		},
		{
			Name: "key2",
			Value: v1beta1.ParamValue{
				Type:      v1beta1.ParamTypeString,
				StringVal: "value2",
			},
		},
	}

	paramByType["key1"] = v1beta1.ParamTypeString
	paramByType["key2"] = v1beta1.ParamTypeString
	_, err := MergeParam(params, []string{"test"})
	if err == nil {
		t.Errorf("Expected error")
	}
	test.AssertOutput(t, "invalid input format for param parameter: test", err.Error())

	_, err = MergeParam(params, []string{"test=value"})
	if err == nil {
		t.Errorf("Expected error")
	}
	test.AssertOutput(t, "param 'test' not present in spec", err.Error())

	params, err = MergeParam(params, []string{})
	if err != nil {
		t.Errorf("Did not expect error")
	}
	test.AssertOutput(t, 2, len(params))
	test.AssertOutput(t, "value1", params[0].Value.StringVal)
	test.AssertOutput(t, "value2", params[1].Value.StringVal)

	params, err = MergeParam(params, []string{"key1=test"})
	if err != nil {
		t.Errorf("Did not expect error")
	}
	test.AssertOutput(t, 2, len(params))
	test.AssertOutput(t, "test", params[0].Value.StringVal)
	test.AssertOutput(t, "value2", params[1].Value.StringVal)

	params, err = MergeParam(params, []string{"key1=test-new", "key2=test-2"})
	if err != nil {
		t.Errorf("Did not expect error")
	}
	test.AssertOutput(t, 2, len(params))
	test.AssertOutput(t, "test-new", params[0].Value.StringVal)
	test.AssertOutput(t, "test-2", params[1].Value.StringVal)
}

func Test_MergeParam_Array(t *testing.T) {
	params := []v1beta1.Param{
		{
			Name: "key1",
			Value: v1beta1.ParamValue{
				Type:     v1beta1.ParamTypeArray,
				ArrayVal: []string{"value1", "value2"},
			},
		},
	}

	paramByType["key1"] = v1beta1.ParamTypeArray
	_, err := MergeParam(params, []string{"test"})
	if err == nil {
		t.Errorf("Expected error")
	}
	test.AssertOutput(t, "invalid input format for param parameter: test", err.Error())

	_, err = MergeParam(params, []string{"test=value"})
	if err == nil {
		t.Errorf("Expected error")
	}
	test.AssertOutput(t, "param 'test' not present in spec", err.Error())

	params, err = MergeParam(params, []string{})
	if err != nil {
		t.Errorf("Did not expect error")
	}
	test.AssertOutput(t, 1, len(params))
	test.AssertOutput(t, []string{"value1", "value2"}, params[0].Value.ArrayVal)

	params, err = MergeParam(params, []string{"key1=test"})
	if err != nil {
		t.Errorf("Did not expect error")
	}
	test.AssertOutput(t, 1, len(params))
	test.AssertOutput(t, []string{"test"}, params[0].Value.ArrayVal)

	params, err = MergeParam(params, []string{"key1=test-new,test-new-2"})
	if err != nil {
		t.Errorf("Did not expect error")
	}
	test.AssertOutput(t, 1, len(params))
	test.AssertOutput(t, []string{"test-new", "test-new-2"}, params[0].Value.ArrayVal)
}

func Test_parseParam(t *testing.T) {
	type args struct {
		p  []string
		pt []v1beta1.ParamSpec
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]v1beta1.Param
		wantErr bool
	}{{
		name: "Test_parseParam No Err",
		args: args{
			p: []string{"key1=value1", "key2=value2", "key3=value3,value4,value5", "key4=value4", "key5=a:b,c:d"},
			pt: []v1beta1.ParamSpec{
				{
					Name: "key1",
					Type: "string",
				},
				{
					Name: "key2",
					Type: "string",
				},
				{
					Name: "key3",
					Type: "array",
				},
				{
					Name: "key4",
					Type: "array",
				},
				{
					Name: "key5",
					Type: "object",
				},
			},
		},
		want: map[string]v1beta1.Param{
			"key1": {Name: "key1", Value: v1beta1.ParamValue{
				Type:      v1beta1.ParamTypeString,
				StringVal: "value1",
			},
			},
			"key2": {Name: "key2", Value: v1beta1.ParamValue{
				Type:      v1beta1.ParamTypeString,
				StringVal: "value2",
			},
			},
			"key3": {Name: "key3", Value: v1beta1.ParamValue{
				Type:     v1beta1.ParamTypeArray,
				ArrayVal: []string{"value3", "value4", "value5"},
			},
			},
			"key4": {Name: "key4", Value: v1beta1.ParamValue{
				Type:     v1beta1.ParamTypeArray,
				ArrayVal: []string{"value4"},
			},
			},
			"key5": {Name: "key5", Value: v1beta1.ParamValue{
				Type:      v1beta1.ParamTypeObject,
				ObjectVal: map[string]string{"a": "b", "c": "d"},
			},
			},
		},
		wantErr: false,
	}, {
		name: "Test_parseParam with empty array",
		args: args{
			p: []string{"key1=value1", "key2="},
			pt: []v1beta1.ParamSpec{
				{
					Name: "key1",
					Type: "string",
				},
				{
					Name: "key2",
					Type: "array",
				},
			},
		},
		want: map[string]v1beta1.Param{
			"key1": {
				Name: "key1",
				Value: v1beta1.ParamValue{
					Type:      v1beta1.ParamTypeString,
					StringVal: "value1",
				},
			},
			"key2": {
				Name: "key2",
				Value: v1beta1.ParamValue{
					Type:     v1beta1.ParamTypeArray,
					ArrayVal: make([]string, 0),
				},
			},
		},
		wantErr: false,
	}, {
		name: "Test_parseParam Err",
		args: args{
			p: []string{"value1", "value2"},
		},
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			FilterParamsByType(tt.args.pt)
			got, err := parseParam(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseParams() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ParseParams(t *testing.T) {
	t.Run("happy day", func(t *testing.T) {
		pass := []string{"abc=bcd", "one=two"}
		got, err := ParseParams(pass)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("expected two elements, got: %v", got)
		}
		test.AssertOutput(t, "bcd", got["abc"])
		test.AssertOutput(t, "two", got["one"])
	})

	t.Run("missing =", func(t *testing.T) {
		pass := []string{"abc"}
		_, err := ParseParams(pass)
		if err == nil {
			t.Errorf("Expected error")
		}
		test.AssertOutput(t, "invalid input format for param parameter: abc", err.Error())
	})

	t.Run("missing key and value", func(t *testing.T) {
		pass := []string{"="}
		_, err := ParseParams(pass)
		if err == nil {
			t.Errorf("Expected error")
		}
		test.AssertOutput(t, "invalid input format for param parameter: =", err.Error())
	})

	t.Run("missing key", func(t *testing.T) {
		pass := []string{"=val"}
		_, err := ParseParams(pass)
		if err == nil {
			t.Errorf("Expected error")
		}
		test.AssertOutput(t, "invalid input format for param parameter: =val", err.Error())
	})

	t.Run("empty value", func(t *testing.T) {
		got, err := ParseParams([]string{"key="})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		test.AssertOutput(t, "", got["key"])
	})
}
