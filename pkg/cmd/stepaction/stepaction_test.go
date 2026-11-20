// Copyright © 2026 The Tekton Authors.
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

package stepaction

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

const versionv1beta1 = "v1beta1"

func testStepActions(now time.Time) []*v1beta1.StepAction {
	return []*v1beta1.StepAction{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "kaniko",
				Namespace:         "foo",
				CreationTimestamp: metav1.Time{Time: now},
			},
			Spec: v1beta1.StepActionSpec{
				Description: "Builds an image",
				Image:       "gcr.io/kaniko-project/executor",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "git-clone",
				Namespace:         "foo",
				Labels:            map[string]string{"app": "git"},
				CreationTimestamp: metav1.Time{Time: now},
			},
			Spec: v1beta1.StepActionSpec{
				Description: "Clones a repo",
				Image:       "alpine/git",
				Command:     []string{"git"},
				Args:        []string{"clone"},
				Params: v1.ParamSpecs{
					{Name: "url", Type: v1.ParamTypeString, Description: "repo url"},
					{Name: "revision", Type: v1.ParamTypeString, Description: "git revision", Default: v1.NewStructuredValues("main")},
				},
				Results: []v1.StepResult{
					{Name: "commit", Description: "the cloned commit"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "lint",
				Namespace:         "bar",
				CreationTimestamp: metav1.Time{Time: now},
			},
			Spec: v1beta1.StepActionSpec{
				Description: "Runs a linter",
				Image:       "golangci/golangci-lint",
			},
		},
	}
}

func newDynamicClient(t *testing.T, sas []*v1beta1.StepAction) dynamic.Interface {
	t.Helper()
	tdc := testDynamic.Options{}
	objs := make([]runtime.Object, 0, len(sas))
	for _, sa := range sas {
		objs = append(objs, cb.UnstructuredV1beta1StepAction(sa, versionv1beta1))
	}
	dc, err := tdc.Client(objs...)
	if err != nil {
		t.Fatalf("unable to create dynamic client: %v", err)
	}
	return dc
}

func newCommand(t *testing.T, now time.Time, dc dynamic.Interface) *cobra.Command {
	t.Helper()
	clock := clockwork.NewFakeClockAt(now)
	clock.Advance(time.Hour)
	ns := []*corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "bar"}},
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"stepaction"})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Dynamic: dc}
	return Command(p)
}

