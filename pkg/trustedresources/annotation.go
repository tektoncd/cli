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

package trustedresources

import (
	"fmt"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// insertAnnotation sets key: value in metadata.annotations of doc, replacing an
// existing entry for key, and returns the result, leaving every other byte of
// the document untouched.
//
// The document is parsed only to locate the insertion point. Re-serializing the
// parsed tree would be simpler, but it does not round-trip: a folded scalar
// (`description: >-`) comes back with its line breaks collapsed, which is
// exactly the kind of unrelated diff this is meant to avoid.
//
// An error is returned when the document is not a mapping with a metadata
// mapping in it, when either mapping is written in flow style, where there
// is no line to insert, or when an existing entry for key spans several lines.
func insertAnnotation(doc []byte, key, value string) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(doc, &root); err != nil {
		return nil, fmt.Errorf("error parsing document: %w", err)
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil, fmt.Errorf("document is empty")
	}

	top := root.Content[0]
	if top.Kind != yaml.MappingNode || top.Style == yaml.FlowStyle {
		return nil, fmt.Errorf("document is not a block mapping")
	}

	metadata := mappingValue(top, "metadata")
	if metadata == nil {
		return nil, fmt.Errorf("document has no metadata")
	}
	if metadata.Kind != yaml.MappingNode || metadata.Style == yaml.FlowStyle || len(metadata.Content) == 0 {
		return nil, fmt.Errorf("metadata is not a non-empty block mapping")
	}

	encoded, err := encodeScalar(value)
	if err != nil {
		return nil, err
	}

	annotationsKey, annotations := mappingEntry(metadata, "annotations")

	switch {
	// An existing block mapping: write the entry above the first one it has.
	// Content alternates key, value, so Content[0] is that first key and its
	// position is where a new line belongs.
	case annotations != nil && annotations.Kind == yaml.MappingNode &&
		annotations.Style != yaml.FlowStyle && len(annotations.Content) > 0:
		// Re-signing: overwrite the old entry instead of adding a duplicate key.
		if oldKey, oldValue := mappingEntry(annotations, key); oldKey != nil {
			if !onOneLine(doc, oldKey, oldValue) {
				return nil, fmt.Errorf("existing %s annotation spans several lines", key)
			}
			return replaceLine(doc, oldKey.Line, indentOf(oldKey)+key+": "+encoded)
		}
		first := annotations.Content[0]
		return insertLine(doc, first.Line, indentOf(first)+key+": "+encoded)

	// `annotations: {}` or `annotations:` with nothing under it. There is no
	// entry to anchor to, so the key's own line is rewritten.
	case annotations != nil && isEmptyMapping(annotations):
		indent := indentOf(annotationsKey)
		return replaceLine(doc, annotationsKey.Line, indent+"annotations:\n"+indent+"  "+key+": "+encoded)

	case annotations != nil:
		return nil, fmt.Errorf("annotations is not a block mapping")

	// No annotations at all: open one above the first metadata entry.
	default:
		first := metadata.Content[0]
		indent := indentOf(first)
		return insertLine(doc, first.Line, indent+"annotations:\n"+indent+"  "+key+": "+encoded)
	}
}

// mappingValue returns the value node for key in a mapping node, or nil.
func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	_, value := mappingEntry(mapping, key)
	return value
}

// mappingEntry returns the key and value nodes for key in a mapping node. Both
// are nil when the mapping does not have that key.
func mappingEntry(mapping *yaml.Node, key string) (*yaml.Node, *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i], mapping.Content[i+1]
		}
	}
	return nil, nil
}

// onOneLine reports whether the key: value entry is written entirely on the
// key's line, so that rewriting that line rewrites the whole entry.
func onOneLine(doc []byte, key, value *yaml.Node) bool {
	if value.Kind != yaml.ScalarNode || value.Line != key.Line {
		return false
	}
	lines := strings.SplitAfter(string(doc), "\n")
	var entry map[string]string
	if err := yaml.Unmarshal([]byte(lines[key.Line-1]), &entry); err != nil {
		return false
	}
	return entry[key.Value] == value.Value
}

// isEmptyMapping reports whether n holds no entries, covering both `{}` and a
// key written with nothing under it, which parses as null.
func isEmptyMapping(n *yaml.Node) bool {
	if n.Kind == yaml.MappingNode && len(n.Content) == 0 {
		return true
	}
	return n.Kind == yaml.ScalarNode && n.Tag == "!!null"
}

// indentOf returns the leading whitespace that puts a new line at the same
// depth as n. Column is 1 based.
func indentOf(n *yaml.Node) string {
	return strings.Repeat(" ", n.Column-1)
}

// encodeScalar renders value the way YAML expects it, quoting it if needed.
func encodeScalar(value string) (string, error) {
	out, err := yaml.Marshal(value)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// insertLine returns doc with text inserted as its own line, immediately before
// the 1 based line number.
func insertLine(doc []byte, line int, text string) ([]byte, error) {
	lines := strings.SplitAfter(string(doc), "\n")
	if line < 1 || line > len(lines) {
		return nil, fmt.Errorf("line %d is outside the document", line)
	}
	at := line - 1
	eol := lineEnding(lines[at])

	var b strings.Builder
	b.Grow(len(doc) + len(text) + 1)
	for _, l := range lines[:at] {
		b.WriteString(l)
	}
	b.WriteString(strings.ReplaceAll(text, "\n", eol))
	b.WriteString(eol)
	for _, l := range lines[at:] {
		b.WriteString(l)
	}

	return []byte(b.String()), nil
}

// replaceLine returns doc with the 1 based line number replaced by text.
func replaceLine(doc []byte, line int, text string) ([]byte, error) {
	lines := strings.SplitAfter(string(doc), "\n")
	if line < 1 || line > len(lines) {
		return nil, fmt.Errorf("line %d is outside the document", line)
	}
	at := line - 1
	eol := lineEnding(lines[at])

	var b strings.Builder
	b.Grow(len(doc) + len(text))
	for _, l := range lines[:at] {
		b.WriteString(l)
	}
	b.WriteString(strings.ReplaceAll(text, "\n", eol))
	b.WriteString(eol)
	for _, l := range lines[at+1:] {
		b.WriteString(l)
	}

	return []byte(b.String()), nil
}

// lineEnding returns the terminator of l, so that written lines match a
// document that uses CRLF.
func lineEnding(l string) string {
	if strings.HasSuffix(l, "\r\n") {
		return "\r\n"
	}
	return "\n"
}
