// Copyright © 2026 The Tekton Authors.
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
	"bytes"
	"encoding/json"
	"testing"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

func TestIsStructured(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   bool
	}{
		{name: "json", format: "json", want: true},
		{name: "uppercase yaml", format: "YAML", want: true},
		{name: "unsupported format", format: "csv", want: false},
		{name: "empty", format: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, IsStructured(tc.format), tc.want)
		})
	}
}

func TestPrintStructuredOutput(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		input   interface{}
		wantErr string
		verify  func(t *testing.T, out []byte)
	}{
		{
			name:   "json array",
			format: "json",
			input: []map[string]string{
				{"name": "foo"},
				{"name": "bar"},
			},
			verify: func(t *testing.T, out []byte) {
				var got []map[string]string
				assert.NilError(t, json.Unmarshal(out, &got))
				assert.DeepEqual(t, []map[string]string{{"name": "foo"}, {"name": "bar"}}, got)
			},
		},
		{
			name:   "yaml array",
			format: "yaml",
			input: []map[string]string{
				{"name": "foo"},
				{"name": "bar"},
			},
			verify: func(t *testing.T, out []byte) {
				var got []map[string]string
				assert.NilError(t, yaml.Unmarshal(out, &got))
				assert.DeepEqual(t, []map[string]string{{"name": "foo"}, {"name": "bar"}}, got)
			},
		},
		{
			name:   "json object",
			format: "json",
			input:  map[string]string{"name": "foo"},
			verify: func(t *testing.T, out []byte) {
				var got map[string]string
				assert.NilError(t, json.Unmarshal(out, &got))
				assert.DeepEqual(t, map[string]string{"name": "foo"}, got)
			},
		},
		{
			name:    "invalid format",
			format:  "csv",
			input:   map[string]string{"name": "foo"},
			wantErr: `invalid structured output format "csv"`,
		},
		{
			name:   "empty slice is json array",
			format: "json",
			input:  []string{},
			verify: func(t *testing.T, out []byte) {
				var got []string
				assert.NilError(t, json.Unmarshal(out, &got))
				assert.Equal(t, len(got), 0)
			},
		},
		{
			name:   "empty slice is yaml sequence",
			format: "yaml",
			input:  []string{},
			verify: func(t *testing.T, out []byte) {
				var got []string
				assert.NilError(t, yaml.Unmarshal(out, &got))
				assert.Equal(t, len(got), 0)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := PrintStructuredOutput(&buf, tc.format, tc.input)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("expected error for invalid output format")
				}
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}

			assert.NilError(t, err)
			tc.verify(t, buf.Bytes())
		})
	}
}

func TestNormalizeOutput(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   string
	}{
		{name: "trims and lowercases json", format: " JSON ", want: "json"},
		{name: "lowercases yaml", format: "YAML", want: "yaml"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, NormalizeOutput(tc.format), tc.want)
		})
	}
}

func TestSetTypeMeta(t *testing.T) {
	type obj struct {
		metav1.TypeMeta
		Name string
	}
	items := []obj{{Name: "foo"}}
	SetTypeMeta(items, schema.GroupVersionKind{Group: "tekton.dev", Version: "v1", Kind: "Pipeline"})
	assert.Equal(t, items[0].Kind, "Pipeline")
	assert.Equal(t, items[0].APIVersion, "tekton.dev/v1")
}
