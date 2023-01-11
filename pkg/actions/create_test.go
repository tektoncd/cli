// Copyright © 2020 The Tekton Authors.
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

package actions

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/test/diff"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var tektonGroup = "tekton.dev"

func Test_getGVRWithObject(t *testing.T) {
	type args struct {
		object *unstructured.Unstructured
		gvr    schema.GroupVersionResource
	}
	tests := []struct {
		name string
		args args
		want *schema.GroupVersionResource
	}{
		{
			name: "v1 task",
			args: args{
				object: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "tekton.dev/v1",
					},
				},
				gvr: schema.GroupVersionResource{
					Group:    tektonGroup,
					Resource: "tasks",
				},
			},
			want: &schema.GroupVersionResource{
				Group:    tektonGroup,
				Resource: "tasks",
				Version:  "v1",
			},
		},
		{
			name: "v1beta1 task",
			args: args{
				object: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "tekton.dev/v1beta1",
					},
				},
				gvr: schema.GroupVersionResource{
					Group:    tektonGroup,
					Resource: "tasks",
				},
			},
			want: &schema.GroupVersionResource{
				Group:    tektonGroup,
				Resource: "tasks",
				Version:  "v1beta1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getGVRWithObject(tt.args.object, tt.args.gvr)
			if err != nil {
				t.Errorf("getGVRWithObject() got error: %v", err)
				return
			}
			if d := cmp.Diff(tt.want, got); d != "" {
				t.Errorf("ApplyParameters() got diff %s", diff.PrintWantGot(d))
			}
		})
	}
}
