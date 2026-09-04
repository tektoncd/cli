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

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// PrintNDJSON serialises each item of a Kubernetes list object as a single
// JSON line (NDJSON / JSON Lines).  When fields is non-empty only those
// dot-separated paths are included in each output object.
func PrintNDJSON(w io.Writer, obj runtime.Object, fields []string) error {
	return meta.EachListItem(obj, func(o runtime.Object) error {
		raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
		if err != nil {
			return fmt.Errorf("failed to convert item to unstructured: %w", err)
		}
		out := map[string]any(raw)
		if len(fields) > 0 {
			out = pickFields(raw, fields)
		}
		line, err := json.Marshal(out)
		if err != nil {
			return fmt.Errorf("failed to marshal item: %w", err)
		}
		_, err = fmt.Fprintf(w, "%s\n", line)
		return err
	})
}

// fieldResult holds the outcome of a field lookup, distinguishing between
// "path not found" and "path found with a nil value".
type fieldResult struct {
	found bool
	value any
}

// pickFields returns a new map containing only the requested dot-path fields.
// Each field is a dot-separated path such as "metadata.name" or "status.startTime".
// Multiple fields that share a common prefix are merged into the same nested map.
// Fields whose path is not found in src are omitted; fields found with a nil value
// are emitted as null so consumers can distinguish "absent" from "null".
func pickFields(src map[string]any, fields []string) map[string]any {
	dst := map[string]any{}
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		res := getNestedField(src, f)
		if res.found {
			setNestedField(dst, res, f)
		}
	}
	return dst
}

// getNestedField retrieves a value from a nested map using a dot-separated path.
// Returns a fieldResult with found=false if any segment of the path does not exist.
func getNestedField(src map[string]any, path string) fieldResult {
	parts := strings.SplitN(path, ".", 2)
	val, ok := src[parts[0]]
	if !ok {
		return fieldResult{found: false}
	}
	if len(parts) == 1 {
		return fieldResult{found: true, value: val}
	}
	// val is nil — path exists up to here but cannot descend further.
	if val == nil {
		return fieldResult{found: false}
	}
	child, ok := val.(map[string]any)
	if !ok {
		return fieldResult{found: false}
	}
	return getNestedField(child, parts[1])
}

// setNestedField sets a value in dst at the given dot-separated path,
// creating intermediate maps as needed and merging with existing maps.
// It always writes the value (even nil) so that null fields are preserved.
func setNestedField(dst map[string]any, res fieldResult, path string) {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) == 1 {
		dst[parts[0]] = res.value
		return
	}
	// Ensure the intermediate map exists.
	child, ok := dst[parts[0]].(map[string]any)
	if !ok {
		child = map[string]any{}
		dst[parts[0]] = child
	}
	setNestedField(child, res, parts[1])
}
