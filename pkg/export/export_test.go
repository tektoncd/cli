// Copyright © 2022 The Tekton Authors.
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
		p       interface{}
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
					Name:         "hello-moto-1",
					GenerateName: "hello-moto-",
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
  generateName: hello-moto-
spec: {}
`,
		},
		{
			name: "export pipelinerun to yaml",
			p: &v1beta1.PipelineRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "tekton.dev/v1beta1",
					Kind:       "PipelineRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "pipelinerun",
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
kind: PipelineRun
metadata:
  name: pipelinerun
spec: {}
status: {}
`,
		},
		{
			name: "export taskrun to yaml",
			p: &v1beta1.TaskRun{
				Spec: v1beta1.TaskRunSpec{
					ServiceAccountName: "hellomoto",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: "tekton.dev/v1beta1",
					Kind:       "TaskRun",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "taskrun",
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
kind: TaskRun
metadata:
  name: taskrun
spec:
  serviceAccountName: hellomoto
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TektonResourceToYaml(tt.p)
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
