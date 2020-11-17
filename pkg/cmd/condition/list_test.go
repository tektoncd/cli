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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ns-1",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ns-2",
			},
		},
	}

	// Testdata pattern1.
	conditions := []*v1alpha1.Condition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "condition1",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "condition2",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-20 * time.Second)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "condition3",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-512 * time.Hour)},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "condition4",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-513 * time.Hour)},
			},
			Spec: v1alpha1.ConditionSpec{
				Description: "a test condition",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "condition5",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-514 * time.Hour)},
			},
			Spec: v1alpha1.ConditionSpec{
				Description: "a test condition to check the trimming is working",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "condition6",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-516 * time.Hour)},
			},
			Spec: v1alpha1.ConditionSpec{
				Description: "",
			},
		},
	}
	s, _ := test.SeedTestData(t, pipelinetest.Data{Conditions: conditions, Namespaces: ns})

	// Testdata pattern2.
	conditions2 := []*v1alpha1.Condition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "condition1",
				Namespace:         "ns",
				CreationTimestamp: metav1.Time{Time: clock.Now().Add(-1 * time.Minute)},
			},
		},
	}
	s2, _ := test.SeedTestData(t, pipelinetest.Data{Conditions: conditions2, Namespaces: ns})
	s2.Pipeline.PrependReactor("list", "conditions", func(action k8stest.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("test error")
	})

	// Testdata patterne
	conditions3 := []*v1alpha1.Condition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-condition-1",
				Namespace:         "test-ns-1",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-condition-2",
				Namespace:         "test-ns-2",
				CreationTimestamp: metav1.Time{Time: clock.Now()},
			},
		},
	}
	s3, _ := test.SeedTestData(t, pipelinetest.Data{Conditions: conditions3, Namespaces: ns})

	// Testdata pattern4.
	conditions4 := []*v1alpha1.Condition{}
	s4, _ := test.SeedTestData(t, pipelinetest.Data{Conditions: conditions4, Namespaces: ns})

	seeds = append(seeds, s)
	seeds = append(seeds, s2)
	seeds = append(seeds, s3)
	seeds = append(seeds, s4)

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
			wantError: false,
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
		{
			name:      "Conditions from all namespaces",
			command:   []string{"list", "--all-namespaces"},
			input:     seeds[2],
			wantError: false,
		},
		{
			name:      "No conditions found in all namespaces",
			command:   []string{"list", "--all-namespaces"},
			input:     seeds[3],
			wantError: false,
		},
		{
			name:      "List conditions without headers",
			command:   []string{"list", "--no-headers"},
			input:     seeds[0],
			wantError: false,
		},
		{
			name:      "List conditions from all namespaces without headers",
			command:   []string{"list", "--no-headers", "--all-namespaces"},
			input:     seeds[2],
			wantError: false,
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
