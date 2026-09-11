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
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	// OutputJSON is the JSON output format.
	OutputJSON = "json"
	// OutputYAML is the YAML output format.
	OutputYAML = "yaml"
	// OutputFlagUsage is the shared --output flag help text for json/yaml commands.
	OutputFlagUsage = "Output format. One of: json|yaml"
)

// NormalizeOutput lowercases and trims an --output value.
func NormalizeOutput(format string) string {
	return strings.ToLower(strings.TrimSpace(format))
}

// IsStructured reports whether format is json or yaml.
func IsStructured(format string) bool {
	switch NormalizeOutput(format) {
	case OutputJSON, OutputYAML:
		return true
	default:
		return false
	}
}

// SetTypeMeta populates apiVersion and kind on each item from gvk so JSON/YAML
// include the canonical Kubernetes TypeMeta instead of the lowercase resource name.
func SetTypeMeta[T any](items []T, gvk schema.GroupVersionKind) {
	for i := range items {
		tm, ok := any(&items[i]).(interface {
			SetGroupVersionKind(schema.GroupVersionKind)
		})
		if !ok {
			continue
		}
		tm.SetGroupVersionKind(gvk)
	}
}

// Print writes obj to w as pretty-printed JSON or YAML.
func PrintStructuredOutput(w io.Writer, format string, obj interface{}) error {
	switch NormalizeOutput(format) {
	case OutputJSON:
		data, err := json.MarshalIndent(obj, "", "    ")
		if err != nil {
			return err
		}
		data = append(data, '\n')
		_, err = w.Write(data)
		return err
	case OutputYAML:
		data, err := yaml.Marshal(obj)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	default:
		return fmt.Errorf("invalid structured output format %q: must be json or yaml", format)
	}
}
