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
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// PrintNDJSON serialises each item of a Kubernetes list object as a single
// JSON line (NDJSON / JSON Lines).  When fields is non-empty only those
// dot-separated paths are included in each output object.
// If showManagedFields is false, metadata.managedFields is filtered out before projection.
func PrintNDJSON(w io.Writer, obj runtime.Object, fields []string, showManagedFields bool) error {
	// Deduplicate fields
	fields = deduplicateFields(fields)

	return meta.EachListItem(obj, func(o runtime.Object) error {
		// Filter out managedFields if needed
		if !showManagedFields {
			if accessor, err := meta.Accessor(o); err == nil {
				accessor.SetManagedFields(nil)
			}
		}

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

// PrintJSON serialises a Kubernetes list object as a JSON array.
// When fields is non-empty only those dot-separated paths are included
// in each output object.
// If showManagedFields is false, metadata.managedFields is filtered out before projection.
func PrintJSON(w io.Writer, obj runtime.Object, fields []string, showManagedFields bool) error {
	if len(fields) == 0 {
		// No field projection - marshal the whole object
		data, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal object: %w", err)
		}
		_, err = fmt.Fprintf(w, "%s\n", data)
		return err
	}

	// Deduplicate fields
	fields = deduplicateFields(fields)

	// Apply field projection to each item
	items := []map[string]any{}
	err := meta.EachListItem(obj, func(o runtime.Object) error {
		// Filter out managedFields if needed
		if !showManagedFields {
			if accessor, err := meta.Accessor(o); err == nil {
				accessor.SetManagedFields(nil)
			}
		}

		raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
		if err != nil {
			return fmt.Errorf("failed to convert item to unstructured: %w", err)
		}
		out := pickFields(raw, fields)
		items = append(items, out)
		return nil
	})
	if err != nil {
		return err
	}

	// Marshal the items array
	data, err := json.MarshalIndent(map[string]any{"items": items}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

// fieldResult holds the outcome of a field lookup, distinguishing between
// "path not found" and "path found with a nil value".
type fieldResult struct {
	found bool
	value any
}

// ValidateFields validates the fields list and returns an error if invalid.
// Checks for empty field list and validates field syntax.
func ValidateFields(fields []string) error {
	if len(fields) == 0 {
		return nil
	}

	// Check if all fields are empty strings
	hasNonEmpty := false
	for _, f := range fields {
		trimmed := strings.TrimSpace(f)
		if trimmed != "" {
			hasNonEmpty = true
			// Basic syntax validation: field names should be alphanumeric with dots, brackets, and underscores
			if err := validateFieldSyntax(trimmed); err != nil {
				return fmt.Errorf("invalid field %q: %w", f, err)
			}
		}
	}

	if !hasNonEmpty {
		return fmt.Errorf("at least one field required")
	}

	return nil
}

// validateFieldSyntax checks if a field path has valid syntax.
// Valid: "metadata.name", "status.conditions[0].reason", "spec.params[1].value"
// Invalid: "metadata..name", "status.[0]", "field[abc]", "status.conditions[0]suffix", "status.conditions]"
func validateFieldSyntax(field string) error {
	if field == "" {
		return fmt.Errorf("empty field path")
	}

	// Check for consecutive dots
	if strings.Contains(field, "..") {
		return fmt.Errorf("consecutive dots not allowed")
	}

	// Check for leading/trailing dots
	if strings.HasPrefix(field, ".") || strings.HasSuffix(field, ".") {
		return fmt.Errorf("field path cannot start or end with dot")
	}

	// Validate each segment (split by dots)
	segments := strings.Split(field, ".")
	for _, segment := range segments {
		if segment == "" {
			continue // Already checked for ".." above
		}

		// Check for stray closing bracket without opening bracket
		if strings.Contains(segment, "]") && !strings.Contains(segment, "[") {
			return fmt.Errorf("unmatched closing bracket in %q", segment)
		}

		// If segment contains brackets, validate array indexing syntax
		if strings.Contains(segment, "[") {
			bracketIdx := strings.IndexByte(segment, '[')

			// Must have a non-empty field name before the bracket
			if bracketIdx == 0 {
				return fmt.Errorf("empty field name before array index in %q", segment)
			}

			// Find the closing bracket
			closeIdx := strings.IndexByte(segment, ']')
			if closeIdx == -1 {
				return fmt.Errorf("unclosed bracket in %q", segment)
			}

			// Check that closing bracket comes after opening bracket
			if closeIdx <= bracketIdx {
				return fmt.Errorf("closing bracket before opening bracket in %q", segment)
			}

			// Check what's between brackets
			indexStr := segment[bracketIdx+1 : closeIdx]
			if indexStr == "" {
				return fmt.Errorf("empty array index in %q", segment)
			}

			// Index must be a number
			for _, c := range indexStr {
				if c < '0' || c > '9' {
					return fmt.Errorf("array index must be numeric, got %q in %q", indexStr, segment)
				}
			}

			// The closing bracket must be at the end of the segment (no suffix allowed)
			if closeIdx != len(segment)-1 {
				return fmt.Errorf("unexpected characters after closing bracket in %q", segment)
			}
		}
	}

	return nil
}

// deduplicateFields removes duplicate field names from the list while preserving order.
func deduplicateFields(fields []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if !seen[f] {
			seen[f] = true
			result = append(result, f)
		}
	}

	return result
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
// Supports array indexing with bracket notation: "status.conditions[0].reason"
// Returns a fieldResult with found=false if any segment of the path does not exist.
func getNestedField(src map[string]any, path string) fieldResult {
	parts := strings.SplitN(path, ".", 2)
	currentPart := parts[0]

	// Check for array indexing syntax: fieldName[index]
	var val any
	var ok bool

	if idx := strings.Index(currentPart, "["); idx != -1 {
		// Parse array syntax: "conditions[0]"
		fieldName := currentPart[:idx]
		indexStr := strings.TrimSuffix(currentPart[idx+1:], "]")

		index, err := strconv.Atoi(indexStr)
		if err != nil {
			return fieldResult{found: false}
		}

		val, ok = src[fieldName]
		if !ok {
			return fieldResult{found: false}
		}

		// Handle slice access
		switch v := val.(type) {
		case []any:
			if index < 0 || index >= len(v) {
				return fieldResult{found: false}
			}
			val = v[index]
		case []map[string]any:
			if index < 0 || index >= len(v) {
				return fieldResult{found: false}
			}
			val = v[index]
		default:
			return fieldResult{found: false}
		}
	} else {
		// Regular map access
		val, ok = src[currentPart]
		if !ok {
			return fieldResult{found: false}
		}
	}

	// If this is the last part, return the value
	if len(parts) == 1 {
		return fieldResult{found: true, value: val}
	}

	// val is nil — path exists up to here but cannot descend further
	if val == nil {
		return fieldResult{found: false}
	}

	// Continue traversing nested maps
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
