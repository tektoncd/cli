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

package pipelineresource

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPipelineResourceList(t *testing.T) {

	pres := []*v1alpha1.PipelineResource{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test-ns-1",
			},
			Spec: v1alpha1.PipelineResourceSpec{
				Type: v1alpha1.PipelineResourceTypeGit,
				Params: []v1alpha1.ResourceParam{
					{
						Name:  "url",
						Value: "git@github.com:tektoncd/cli-new.git",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-1",
				Namespace: "test-ns-1",
			},
			Spec: v1alpha1.PipelineResourceSpec{
				Type: v1alpha1.PipelineResourceTypeImage,
				Params: []v1alpha1.ResourceParam{
					{
						Name:  "URL",
						Value: "quey.io/tekton/controller",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-2",
				Namespace: "test-ns-1",
			},
			Spec: v1alpha1.PipelineResourceSpec{
				Type: v1alpha1.PipelineResourceTypeGit,
				Params: []v1alpha1.ResourceParam{
					{
						Name:  "url",
						Value: "git@github.com:tektoncd/cli.git",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-3",
				Namespace: "test-ns-1",
			},
			Spec: v1alpha1.PipelineResourceSpec{
				Type: v1alpha1.PipelineResourceTypeImage,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-4",
				Namespace: "test-ns-2",
			},
			Spec: v1alpha1.PipelineResourceSpec{
				Type: v1alpha1.PipelineResourceTypeImage,
				Params: []v1alpha1.ResourceParam{
					{
						Name:  "URL",
						Value: "quey.io/tekton/webhook",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-5",
				Namespace: "test-ns-1",
			},
			Spec: v1alpha1.PipelineResourceSpec{
				Type: v1alpha1.PipelineResourceTypeCloudEvent,
				Params: []v1alpha1.ResourceParam{
					{
						Name:  "targetURI",
						Value: "http://sink",
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
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

	tests := []struct {
		name      string
		command   *cobra.Command
		args      []string
		wantError bool
	}{
		{
			name:      "Invalid namespace",
			command:   command(t, pres, ns),
			args:      []string{"list", "-n", "invalid"},
			wantError: true,
		},
		{
			name:      "Multiple pipeline resources",
			command:   command(t, pres, ns),
			args:      []string{"list", "-n", "test-ns-1"},
			wantError: false,
		},
		{
			name:      "Single pipeline resource",
			command:   command(t, pres, ns),
			args:      []string{"list", "-n", "test-ns-2"},
			wantError: false,
		},
		{
			name:      "Single Pipeline Resource by type",
			command:   command(t, pres, ns),
			args:      []string{"list", "-n", "test-ns-2", "-t", "image"},
			wantError: false,
		},
		{
			name:      "Multiple Pipeline Resource by type",
			command:   command(t, pres, ns),
			args:      []string{"list", "-n", "test-ns-1", "-t", "image"},
			wantError: false,
		},
		{
			name:      "Empty Pipeline Resource by type",
			command:   command(t, pres, ns),
			args:      []string{"list", "-n", "test-ns-1", "-t", "storage"},
			wantError: false,
		},
		{
			name:      "By template",
			command:   command(t, pres, ns),
			args:      []string{"list", "-n", "test-ns-1", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}"},
			wantError: false,
		},
		{
			name:      "No Headers",
			command:   command(t, pres, ns),
			args:      []string{"list", "-n", "test-ns-1", "--no-headers"},
			wantError: false,
		},
		{
			name:      "All Namespaces",
			command:   command(t, pres, ns),
			args:      []string{"list", "--all-namespaces"},
			wantError: false,
		},
	}

	for _, td := range tests {
		t.Run(td.name, func(t *testing.T) {
			out, err := test.ExecuteCommand(td.command, td.args...)

			if !td.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			golden.Assert(t, out, strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
		})
	}

}

func TestPipelineResourceList_empty(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ns-3",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource}
	pipelineresource := Command(p)

	out, _ := test.ExecuteCommand(pipelineresource, "list", "-n", "test-ns-3")
	test.AssertOutput(t, "Command \"list\" is deprecated, PipelineResource commands are deprecated, they will be removed soon as it get removed from API.\n"+msgNoPREsFound+"\n", out)
}

func TestPipelineResourceList_invalidType(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource}
	c := Command(p)

	_, err := test.ExecuteCommand(c, "list", "-n", "ns", "-t", "registry")

	if err == nil {
		t.Error("Expecting an error but it's empty")
	}

	test.AssertOutput(t, "failed to list pipelineresources. Invalid resource type registry", err.Error())
}

func command(t *testing.T, pres []*v1alpha1.PipelineResource, ns []*corev1.Namespace) *cobra.Command {
	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{PipelineResources: pres, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource}
	return Command(p)
}
