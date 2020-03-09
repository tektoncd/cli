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
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stest "k8s.io/client-go/testing"
)

func TestConditionList(t *testing.T) {
	clock := clockwork.NewFakeClock()

	seeds := make([]pipelinetest.Clients, 0)

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "empty",
			},
		},
	}

	// Testdata pattern1.
	conditions := []*v1alpha1.Condition{
		tb.Condition("condition1", "ns", cb.ConditionCreationTime(clock.Now().Add(-1*time.Minute))),
		tb.Condition("condition2", "ns", cb.ConditionCreationTime(clock.Now().Add(-20*time.Second))),
		tb.Condition("condition3", "ns", cb.ConditionCreationTime(clock.Now().Add(-512*time.Hour))),
		tb.Condition("condition4", "ns", tb.ConditionSpec(tb.ConditionDescription("a test condition")), cb.ConditionCreationTime(clock.Now().Add(-513*time.Hour))),
		tb.Condition("condition5", "ns", tb.ConditionSpec(tb.ConditionDescription("a test condition to check the trimming is working")), cb.ConditionCreationTime(clock.Now().Add(-514*time.Hour))),
		tb.Condition("condition6", "ns", tb.ConditionSpec(tb.ConditionDescription("")), cb.ConditionCreationTime(clock.Now().Add(-516*time.Hour))),
	}
	s, _ := test.SeedTestData(t, pipelinetest.Data{Conditions: conditions, Namespaces: ns})

	// Testdata pattern2.
	conditions2 := []*v1alpha1.Condition{
		tb.Condition("condition1", "ns", cb.ConditionCreationTime(clock.Now().Add(-1*time.Minute))),
	}
	s2, _ := test.SeedTestData(t, pipelinetest.Data{Conditions: conditions2, Namespaces: ns})
	s2.Pipeline.PrependReactor("list", "conditions", func(action k8stest.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("test error")
	})

	seeds = append(seeds, s)
	seeds = append(seeds, s2)

	testParams := []struct {
		name      string
		command   []string
		input     pipelinetest.Clients
		wantError bool
	}{
		{
			name:      "Invalid Namespace",
			command:   []string{"ls", "-n", "invalid"},
			input:     seeds[0],
			wantError: true,
		},
		{
			name:      "Found no conditions",
			command:   []string{"ls", "-n", "empty"},
			input:     seeds[0],
			wantError: false,
		},
		{
			name:      "Found conditions",
			command:   []string{"ls", "-n", "ns"},
			input:     seeds[0],
			wantError: false,
		},
		{
			name:      "Specify output flag",
			command:   []string{"ls", "-n", "ns", "--output", "yaml"},
			input:     seeds[0],
			wantError: false,
		},
		{
			name:      "Failed to list condition resources",
			command:   []string{"ls", "-n", "ns"},
			input:     seeds[1],
			wantError: true,
		},
		{
			name:      "Failed to list condition resources with specify output flag",
			command:   []string{"ls", "-n", "ns", "--output", "yaml"},
			input:     seeds[1],
			wantError: true,
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube}
			pipelineResource := Command(p)

			out, err := test.ExecuteCommand(pipelineResource, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("Error expected here")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected Error")
				}
			}
			golden.Assert(t, out, strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
		})
	}
}
