// Copyright © 2019 The Tekton Authors.
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
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConditionDelete(t *testing.T) {
	clock := clockwork.NewFakeClock()

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seeds := make([]pipelinetest.Clients, 0)

	for i := 0; i < 5; i++ {
		conditions := []*v1alpha1.Condition{
			tb.Condition("condition1",
				tb.ConditionNamespace("ns"),
				cb.ConditionCreationTime(clock.Now().Add(-1*time.Minute)),
			),
			tb.Condition("condition2",
				tb.ConditionNamespace("ns"),
				cb.ConditionCreationTime(clock.Now().Add(-1*time.Minute)),
			),
			tb.Condition("condition3",
				tb.ConditionNamespace("ns"),
				cb.ConditionCreationTime(clock.Now().Add(-1*time.Minute)),
			),
		}
		s, _ := test.SeedTestData(t, pipelinetest.Data{Conditions: conditions, Namespaces: ns})
		seeds = append(seeds, s)
	}

	testParams := []struct {
		name        string
		command     []string
		input       pipelinetest.Clients
		inputStream io.Reader
		wantError   bool
		want        string
	}{
		{
			name:        "Invalid namespace",
			command:     []string{"rm", "condition1", "-n", "invalid"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "namespaces \"invalid\" not found",
		},
		{
			name:        "With force delete flag (shorthand)",
			command:     []string{"rm", "condition1", "-n", "ns", "-f"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "Conditions deleted: \"condition1\"\n",
		},
		{
			name:        "With force delete flag (shorthand), multiple conditions",
			command:     []string{"rm", "condition2", "condition3", "-n", "ns", "-f"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "Conditions deleted: \"condition2\", \"condition3\"\n",
		},
		{
			name:        "With force delete flag",
			command:     []string{"rm", "condition1", "-n", "ns", "--force"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   false,
			want:        "Conditions deleted: \"condition1\"\n",
		},
		{
			name:        "Without force delete flag, reply no",
			command:     []string{"rm", "condition1", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("n"),
			wantError:   true,
			want:        "canceled deleting condition(s) \"condition1\"",
		},
		{
			name:        "Without force delete flag, reply yes",
			command:     []string{"rm", "condition1", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete condition(s) \"condition1\" (y/n): Conditions deleted: \"condition1\"\n",
		},
		{
			name:        "Without force delete flag, reply yes, multiple conditions",
			command:     []string{"rm", "condition2", "condition3", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete condition(s) \"condition2\", \"condition3\" (y/n): Conditions deleted: \"condition2\", \"condition3\"\n",
		},
		{
			name:        "Remove non existent resource",
			command:     []string{"rm", "nonexistent", "-n", "ns"},
			input:       seeds[2],
			inputStream: strings.NewReader("y"),
			wantError:   true,
			want:        "failed to delete Condition \"nonexistent\": conditions.tekton.dev \"nonexistent\" not found",
		},
		{
			name:        "Remove multiple non existent resources",
			command:     []string{"rm", "nonexistent", "nonexistent2", "-n", "ns"},
			input:       seeds[2],
			inputStream: nil,
			wantError:   true,
			want:        "failed to delete Condition \"nonexistent\": conditions.tekton.dev \"nonexistent\" not found; failed to delete Condition \"nonexistent2\": conditions.tekton.dev \"nonexistent2\" not found",
		},
		{
			name:        "Delete all with prompt",
			command:     []string{"delete", "--all", "-n", "ns"},
			input:       seeds[3],
			inputStream: strings.NewReader("y"),
			wantError:   false,
			want:        "Are you sure you want to delete all conditions in namespace \"ns\" (y/n): All Conditions deleted in namespace \"ns\"\n",
		},
		{
			name:        "Delete all with -f",
			command:     []string{"delete", "--all", "-f", "-n", "ns"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   false,
			want:        "All Conditions deleted in namespace \"ns\"\n",
		},
		{
			name:        "Error from using condition name with --all",
			command:     []string{"delete", "cond", "--all", "-n", "ns"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   true,
			want:        "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:        "Error from using condition delete with no names or --all",
			command:     []string{"delete"},
			input:       seeds[4],
			inputStream: nil,
			wantError:   true,
			want:        "must provide condition name(s) or use --all flag with delete",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube}
			condition := Command(p)

			if tp.inputStream != nil {
				condition.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(condition, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}
