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
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConditionDescribe(t *testing.T) {
	conditions := []*v1alpha1.Condition{
		tb.Condition("cond-1",
			tb.ConditionNamespace("ns"),
			tb.ConditionSpec(
				tb.ConditionSpecCheck("test", "busybox"),
				tb.ConditionSpecCheckScript("echo hello"),
				tb.ConditionParamSpec("myarg", v1alpha1.ParamTypeString, tb.ParamSpecDescription("param type is string"),
					tb.ParamSpecDefault("default")),
				tb.ConditionParamSpec("print", v1alpha1.ParamTypeArray, tb.ParamSpecDescription("param type is array"),
					tb.ParamSpecDefault("booms", "booms", "booms")),
				tb.ConditionParamSpec("print", v1alpha1.ParamTypeString),
				tb.ConditionResource("my-repo", v1alpha1.PipelineResourceTypeGit),
				tb.ConditionResource("my-image", v1alpha1.PipelineResourceTypeImage),
				tb.ConditionResource("code-image", v1alpha1.PipelineResourceTypeImage),
				tb.ConditionDescription("test"),
			),
		),
		tb.Condition("cond-2",
			tb.ConditionNamespace("ns"),
			tb.ConditionSpec(
				tb.ConditionSpecCheck("test", "busybox"),
				tb.ConditionSpecCheckScript("echo hello"),
			),
		),
		tb.Condition("cond-3",
			tb.ConditionNamespace("ns"),
			tb.ConditionSpec(
				tb.ConditionSpecCheck("test", "busybox"),
				tb.ConditionSpecCheckScript("echo hello"),
				tb.ConditionParamSpec("print", v1alpha1.ParamTypeString),
				tb.ConditionResource("my-repo", v1alpha1.PipelineResourceTypeGit),
				tb.ConditionDescription("test"),
			),
		),
		tb.Condition("cond-4",
			tb.ConditionNamespace("ns"),
			tb.ConditionSpec(
				tb.ConditionSpecCheck("test", "busybox",
					tb.Args("echo", "hello"),
					tb.Command("/bin/sh"),
				),
				tb.ConditionParamSpec("myarg", v1alpha1.ParamTypeString, tb.ParamSpecDescription("param type is string"),
					tb.ParamSpecDefault("default")),
				tb.ConditionParamSpec("print", v1alpha1.ParamTypeArray, tb.ParamSpecDescription("param type is array"),
					tb.ParamSpecDefault("booms", "booms", "booms")),
				tb.ConditionParamSpec("print", v1alpha1.ParamTypeString),
				tb.ConditionResource("my-repo", v1alpha1.PipelineResourceTypeGit),
				tb.ConditionResource("my-image", v1alpha1.PipelineResourceTypeImage),
				tb.ConditionResource("code-image", v1alpha1.PipelineResourceTypeImage),
				tb.ConditionDescription("test"),
			),
		),
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
