// Copyright © 2024 The Tekton Authors.
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

	"k8s.io/apimachinery/pkg/runtime"
)

// PrintNDJSON serialises each item of a Kubernetes list object as a single
// JSON line (NDJSON / JSON Lines).  When fields is non-empty only those
// dot-separated paths are included in each output object.
func PrintNDJSON(w io.Writer, obj runtime.Object, fields []string) error {
	// Convert the list to unstructured so we can work with raw map[string]any.
	raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return fmt.Errorf("failed to convert object to unstructured: %w", err)
	}

	itemsVal, ok := raw["items"]
	if !ok || itemsVal == nil {
		// A missing or nil "items" field means an empty list — nothing to emit.
		return nil
	}
	items, ok := itemsVal.([]any)
	if !ok {
		return fmt.Errorf("\"items\" field is not a slice (got %T): not a list type", itemsVal)
	}

	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out := m
		if len(fields) > 0 {
			out = pickFields(m, fields)
		}
		line, err := json.Marshal(out)
		if err != nil {
			return fmt.Errorf("failed to marshal item: %w", err)
		}
		if _, err := fmt.Fprintf(w, "%s\n", line); err != nil {
			return err
		}
	}
	return nil
}

// pickFields returns a new map containing only the requested dot-path fields.
// Each field is a dot-separated path such as "metadata.name" or "status.startTime".
// Multiple fields that share a common prefix are merged into the same nested map.
func pickFields(src map[string]any, fields []string) map[string]any {
	dst := map[string]any{}
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		setNestedField(dst, getNestedField(src, f), f)
	}
	return dst
}

// getNestedField retrieves a value from a nested map using a dot-separated path.
// Returns nil if the path does not exist.
func getNestedField(src map[string]any, path string) any {
	parts := strings.SplitN(path, ".", 2)
	val, ok := src[parts[0]]
	if !ok {
		return nil
	}
	if len(parts) == 1 {
		return val
	}
	child, ok := val.(map[string]any)
	if !ok {
		return nil
	}
	return getNestedField(child, parts[1])
}

// setNestedField sets a value in dst at the given dot-separated path,
// creating intermediate maps as needed and merging with existing maps.
func setNestedField(dst map[string]any, val any, path string) {
	if val == nil {
		return
	}
	parts := strings.SplitN(path, ".", 2)
	if len(parts) == 1 {
		dst[parts[0]] = val
		return
	}
	// Ensure the intermediate map exists.
	child, ok := dst[parts[0]].(map[string]any)
	if !ok {
		child = map[string]any{}
		dst[parts[0]] = child
	}
	setNestedField(child, val, parts[1])
}
