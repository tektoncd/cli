// Copyright © 2023 The Tekton Authors.
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

package customrun

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

const (
	versionv1beta1 = "v1beta1"
)

func TestListCustomRuns_v1beta1(t *testing.T) {
	now := time.Now()
	aMinute, _ := time.ParseDuration("1m")
	twoMinute, _ := time.ParseDuration("2m")
	crs := []*v1beta1.CustomRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "cr0-1",
				Labels:    map[string]string{"tekton.dev/customTask": "random"},
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "cr1-1",
				Labels:    map[string]string{"tekton.dev/customTask": "bar"},
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
				CustomRunStatusFields: v1beta1.CustomRunStatusFields{
					StartTime:      &metav1.Time{Time: now},
					CompletionTime: &metav1.Time{Time: now.Add(aMinute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "cr2-1",
				Labels:    map[string]string{"tekton.dev/customTask": "random"},
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: v1beta1.CustomRunReasonRunning.String(),
						},
					},
				},
				CustomRunStatusFields: v1beta1.CustomRunStatusFields{
					StartTime: &metav1.Time{Time: now},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "cr2-2",
				Labels:    map[string]string{"tekton.dev/customTask": "random", "pot": "nutella"},
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.CustomRunReasonFailed.String(),
						},
					},
				},
				CustomRunStatusFields: v1beta1.CustomRunStatusFields{
					StartTime:      &metav1.Time{Time: now.Add(aMinute)},
					CompletionTime: &metav1.Time{Time: now.Add(twoMinute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "cr3-1",
				Labels:    map[string]string{"tekton.dev/customTask": "random", "pot": "honey"},
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.CustomRunReasonFailed.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "cr4-1",
				Labels:    map[string]string{"tekton.dev/customTask": "bar", "pot": "honey"},
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.CustomRunReasonFailed.String(),
						},
					},
				},
			},
		},
	}

	crsMultipleNs := []*v1beta1.CustomRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "tout",
				Name:      "tr4-1",
				Labels:    map[string]string{"tekton.dev/customTask": "random"},
			},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "lacher",
				Name:      "tr4-2",
				Labels:    map[string]string{"tekton.dev/customTask": "random"},
			},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "lacher",
				Name:      "tr5-1",
				Labels:    map[string]string{"tekton.dev/customTask": "random"},
			},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.TaskRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random",
			},
		},
	}
	tdc1 := testDynamic.Options{}
	dc1, err := tdc1.Client(
		cb.UnstructuredV1beta1CustomRun(crs[0], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[1], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[2], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[3], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[4], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[5], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1CustomRun(crsMultipleNs[0], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crsMultipleNs[1], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crsMultipleNs[2], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	tests := []struct {
		name      string
		command   *cobra.Command
		args      []string
		wantError bool
	}{
		{
			name:      "by output as name",
			command:   commandV1beta1(t, crs, now, ns, dc1),
			args:      []string{"list", "-n", "foo", "-o", "name"},
			wantError: false,
		},
		{
			name:      "all in namespace",
			command:   commandV1beta1(t, crs, now, ns, dc1),
			args:      []string{"list", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "print by template",
			command:   commandV1beta1(t, crs, now, ns, dc1),
			args:      []string{"list", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
		{
			name:    "empty list",
			command: commandV1beta1(t, crs, now, ns, dc1),
			args:    []string{"list", "-n", "random"},
		},
		{
			name:      "limit customruns returned to 1",
			command:   commandV1beta1(t, crs, now, ns, dc1),
			args:      []string{"list", "-n", "foo", "--limit", fmt.Sprintf("%d", 1)},
			wantError: false,
		},
		{
			name:      "limit customruns negative case",
			command:   commandV1beta1(t, crs, now, ns, dc1),
			args:      []string{"list", "-n", "foo", "--limit", fmt.Sprintf("%d", -1)},
			wantError: true,
		},
		{
			name:      "limit customruns greater than maximum case",
			command:   commandV1beta1(t, crs, now, ns, dc1),
			args:      []string{"list", "-n", "foo", "--limit", fmt.Sprintf("%d", 7)},
			wantError: false,
		},
		{
			name:      "limit customruns with output flag set",
			command:   commandV1beta1(t, crs, now, ns, dc1),
			args:      []string{"list", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}", "--limit", fmt.Sprintf("%d", 2)},
			wantError: false,
		},
		{
			name:      "error from invalid namespace",
			command:   commandV1beta1(t, crs, now, ns, dc1),
			args:      []string{"list", "-n", "invalid"},
			wantError: true,
		},
		{
			name:      "filter customruns by label",
			command:   commandV1beta1(t, crs, now, ns, dc1),
			args:      []string{"list", "-n", "foo", "--label", "pot=honey"},
			wantError: false,
		},
		{
			name:      "filter customruns by label with in query",
			command:   commandV1beta1(t, crs, now, ns, dc1),
			args:      []string{"list", "-n", "foo", "--label", "pot in (honey,nutella)"},
			wantError: false,
		},
		{
			name:      "print in reverse",
			command:   commandV1beta1(t, crs, now, ns, dc1),
			args:      []string{"list", "--reverse", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "print in reverse with output flag",
			command:   commandV1beta1(t, crs, now, ns, dc1),
			args:      []string{"list", "--reverse", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
		{
			name:      "print customruns in all namespaces",
			command:   commandV1beta1(t, crsMultipleNs, now, ns, dc2),
			args:      []string{"list", "--all-namespaces"},
			wantError: false,
		},
		{
			name:      "print customruns without headers",
			command:   commandV1beta1(t, crsMultipleNs, now, ns, dc2),
			args:      []string{"list", "--no-headers", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "print customruns in all namespaces without headers",
			command:   commandV1beta1(t, crsMultipleNs, now, ns, dc2),
			args:      []string{"list", "--all-namespaces", "--no-headers"},
			wantError: false,
		},
	}

	for _, td := range tests {
		t.Run(td.name, func(t *testing.T) {
			got, err := test.ExecuteCommand(td.command, td.args...)

			if err != nil && !td.wantError {
				t.Errorf("Unexpected error: %v", err)
			}
			golden.Assert(t, got, strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
		})
	}
}

func commandV1beta1(t *testing.T, crs []*v1beta1.CustomRun, now time.Time, ns []*corev1.Namespace, dc dynamic.Interface) *cobra.Command {
	// fake clock advanced by 1 hour
	clock := clockwork.NewFakeClockAt(now)
	clock.Advance(time.Duration(60) * time.Minute)
	cs, _ := test.SeedTestData(t, pipelinetest.Data{CustomRuns: crs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"customrun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}

	return Command(p)
}