func TestStepActionList(t *testing.T) {
	now := time.Now()
	dc := newDynamicClient(t, testStepActions(now))

	out, err := test.ExecuteCommand(newCommand(t, now, dc), "list", "-n", "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected a header and 2 rows, got %q", out)
	}
	if got := strings.Fields(lines[0]); strings.Join(got, " ") != "NAME DESCRIPTION AGE" {
		t.Errorf("unexpected header %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "git-clone ") || !strings.Contains(lines[1], "Clones a repo") {
		t.Errorf("unexpected first row %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "kaniko ") || !strings.Contains(lines[2], "Builds an image") {
		t.Errorf("unexpected second row %q", lines[2])
	}
	if strings.Contains(out, "lint") {
		t.Errorf("StepAction from another namespace was listed: %q", out)
	}
}

func TestStepActionListOptions(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		args  []string
		check func(t *testing.T, out string)
	}{
		{
			name: "all namespaces",
			args: []string{"list", "-A"},
			check: func(t *testing.T, out string) {
				lines := strings.Split(strings.TrimSpace(out), "\n")
				if len(lines) != 4 || !strings.HasPrefix(lines[0], "NAMESPACE ") {
					t.Fatalf("unexpected output %q", out)
				}
				if !strings.HasPrefix(lines[1], "bar ") || !strings.Contains(lines[1], "lint") {
					t.Errorf("expected namespace bar first, got %q", lines[1])
				}
			},
		},
		{
			name: "no headers and reverse",
			args: []string{"list", "-n", "foo", "--no-headers", "--reverse"},
			check: func(t *testing.T, out string) {
				lines := strings.Split(strings.TrimSpace(out), "\n")
				if len(lines) != 2 || !strings.HasPrefix(lines[0], "kaniko ") {
					t.Errorf("unexpected output %q", out)
				}
			},
		},
		{
			name: "limit",
			args: []string{"list", "-n", "foo", "--limit", "1"},
			check: func(t *testing.T, out string) {
				lines := strings.Split(strings.TrimSpace(out), "\n")
				if len(lines) != 2 || !strings.HasPrefix(lines[1], "git-clone ") {
					t.Errorf("unexpected output %q", out)
				}
			},
		},
		{
			name: "output name",
			args: []string{"list", "-n", "foo", "-o", "name"},
			check: func(t *testing.T, out string) {
				want := "stepaction.tekton.dev/git-clone\nstepaction.tekton.dev/kaniko\n"
				if out != want {
					t.Errorf("output = %q, want %q", out, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := newDynamicClient(t, testStepActions(now))
			out, err := test.ExecuteCommand(newCommand(t, now, dc), tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, out)
		})
	}
}

func TestStepActionListEmpty(t *testing.T) {
	now := time.Now()
	dc := newDynamicClient(t, nil)
	out, err := test.ExecuteCommand(newCommand(t, now, dc), "list", "-n", "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "No StepActions found\n" {
		t.Errorf("output = %q", out)
	}
}

func TestStepActionListNegativeLimit(t *testing.T) {
	now := time.Now()
	dc := newDynamicClient(t, nil)
	if _, err := test.ExecuteCommand(newCommand(t, now, dc), "list", "-n", "foo", "--limit", "-1"); err == nil {
		t.Error("expected an error for a negative limit")
	}
}

func TestStepActionDescribe(t *testing.T) {
	now := time.Now()
	dc := newDynamicClient(t, testStepActions(now))

	out, err := test.ExecuteCommand(newCommand(t, now, dc), "describe", "git-clone", "-n", "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"git-clone", "foo", "Clones a repo", "alpine/git", "[git]", "[clone]",
		"app=git", "url", "repo url", "---", "revision", "main",
		"commit", "string", "the cloned commit",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("describe output is missing %q:\n%s", want, out)
		}
	}
}

func TestStepActionDescribeNotFound(t *testing.T) {
	now := time.Now()
	dc := newDynamicClient(t, testStepActions(now))
	_, err := test.ExecuteCommand(newCommand(t, now, dc), "describe", "missing", "-n", "foo")
	if err == nil || !strings.Contains(err.Error(), "failed to get StepAction missing") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStepActionDelete(t *testing.T) {
	now := time.Now()
	dc := newDynamicClient(t, testStepActions(now))

	out, err := test.ExecuteCommand(newCommand(t, now, dc), "delete", "kaniko", "-n", "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "StepAction 'kaniko' deleted successfully from namespace 'foo'\n" {
		t.Errorf("output = %q", out)
	}
	if _, err := dc.Resource(stepactionGroupResource).Namespace("foo").Get(context.TODO(), "kaniko", metav1.GetOptions{}); err == nil {
		t.Error("StepAction kaniko still exists")
	}
	if _, err := dc.Resource(stepactionGroupResource).Namespace("foo").Get(context.TODO(), "git-clone", metav1.GetOptions{}); err != nil {
		t.Errorf("StepAction git-clone should still exist: %v", err)
	}
}

func TestStepActionDeleteMissing(t *testing.T) {
	now := time.Now()
	dc := newDynamicClient(t, testStepActions(now))

	_, err := test.ExecuteCommand(newCommand(t, now, dc), "delete", "missing", "kaniko", "-n", "foo")
	if err == nil || !strings.Contains(err.Error(), "StepAction missing not found in namespace foo") {
		t.Errorf("unexpected error: %v", err)
	}
	// the existing one is still deleted
	if _, err := dc.Resource(stepactionGroupResource).Namespace("foo").Get(context.TODO(), "kaniko", metav1.GetOptions{}); err == nil {
		t.Error("StepAction kaniko still exists")
	}
}

func TestStepActionDeleteRequiresName(t *testing.T) {
	now := time.Now()
	dc := newDynamicClient(t, nil)
	if _, err := test.ExecuteCommand(newCommand(t, now, dc), "delete", "-n", "foo"); err == nil {
		t.Error("expected an error without a StepAction name")
	}
}
