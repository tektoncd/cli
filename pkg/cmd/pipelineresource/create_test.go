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
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
	goexpect "github.com/Netflix/go-expect"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func getPipelineResource() *v1alpha1.PipelineResource {
	return &v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "res",
			Namespace: "namespace",
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: v1alpha1.PipelineResourceTypeImage,
			Params: []v1alpha1.ResourceParam{
				{
					Name:  "url",
					Value: "git@github.com:tektoncd/cli.git",
				},
			},
		},
	}
}

func TestPipelineResource_resource_noName(t *testing.T) {
	pres := getPipelineResource()
	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{
		PipelineResources: []*v1alpha1.PipelineResource{
			pres,
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace",
				},
			},
		},
	})

	tests := []promptTest{
		{
			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Sorry, your reply was invalid: Value is required"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("res"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				if err := c.Close(); err != nil {
					return err
				}

				return nil
			},
		},
	}

	res := resOpts("namespace", cs)
	for _, test := range tests {
		t.Run("no input for name", func(t *testing.T) {
			t.Skip("Skipping due of flakiness")
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_resource_already_exist(t *testing.T) {
	t.Skip("Skipping due of flakiness")
	pres := getPipelineResource()
	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{
		PipelineResources: []*v1alpha1.PipelineResource{
			pres,
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace",
				},
			},
		},
	})

	tests := []promptTest{
		{
			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("res"); err != nil {
					return err
				}

				if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				if err := c.Close(); err != nil {
					return err
				}

				return nil
			},
		},
	}

	res := resOpts("namespace", cs)
	for _, test := range tests {
		t.Run("pre-existing-resource", func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_allResourceType(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace",
				},
			},
		},
	})

	tests := []promptTest{
		{
			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("res"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("git"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("image"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("pullRequest"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("storage"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for url :"); err != nil {
					return err
				}

				if _, err := c.SendLine(""); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				if err := c.Close(); err != nil {
					return err
				}
				return nil
			},
		},
	}

	res := resOpts("namespace", cs)
	for _, test := range tests {
		t.Run("check all type of resource", func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_gitResource(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace",
				},
			},
		},
	})

	tests := []promptTest{
		{
			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("git-res"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("git"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for url :"); err != nil {
					return err
				}

				if _, err := c.SendLine("https://github.com/pradeepitm12"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}
				if _, err := c.ExpectString("Enter a value for revision :"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				// check if the resource is created
				res, err := cs.Resource.TektonV1alpha1().PipelineResources("namespace").Get(context.Background(), "git-res", metav1.GetOptions{})
				if err != nil {
					return err
				}

				if res.Name != "git-res" {
					return errors.New("unexpected error")
				}

				return nil
			},
		},
	}
	res := resOpts("namespace", cs)
	for _, test := range tests {
		t.Run("gitResource", func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_imageResource(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace",
				},
			},
		},
	})

	tests := []promptTest{
		{
			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("image-res"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("git"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("image"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for url :"); err != nil {
					return err
				}

				if _, err := c.SendLine("gcr.io/staging-images/kritis"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for digest :"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				// check if the resource is created
				res, err := cs.Resource.TektonV1alpha1().PipelineResources("namespace").Get(context.Background(), "image-res", metav1.GetOptions{})
				if err != nil {
					return err
				}

				if res.Name != "image-res" {
					return errors.New("unexpected error")
				}

				return nil
			},
		},
	}
	res := resOpts("namespace", cs)
	for _, test := range tests {
		t.Run("imageResource", func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_pullRequestResource(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace",
				},
			},
		},
	})

	tests := []promptTest{
		{
			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("pullRequest-res"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("git"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("image"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("pullRequest"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for url :"); err != nil {
					return err
				}

				if _, err := c.SendLine("https://github.com/tektoncd/cli/pull/1"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Do you want to set secrets ?"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Yes"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Secret Key for githubToken"); err != nil {
					return err
				}

				if _, err := c.SendLine("githubToken"); err != nil {
					return err
				}
				if _, err := c.ExpectString("Secret Name for githubToken"); err != nil {
					return err
				}

				if _, err := c.SendLine("github-secrets"); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				// check if the resource is created
				res, err := cs.Resource.TektonV1alpha1().PipelineResources("namespace").Get(context.Background(), "pullRequest-res", metav1.GetOptions{})
				if err != nil {
					return err
				}

				if res.Name != "pullRequest-res" {
					return errors.New("unexpected error")
				}

				return nil
			},
		},
	}
	res := resOpts("namespace", cs)
	for _, test := range tests {
		t.Run("pullRequestResource", func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_gcsStorageResource(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace",
				},
			},
		},
	})

	tests := []promptTest{
		{
			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("storage-res"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("git"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("image"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("pullRequest"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("storage"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("gcs"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for location :"); err != nil {
					return err
				}

				if _, err := c.SendLine("gs://some-bucket"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for dir :"); err != nil {
					return err
				}

				if _, err := c.SendLine("/home"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Secret Key for GOOGLE_APPLICATION_CREDENTIALS :"); err != nil {
					return err
				}

				if _, err := c.SendLine("service_account.json"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Secret Name for GOOGLE_APPLICATION_CREDENTIALS :"); err != nil {
					return err
				}

				if _, err := c.SendLine("bucket-sa"); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				// check if the resource is created
				res, err := cs.Resource.TektonV1alpha1().PipelineResources("namespace").Get(context.Background(), "storage-res", metav1.GetOptions{})
				if err != nil {
					return err
				}

				if res.Name != "storage-res" {
					return errors.New("unexpected error")
				}

				return nil
			},
		},
	}
	res := resOpts("namespace", cs)
	for _, test := range tests {
		t.Run("gcsStorageResource", func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_buildGCSstorageResource(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedV1beta1TestData(t, pipelinetest.Data{
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace",
				},
			},
		},
	})

	tests := []promptTest{
		{
			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("storage-res"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("git"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("image"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("pullRequest"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("storage"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("gcs"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("build-gcs"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for location :"); err != nil {
					return err
				}

				if _, err := c.SendLine("gs://build-crd-tests/rules_docker-master.zip"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select an artifact type"); err != nil {
					return err
				}

				if _, err := c.ExpectString("ZipArchive"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("TarGzArchive"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Manifest"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Secret Key for GOOGLE_APPLICATION_CREDENTIALS :"); err != nil {
					return err
				}

				if _, err := c.SendLine("service_account.json"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Secret Name for GOOGLE_APPLICATION_CREDENTIALS :"); err != nil {
					return err
				}

				if _, err := c.SendLine("bucket-sa"); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				// check if the resource is created
				res, err := cs.Resource.TektonV1alpha1().PipelineResources("namespace").Get(context.Background(), "storage-res", metav1.GetOptions{})
				if err != nil {
					return err
				}

				if res.Name != "storage-res" {
					return errors.New("unexpected error")
				}

				return nil
			},
		},
	}
	res := resOpts("namespace", cs)
	for _, test := range tests {
		t.Run("buildGCSstorageResource", func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func resOpts(ns string, cs pipelinetest.Clients) *Resource {

	p := test.Params{
		Kube:     cs.Kube,
		Tekton:   cs.Pipeline,
		Resource: cs.Resource,
	}
	out := new(bytes.Buffer)
	p.SetNamespace(ns)
	resOp := Resource{
		Params: &p,
		stream: &cli.Stream{Out: out, Err: out},
	}

	return &resOp
}
