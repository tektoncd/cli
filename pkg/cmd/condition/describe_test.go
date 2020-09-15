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

package condition

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConditionDescribe(t *testing.T) {
	conditions := []*v1alpha1.Condition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cond-1",
				Namespace: "ns",
			},
			Spec: v1alpha1.ConditionSpec{
				Check: v1beta1.Step{
					Container: corev1.Container{
						Name:  "test",
						Image: "busybox",
					},
					Script: "echo hello",
				},
				Params: []v1beta1.ParamSpec{
					{
						Name:        "myarg",
						Type:        v1beta1.ParamTypeString,
						Description: "param type is string",
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "default",
						},
					},
					{
						Name:        "print",
						Type:        v1beta1.ParamTypeArray,
						Description: "param type is array",
						Default: &v1beta1.ArrayOrString{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
					{
						Name: "print",
						Type: v1beta1.ParamTypeString,
					},
				},
				Resources: []v1beta1.ResourceDeclaration{
					{
						Name: "my-repo",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
					{
						Name: "my-image",
						Type: v1alpha1.PipelineResourceTypeImage,
					},
					{
						Name: "code-image",
						Type: v1alpha1.PipelineResourceTypeImage,
					},
				},
				Description: "test",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cond-2",
				Namespace: "ns",
			},
			Spec: v1alpha1.ConditionSpec{
				Check: v1beta1.Step{
					Container: corev1.Container{
						Name:  "test",
						Image: "busybox",
					},
					Script: "echo hello",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cond-3",
				Namespace: "ns",
			},
			Spec: v1alpha1.ConditionSpec{
				Check: v1beta1.Step{
					Container: corev1.Container{
						Name:  "test",
						Image: "busybox",
					},
					Script: "echo hello",
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "print",
						Type: v1beta1.ParamTypeString,
					},
				},
				Resources: []v1beta1.ResourceDeclaration{
					{
						Name: "my-repo",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
				},
				Description: "test",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cond-4",
				Namespace: "ns",
			},
			Spec: v1alpha1.ConditionSpec{
				Check: v1beta1.Step{
					Container: corev1.Container{
						Name:    "test",
						Image:   "busybox",
						Args:    []string{"echo", "hello"},
						Command: []string{"/bin/sh"},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name:        "myarg",
						Type:        v1beta1.ParamTypeString,
						Description: "param type is string",
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "default",
						},
					},
					{
						Name:        "print",
						Type:        v1beta1.ParamTypeArray,
						Description: "param type is array",
						Default: &v1beta1.ArrayOrString{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
					{
						Name: "print",
						Type: v1beta1.ParamTypeString,
					},
				},
				Resources: []v1beta1.ResourceDeclaration{
					{
						Name: "my-repo",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
					{
						Name: "my-image",
						Type: v1alpha1.PipelineResourceTypeImage,
					},
					{
						Name: "code-image",
						Type: v1alpha1.PipelineResourceTypeImage,
					},
				},
				Description: "test",
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	s, _ := test.SeedTestData(t, pipelinetest.Data{Conditions: conditions, Namespaces: ns})
	p := &test.Params{Tekton: s.Pipeline, Resource: s.Resource, Kube: s.Kube}
	cmd := Command(p)

	testParams := []struct {
		name      string
		command   []string
		wantError bool
	}{
		{
			name:      "invalid namespace",
			command:   []string{"describe", "cond-2", "-n", "invalid"},
			wantError: true,
		},
		{
			name:      "invalid condition name",
			command:   []string{"describe", "invalid", "-n", "ns"},
			wantError: true,
		},
		{
			name:      "condition describe without resource param",
			command:   []string{"describe", "cond-2", "-n", "ns"},
			wantError: false,
		},
		{
			name:      "condition describe with one resource param",
			command:   []string{"describe", "cond-3", "-n", "ns"},
			wantError: false,
		},
		{
			name:      "condition describe full with script",
			command:   []string{"describe", "cond-1", "-n", "ns"},
			wantError: false,
		},
		{
			name:      "condition describe full with command and arg",
			command:   []string{"describe", "cond-4", "-n", "ns"},
			wantError: false,
		},
		{
			name:      "condition describe full with script with output",
			command:   []string{"describe", "cond-1", "-n", "ns", "-o", "yaml"},
			wantError: false,
		},
		{
			name:      "condition describe full with command and arg with output",
			command:   []string{"describe", "cond-4", "-n", "ns", "-o", "yaml"},
			wantError: false,
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			got, err := test.ExecuteCommand(cmd, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("Error expected here")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected Error")
				}
			}
			golden.Assert(t, got, strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
		})
	}
}
