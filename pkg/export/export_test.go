package export

import (
	"testing"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPipelineToYaml(t *testing.T) {
	tests := []struct {
		name    string
		p       *v1beta1.Pipeline
		want    string
		wantErr bool
	}{
		{
			name: "export pipeline to yaml",
			p: &v1beta1.Pipeline{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "tekton.dev/v1beta1",
					Kind:       "Pipeline",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "pipeline",
					CreationTimestamp: metav1.Time{
						Time: time.Now(),
					},
					ManagedFields: []metav1.ManagedFieldsEntry{
						{
							Manager: "tekton.dev",
						},
					},
					ResourceVersion: "1",
					UID:             "2",
					Namespace:       "ironman",
				},
			},
			want: `apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: pipeline
spec: {}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PipelineToYaml(tt.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("PipelineToYaml() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PipelineToYaml() got = %v, want %v", got, tt.want)
			}
		})
	}
}
