package export

import (
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRemoveFieldForExport(t *testing.T) {
	tests := []struct {
		name     string
		input    *unstructured.Unstructured
		expected map[string]interface{}
	}{
		{
			name: "Remove status and metadata fields",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": "some-status",
					"metadata": map[string]interface{}{
						"managedFields":     "some-managed-fields",
						"resourceVersion":   "some-resource-version",
						"uid":               "some-uid",
						"finalizers":        "some-finalizers",
						"generation":        "some-generation",
						"namespace":         "some-namespace",
						"creationTimestamp": "some-timestamp",
						"ownerReferences":   "some-owner-references",
						"annotations": map[string]interface{}{
							"kubectl.kubernetes.io/last-applied-configuration": "some-configuration",
						},
					},
					"spec": map[string]interface{}{
						"status":        "some-spec-status",
						"statusMessage": "some-status-message",
					},
				},
			},
			expected: map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{},
				},
				"spec": map[string]interface{}{},
			},
		},
		{
			name: "Remove name if generateName exists",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"generateName": "some-generate-name",
						"name":         "some-name",
					},
				},
			},
			expected: map[string]interface{}{
				"metadata": map[string]interface{}{
					"generateName": "some-generate-name",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RemoveFieldForExport(tt.input)
			assert.NilError(t, err)
			assert.DeepEqual(t, tt.expected, tt.input.Object)
		})
	}
}
