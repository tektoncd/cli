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
	assert.Equal(t, IsStructured("json"), true)
	assert.Equal(t, IsStructured("YAML"), true)
	assert.Equal(t, IsStructured("csv"), false)
	assert.Equal(t, IsStructured(""), false)
}

func TestPrint_JSONArray(t *testing.T) {
	var buf bytes.Buffer
	items := []map[string]string{
		{"name": "foo"},
		{"name": "bar"},
	}
	assert.NilError(t, PrintStructuredOutput(&buf, "json", items))

	var got []map[string]string
	assert.NilError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.DeepEqual(t, items, got)
}

func TestPrint_YAMLArray(t *testing.T) {
	var buf bytes.Buffer
	items := []map[string]string{
		{"name": "foo"},
		{"name": "bar"},
	}
	assert.NilError(t, PrintStructuredOutput(&buf, "yaml", items))

	var got []map[string]string
	assert.NilError(t, yaml.Unmarshal(buf.Bytes(), &got))
	assert.DeepEqual(t, items, got)
}

func TestPrint_JSONObject(t *testing.T) {
	var buf bytes.Buffer
	obj := map[string]string{"name": "foo"}
	assert.NilError(t, PrintStructuredOutput(&buf, "json", obj))

	var got map[string]string
	assert.NilError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.DeepEqual(t, obj, got)
}

func TestPrint_InvalidFormat(t *testing.T) {
	var buf bytes.Buffer
	err := PrintStructuredOutput(&buf, "csv", map[string]string{"name": "foo"})
	if err == nil {
		t.Fatal("expected error for invalid output format")
	}
	assert.ErrorContains(t, err, "invalid structured output format \"csv\"")
}

func TestPrint_EmptySliceIsJSONArray(t *testing.T) {
	var buf bytes.Buffer
	assert.NilError(t, PrintStructuredOutput(&buf, "json", []string{}))

	var got []string
	assert.NilError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, len(got), 0)
}

func TestPrint_EmptySliceIsYAMLSequence(t *testing.T) {
	var buf bytes.Buffer
	assert.NilError(t, PrintStructuredOutput(&buf, "yaml", []string{}))

	var got []string
	assert.NilError(t, yaml.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, len(got), 0)
}

func TestNormalizeOutput(t *testing.T) {
	assert.Equal(t, NormalizeOutput(" JSON "), "json")
	assert.Equal(t, NormalizeOutput("YAML"), "yaml")
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
