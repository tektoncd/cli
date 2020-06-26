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
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	k8stest "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestListTaskRuns(t *testing.T) {
	version := "v1alpha1"
	now := time.Now()
	aMinute, _ := time.ParseDuration("1m")
	twoMinute, _ := time.ParseDuration("2m")

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr0-1",
			tb.TaskRunNamespace("foo"),
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random", tb.TaskRefKind(v1alpha1.NamespacedTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
		tb.TaskRun("tr1-1",
			tb.TaskRunNamespace("foo"),
			tb.TaskRunLabel("tekton.dev/task", "bar"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("bar", tb.TaskRefKind(v1alpha1.NamespacedTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
				tb.TaskRunStartTime(now),
				taskRunCompletionTime(now.Add(aMinute)),
			),
		),
		tb.TaskRun("tr2-1",
			tb.TaskRunNamespace("foo"),
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random", tb.TaskRefKind(v1alpha1.NamespacedTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionUnknown,
					Reason: resources.ReasonRunning,
				}),
				tb.TaskRunStartTime(now),
			),
		),
		tb.TaskRun("tr2-2",
			tb.TaskRunNamespace("foo"),
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunLabel("pot", "nutella"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random", tb.TaskRefKind(v1alpha1.NamespacedTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionFalse,
					Reason: resources.ReasonFailed,
				}),
				tb.TaskRunStartTime(now.Add(aMinute)),
				taskRunCompletionTime(now.Add(twoMinute)),
			),
		),
		tb.TaskRun("tr3-1",
			tb.TaskRunNamespace("foo"),
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunLabel("pot", "honey"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random", tb.TaskRefKind(v1alpha1.NamespacedTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionFalse,
					Reason: resources.ReasonFailed,
				}),
			),
		),
		tb.TaskRun("tr4-1",
			tb.TaskRunNamespace("foo"),
			tb.TaskRunLabel("tekton.dev/task", "bar"),
			tb.TaskRunLabel("pot", "honey"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("bar", tb.TaskRefKind(v1alpha1.ClusterTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionFalse,
					Reason: resources.ReasonFailed,
				}),
			),
		),
	}

	trsMultipleNs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr4-1",
			tb.TaskRunNamespace("tout"),
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random", tb.TaskRefKind(v1alpha1.NamespacedTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
		tb.TaskRun("tr4-2",
			tb.TaskRunNamespace("lacher"),
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random", tb.TaskRefKind(v1alpha1.NamespacedTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
		tb.TaskRun("tr5-1",
			tb.TaskRunNamespace("lacher"),
			tb.TaskRunLabel("tekton.dev/task", "random"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("random", tb.TaskRefKind(v1alpha1.ClusterTaskKind))),
			tb.TaskRunStatus(
				tb.StatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
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
	tdc1 := testDynamic.Options{}
	dc1, err := tdc1.Client(
		cb.UnstructuredTR(trs[0], version),
		cb.UnstructuredTR(trs[1], version),
		cb.UnstructuredTR(trs[2], version),
		cb.UnstructuredTR(trs[3], version),
		cb.UnstructuredTR(trs[4], version),
		cb.UnstructuredTR(trs[5], version),
		cb.UnstructuredTR(trsMultipleNs[0], version),
		cb.UnstructuredTR(trsMultipleNs[1], version),
		cb.UnstructuredTR(trsMultipleNs[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredTR(trsMultipleNs[0], version),
		cb.UnstructuredTR(trsMultipleNs[1], version),
		cb.UnstructuredTR(trsMultipleNs[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	tdc3 := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{
			{
				Verb:     "list",
				Resource: "taskruns",
				Action: func(action k8stest.Action) (bool, runtime.Object, error) {
					fieldSelector := action.(k8stest.ListAction).GetListRestrictions().Fields.String()
					if fieldSelector == "metadata.name=tr0-1" {
						unstructuredTR := cb.UnstructuredTR(trs[0], version)
						list := &unstructured.UnstructuredList{}
						list.Items = append(list.Items, *unstructuredTR)
						return true, list, nil
					} else if fieldSelector == "a=b" {
						return true, nil, errors.New("field label not supported: a")
					}
					return false, nil, nil
				},
			},
		},
	}
	dc3, err := tdc3.Client(
		cb.UnstructuredTR(trs[0], version),
		cb.UnstructuredTR(trs[1], version),
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
			name:      "by Task name",
			command:   commandV1alpha1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "bar", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "by output as name",
			command:   commandV1alpha1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "-o", "name"},
			wantError: false,
		},
		{
			name:      "all in namespace",
			command:   commandV1alpha1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "print by template",
			command:   commandV1alpha1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
		{
			name:    "empty list",
			command: commandV1alpha1(t, trs, now, ns, version, dc1),
			args:    []string{"list", "-n", "random"},
		},
		{
			name:      "limit taskruns returned to 1",
			command:   commandV1alpha1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "--limit", fmt.Sprintf("%d", 1)},
			wantError: false,
		},
		{
			name:      "limit taskruns negative case",
			command:   commandV1alpha1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "--limit", fmt.Sprintf("%d", -1)},
			wantError: true,
		},
		{
			name:      "limit taskruns greater than maximum case",
			command:   commandV1alpha1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "--limit", fmt.Sprintf("%d", 7)},
			wantError: false,
		},
		{
			name:      "limit taskruns with output flag set",
			command:   commandV1alpha1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}", "--limit", fmt.Sprintf("%d", 2)},
			wantError: false,
		},
		{
			name:      "error from invalid namespace",
			command:   commandV1alpha1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "invalid"},
			wantError: true,
		},
		{
			name:      "filter taskruns by label",
			command:   commandV1alpha1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "--label", "pot=honey"},
			wantError: false,
		},
		{
			name:      "filter taskruns by label with in query",
			command:   commandV1alpha1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "--label", "pot in (honey,nutella)"},
			wantError: false,
		},
		{
			name:      "no mixing pipelinename and labels",
			command:   commandV1alpha1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "--label", "honey=nutella", "tr3-1"},
			wantError: true,
		},
		{
			name:      "print in reverse",
			command:   commandV1alpha1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "--reverse", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "print in reverse with output flag",
			command:   commandV1alpha1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "--reverse", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
		{
			name:      "print taskruns in all namespaces",
			command:   commandV1alpha1(t, trsMultipleNs, now, ns, version, dc2),
			args:      []string{"list", "--all-namespaces"},
			wantError: false,
		},
		{
			name:      "print taskruns without headers",
			command:   commandV1alpha1(t, trsMultipleNs, now, ns, version, dc2),
			args:      []string{"list", "--no-headers", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "print taskruns in all namespaces without headers",
			command:   commandV1alpha1(t, trsMultipleNs, now, ns, version, dc2),
			args:      []string{"list", "--all-namespaces", "--no-headers"},
			wantError: false,
		},
		{
			name:      "list taskruns by field-selector",
			command:   commandV1alpha1(t, trs, now, ns, version, dc3),
			args:      []string{"list", "-n", "foo", "--field-selector", "metadata.name=tr0-1"},
			wantError: false,
		},
		{
			name:      "list taskruns by field-selector negative case",
			command:   commandV1alpha1(t, trs, now, ns, version, dc3),
			args:      []string{"list", "-n", "foo", "--field-selector", "a=b"},
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

func TestListTaskRuns_v1beta1(t *testing.T) {
	version := "v1beta1"
	now := time.Now()
	aMinute, _ := time.ParseDuration("1m")
	twoMinute, _ := time.ParseDuration("2m")

	trs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "tr0-1",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "tr1-1",
				Labels:    map[string]string{"tekton.dev/task": "bar"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "bar",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: now},
					CompletionTime: &metav1.Time{Time: now.Add(aMinute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "tr2-1",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionUnknown,
							Reason: resources.ReasonRunning,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime: &metav1.Time{Time: now},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "tr2-2",
				Labels:    map[string]string{"tekton.dev/task": "random", "pot": "nutella"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: resources.ReasonFailed,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					StartTime:      &metav1.Time{Time: now.Add(aMinute)},
					CompletionTime: &metav1.Time{Time: now.Add(twoMinute)},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "tr3-1",
				Labels:    map[string]string{"tekton.dev/task": "random", "pot": "honey"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: resources.ReasonFailed,
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
				Name:      "tr4-1",
				Labels:    map[string]string{"tekton.dev/task": "bar", "pot": "honey"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "bar",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: resources.ReasonFailed,
						},
					},
				},
			},
		},
	}

	trsMultipleNs := []*v1beta1.TaskRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "tout",
				Name:      "tr4-1",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "lacher",
				Name:      "tr4-2",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.NamespacedTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "lacher",
				Name:      "tr5-1",
				Labels:    map[string]string{"tekton.dev/task": "random"},
			},
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name: "random",
					Kind: v1beta1.ClusterTaskKind,
				},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
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
		cb.UnstructuredV1beta1TR(trs[0], version),
		cb.UnstructuredV1beta1TR(trs[1], version),
		cb.UnstructuredV1beta1TR(trs[2], version),
		cb.UnstructuredV1beta1TR(trs[3], version),
		cb.UnstructuredV1beta1TR(trs[4], version),
		cb.UnstructuredV1beta1TR(trs[5], version),
		cb.UnstructuredV1beta1TR(trsMultipleNs[0], version),
		cb.UnstructuredV1beta1TR(trsMultipleNs[1], version),
		cb.UnstructuredV1beta1TR(trsMultipleNs[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1TR(trsMultipleNs[0], version),
		cb.UnstructuredV1beta1TR(trsMultipleNs[1], version),
		cb.UnstructuredV1beta1TR(trsMultipleNs[2], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	tdc3 := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{
			{
				Verb:     "list",
				Resource: "taskruns",
				Action: func(action k8stest.Action) (bool, runtime.Object, error) {
					fieldSelector := action.(k8stest.ListAction).GetListRestrictions().Fields.String()
					if fieldSelector == "metadata.name!=tr0-1" {
						unstructuredTR := cb.UnstructuredV1beta1TR(trs[1], version)
						list := &unstructured.UnstructuredList{}
						list.Items = append(list.Items, *unstructuredTR)
						return true, list, nil
					} else if fieldSelector == "a=b" {
						errStr := "field label not supported: a"
						return true, nil, errors.New(errStr)
					}
					return false, nil, nil
				},
			},
		},
	}
	dc3, err := tdc3.Client(
		cb.UnstructuredV1beta1TR(trs[0], version),
		cb.UnstructuredV1beta1TR(trs[1], version),
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
			name:      "by Task name",
			command:   commandV1beta1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "bar", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "by output as name",
			command:   commandV1beta1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "-o", "name"},
			wantError: false,
		},
		{
			name:      "all in namespace",
			command:   commandV1beta1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "print by template",
			command:   commandV1beta1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
		{
			name:    "empty list",
			command: commandV1beta1(t, trs, now, ns, version, dc1),
			args:    []string{"list", "-n", "random"},
		},
		{
			name:      "limit taskruns returned to 1",
			command:   commandV1beta1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "--limit", fmt.Sprintf("%d", 1)},
			wantError: false,
		},
		{
			name:      "limit taskruns negative case",
			command:   commandV1beta1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "--limit", fmt.Sprintf("%d", -1)},
			wantError: true,
		},
		{
			name:      "limit taskruns greater than maximum case",
			command:   commandV1beta1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "--limit", fmt.Sprintf("%d", 7)},
			wantError: false,
		},
		{
			name:      "limit taskruns with output flag set",
			command:   commandV1beta1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}", "--limit", fmt.Sprintf("%d", 2)},
			wantError: false,
		},
		{
			name:      "error from invalid namespace",
			command:   commandV1beta1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "invalid"},
			wantError: true,
		},
		{
			name:      "filter taskruns by label",
			command:   commandV1beta1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "--label", "pot=honey"},
			wantError: false,
		},
		{
			name:      "filter taskruns by label with in query",
			command:   commandV1beta1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "--label", "pot in (honey,nutella)"},
			wantError: false,
		},
		{
			name:      "no mixing pipelinename and labels",
			command:   commandV1beta1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "-n", "foo", "--label", "honey=nutella", "tr3-1"},
			wantError: true,
		},
		{
			name:      "print in reverse",
			command:   commandV1beta1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "--reverse", "-n", "foo"},
			wantError: false,
		},
		{
			name:      "print in reverse with output flag",
			command:   commandV1beta1(t, trs, now, ns, version, dc1),
			args:      []string{"list", "--reverse", "-n", "foo", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
		{
			name:      "print taskruns in all namespaces",
			command:   commandV1beta1(t, trsMultipleNs, now, ns, version, dc2),
			args:      []string{"list", "--all-namespaces"},
			wantError: false,
		},
		{
			name:      "list taskruns by field-selector",
			command:   commandV1beta1(t, trs, now, ns, version, dc3),
			args:      []string{"list", "-n", "foo", "--field-selector", "metadata.name!=tr0-1"},
			wantError: false,
		},
		{
			name:      "list taskruns by field-selector negative case",
			command:   commandV1beta1(t, trs, now, ns, version, dc3),
			args:      []string{"list", "-n", "foo", "--field-selector", "a=b"},
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
	version := "v1alpha1"
	now := time.Now()
	aMinute, _ := time.ParseDuration("1m")

	trs := []*v1alpha1.TaskRun{
		tb.TaskRun("tr1-1", tb.TaskRunNamespace("foo"),
			tb.TaskRunLabel("tekton.dev/task", "bar"),
			tb.TaskRunSpec(tb.TaskRunTaskRef("bar", tb.TaskRefKind(v1alpha1.NamespacedTaskKind))),
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
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredTR(trs[0], version),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	cmd := commandV1alpha1(t, trs, now, ns, version, dc)
	got, err := test.ExecuteCommand(cmd, "list", "bar", "-n", "foo")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func commandV1alpha1(t *testing.T, trs []*v1alpha1.TaskRun, now time.Time, ns []*corev1.Namespace, version string, dc dynamic.Interface) *cobra.Command {
	// fake clock advanced by 1 hour
	clock := clockwork.NewFakeClockAt(now)
	clock.Advance(time.Duration(60) * time.Minute)
	cs, _ := test.SeedTestData(t, pipelinetest.Data{TaskRuns: trs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}

	return Command(p)
}

func commandV1beta1(t *testing.T, trs []*v1beta1.TaskRun, now time.Time, ns []*corev1.Namespace, version string, dc dynamic.Interface) *cobra.Command {
	// fake clock advanced by 1 hour
	clock := clockwork.NewFakeClockAt(now)
	clock.Advance(time.Duration(60) * time.Minute)
	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{TaskRuns: trs, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"taskrun"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}

	return Command(p)
}

func taskRunCompletionTime(ct time.Time) tb.TaskRunStatusOp {
	return func(s *v1alpha1.TaskRunStatus) {
		s.CompletionTime = &metav1.Time{Time: ct}
	}
}
