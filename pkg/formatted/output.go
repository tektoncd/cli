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

	"sigs.k8s.io/yaml"
)

const (
	// OutputJSON is the JSON output format.
	OutputJSON = "json"
	// OutputYAML is the YAML output format.
	OutputYAML = "yaml"
	// OutputFlagUsage is the shared --output flag help text for json/yaml commands.
	OutputFlagUsage = "Output format. One of: json|yaml"
	// CancelledStatus is the user-facing requestedStatus for a cancel operation.
	CancelledStatus = "Cancelled"
)

// CancelItem is one cancelled resource in machine-readable cancel output.
type CancelItem struct {
	Kind            string `json:"kind"`
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	RequestedStatus string `json:"requestedStatus"`
}

// CancelResult is the machine-readable result of a cancel operation.
type CancelResult struct {
	Cancelled []CancelItem `json:"cancelled"`
}

// NewCancelResult builds a CancelResult for a single cancelled resource.
func NewCancelResult(kind, name, namespace, requestedStatus string) CancelResult {
	if requestedStatus == "" {
		requestedStatus = CancelledStatus
	}
	return CancelResult{
		Cancelled: []CancelItem{{
			Kind:            kind,
			Name:            name,
			Namespace:       namespace,
			RequestedStatus: requestedStatus,
		}},
	}
}

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

// PrintStructuredOutput writes obj to w as pretty-printed JSON or YAML.
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
