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

	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPipelineResourceDescribe_Invalid_Namespace(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource}

	res := Command(p)
	out, err := test.ExecuteCommand(res, "desc", "bar", "-n", "invalid")
	if err == nil {
		t.Errorf("Error expected here")
	}

	expected := "Command \"describe\" is deprecated, PipelineResource commands are deprecated, they will be removed soon as it get removed from API.\nError: failed to find pipelineresource \"bar\"\n"
	test.AssertOutput(t, expected, out)
}

func TestPipelineResourceDescribe_Empty(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource}

	res := Command(p)
	_, err := test.ExecuteCommand(res, "desc", "bar")
	if err == nil {
		t.Errorf("Error expected here")
	}

	expected := "failed to find pipelineresource \"bar\""
	test.AssertOutput(t, expected, err.Error())
}

func TestPipelineResourceDescribe_WithParams(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ns-1",
			},
		},
	}

	pres := []*v1alpha1.PipelineResource{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-1",
				Namespace: "test-ns-1",
			},
			Spec: v1alpha1.PipelineResourceSpec{
				Type:        v1alpha1.PipelineResourceTypeImage,
				Description: "a test description",
				Params: []v1alpha1.ResourceParam{
					{
						Name:  "URL",
						Value: "quay.io/tekton/controller",
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{PipelineResources: pres, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource}
	pipelineresource := Command(p)
	out, _ := test.ExecuteCommand(pipelineresource, "desc", "test-1", "-n", "test-ns-1")
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineResourceDescribe_WithSecretParams(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ns-1",
			},
		},
	}

	pres := []*v1alpha1.PipelineResource{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-1",
				Namespace: "test-ns-1",
			},
			Spec: v1alpha1.PipelineResourceSpec{
				Type:        v1alpha1.PipelineResourceTypeImage,
				Description: "a test description",
				Params: []v1alpha1.ResourceParam{
					{
						Name:  "URL",
						Value: "quay.io/tekton/controller",
					},
					{
						Name:  "TAG",
						Value: "latest",
					},
				},
				SecretParams: []v1alpha1.SecretParam{
					{
						FieldName:  "githubToken",
						SecretKey:  "token",
						SecretName: "github-secrets",
					},
				},
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{PipelineResources: pres, Namespaces: ns})
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource}
	pipelineresource := Command(p)
	out, _ := test.ExecuteCommand(pipelineresource, "desc", "test-1", "-n", "test-ns-1")
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestPipelineResourcesDescribe_custom_output(t *testing.T) {
	name := "pipeline-resource"
	expected := "Command \"describe\" is deprecated, PipelineResource commands are deprecated, they will be removed soon as it get removed from API.\npipelineresource.tekton.dev/" + name

	clock := test.FakeClock()

	prs := []*v1alpha1.PipelineResource{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{
		PipelineResources: prs,
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Resource: cs.Resource}
	p.SetNamespace("ns")
	run := Command(p)

	got, err := test.ExecuteCommand(run, "desc", "-o", "name", name)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	got = strings.TrimSpace(got)
	if got != expected {
		t.Errorf("Result should be '%s' != '%s'", got, expected)
	}
}
