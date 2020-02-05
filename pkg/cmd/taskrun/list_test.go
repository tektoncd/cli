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

package taskrun

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinetest "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestListTaskRuns(t *testing.T) {
	now := time.Now()
	aMinute, _ := time.ParseDuration("1m")
	twoMinute, _ := time.ParseDuration("2m")

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr0-1", "foo",
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
		tb.TaskRun("tr1-1", "foo",
			tb.TaskRunLabel("tekton.dev/task", "bar"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("bar")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.TaskRunStartTime(now),
				taskRunCompletionTime(now.Add(aMinute)),
			),
		),
		tb.TaskRun("tr2-1", "foo",
			tb.TaskRunLabel("tekton.dev/Task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionUnknown,
					Reason: resources.ReasonRunning,
				}),
				tb.TaskRunStartTime(now),
			),
		),
		tb.TaskRun("tr2-2", "foo",
			tb.TaskRunLabel("tekton.dev/Task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionFalse,
					Reason: resources.ReasonFailed,
				}),
				tb.TaskRunStartTime(now.Add(aMinute)),
				taskRunCompletionTime(now.Add(twoMinute)),
			),
		),
		tb.TaskRun("tr3-1", "foo",
			tb.TaskRunLabel("tekton.dev/Task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random")),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionFalse,
					Reason: resources.ReasonFailed,
				}),
			),
		),
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

	tests := []struct {
		name      string
		command   *cobra.Command
		args      []string
		wantError bool
	}{
		{
			name:      "by Task name",
			command:   command(t, trs, now, ns),
			args:      []string{"list", "bar", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "by output as name",
			command:   command(t, trs, now, ns),
			args:      []string{"list", "-n", "foo", "-o", "name"},
			wantError: false,
		},
		{
			name:      "all in namespace",
			command:   command(t, trs, now, ns),
			args:      []string{"list", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "print by template",
			command:   command(t, trs, now, ns),
			args:      []string{"list", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
		{
			name:    "empty list",
			command: command(t, trs, now, ns),
			args:    []string{"list", "-n", "random"},
		},
		{
			name:      "limit taskruns returned to 1",
			command:   command(t, trs, now, ns),
			args:      []string{"list", "-n", "foo", "--limit", fmt.Sprintf("%d", 1)},
			wantError: false,
		},
		{
			name:      "limit taskruns negative case",
			command:   command(t, trs, now, ns),
			args:      []string{"list", "-n", "foo", "--limit", fmt.Sprintf("%d", -1)},
			wantError: false,
		},
		{
			name:      "limit taskruns greater than maximum case",
			command:   command(t, trs, now, ns),
			args:      []string{"list", "-n", "foo", "--limit", fmt.Sprintf("%d", 7)},
			wantError: false,
		},
		{
			name:      "limit taskruns with output flag set",
			command:   command(t, trs, now, ns),
			args:      []string{"list", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}", "--limit", fmt.Sprintf("%d", 2)},
			wantError: false,
		},
		{
			name:      "error from invalid namespace",
			command:   command(t, trs, now, ns),
			args:      []string{"list", "-n", "invalid"},
			wantError: true,
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

func TestListTaskRuns_no_condition(t *testing.T) {
	now := time.Now()
	aMinute, _ := time.ParseDuration("1m")

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr1-1", "foo",
			tb.TaskRunLabel("tekton.dev/task", "bar"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("bar")),
			tb.TaskRunStatus(
				tb.TaskRunStartTime(now),
				taskRunCompletionTime(now.Add(aMinute)),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
	}

	cmd := command(t, trs, now, ns)
	got, err := test.ExecuteCommand(cmd, "list", "bar", "-n", "foo")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func command(t *testing.T, trs []*v1alpha1.TaskRun, now time.Time, ns []*corev1.Namespace) *cobra.Command {
	// fake clock advanced by 1 hour
	clock := clockwork.NewFakeClockAt(now)
	clock.Advance(time.Duration(60) * time.Minute)

	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Namespaces: ns})

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube}

	return Command(p)
}

func taskRunCompletionTime(ct time.Time) tb.TaskRunStatusOp {
	return func(s *v1alpha1.TaskRunStatus) {
		s.CompletionTime = &metav1.Time{Time: ct}
	}
}
