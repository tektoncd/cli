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
	"errors"
	"io"
	"testing"

	"github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
	goexpect "github.com/Netflix/go-expect"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestPipelineResource_resource_noName(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		PipelineResources: []*v1alpha1.PipelineResource{
			tb.PipelineResource("res", "namespace",
				tb.PipelineResourceSpec("git",
					tb.PipelineResourceSpecParam("url", "git@github.com:tektoncd/cli.git"),
				)),
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
			name: "no input for name",

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
		t.Run(test.name, func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_resource_already_exist(t *testing.T) {
	t.Skip("Skipping due of flakiness")
	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		PipelineResources: []*v1alpha1.PipelineResource{
			tb.PipelineResource("res", "namespace",
				tb.PipelineResourceSpec("git",
					tb.PipelineResourceSpecParam("url", "git@github.com:tektoncd/cli.git"),
				),
			),
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
			name: "pre-existing-resource",

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
		t.Run(test.name, func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_allResourceType(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
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
			name: "check all type of resource",

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

				if _, err := c.ExpectString("cloudEvent"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("cluster"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
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

				if _, err := c.ExpectString("Enter a value for targetURI :"); err != nil {
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
		t.Run(test.name, func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_cloudEventResource(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
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
			name: "create-cloudEventResource",

			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("cloudEvent-res"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("cloudEvent"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for targetURI :"); err != nil {
					return err
				}

				if _, err := c.SendLine("git@github.com:tektoncd/pipeline.git"); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				// check if the resource is created
				res, err := cs.Pipeline.Tekton().PipelineResources("namespace").Get("cloudEvent-res", metav1.GetOptions{})
				if err != nil {
					return err
				}

				if res.Name != "cloudEvent-res" {
					return errors.New("unexpected error")
				}

				return nil
			},
		},
	}
	res := resOpts("namespace", cs)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_clusterResource_secure_password_text(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
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
			name: "clusterResource-securePasswordText",

			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("cluster-res"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("cloudEvent"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("cluster"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for name :"); err != nil {
					return err
				}

				if _, err := c.SendLine("some-cluster"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for url :"); err != nil {
					return err
				}

				if _, err := c.SendLine("https://10.10.10.10"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for username :"); err != nil {
					return err
				}

				if _, err := c.SendLine("user"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Is the cluster secure?"); err != nil {
					return err
				}

				if _, err := c.ExpectString("yes"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Which authentication technique you want to use?"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for password :"); err != nil {
					return err
				}

				if _, err := c.SendLine("abcd#@123"); err != nil {
					return err
				}

				if _, err := c.ExpectString("*********"); err != nil {
					return err
				}

				if _, err := c.ExpectString("How do you want to set cadata?"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Passing plain text as parameters"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for cadata :"); err != nil {
					return err
				}

				if _, err := c.SendLine("cadata"); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				// check if the resource is created
				res, err := cs.Pipeline.Tekton().PipelineResources("namespace").Get("cluster-res", metav1.GetOptions{})
				if err != nil {
					return err
				}

				if res.Name != "cluster-res" {
					return errors.New("unexpected error")
				}

				return nil
			},
		},
	}
	res := resOpts("namespace", cs)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_clusterResource_secure_token_text(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
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
			name: "clusterResource-secureTokenText",

			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("cluster-res"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("cloudEvent"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("cluster"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for name :"); err != nil {
					return err
				}

				if _, err := c.SendLine("some-cluster"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for url :"); err != nil {
					return err
				}

				if _, err := c.SendLine("https://10.10.10.10"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for username :"); err != nil {
					return err
				}

				if _, err := c.SendLine("user"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Is the cluster secure?"); err != nil {
					return err
				}

				if _, err := c.ExpectString("yes"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Which authentication technique you want to use?"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("token"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("How do you want to set cluster credentials?"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Passing plain text as parameters"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for token :"); err != nil {
					return err
				}

				if _, err := c.SendLine("token"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for cadata :"); err != nil {
					return err
				}

				if _, err := c.SendLine("cadata"); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				// check if the resource is created
				res, err := cs.Pipeline.Tekton().PipelineResources("namespace").Get("cluster-res", metav1.GetOptions{})
				if err != nil {
					return err
				}

				if res.Name != "cluster-res" {
					return errors.New("unexpected error")
				}

				return nil
			},
		},
	}
	res := resOpts("namespace", cs)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_gitResource(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
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
			name: "gitResource",

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

				if _, err := c.ExpectString("cloudEvent"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("cluster"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
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
				res, err := cs.Pipeline.Tekton().PipelineResources("namespace").Get("git-res", metav1.GetOptions{})
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
		t.Run(test.name, func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_imageResource(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
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
			name: "imageResource",

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

				if _, err := c.ExpectString("cloudEvent"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("cluster"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
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
				res, err := cs.Pipeline.Tekton().PipelineResources("namespace").Get("image-res", metav1.GetOptions{})
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
		t.Run(test.name, func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_clusterResource_secure_password_secret(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
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
			name: "clusterResource-securePasswordSecrets",

			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("cluster-res"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("cloudEvent"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("cluster"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for name :"); err != nil {
					return err
				}

				if _, err := c.SendLine("some-cluster"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for url :"); err != nil {
					return err
				}

				if _, err := c.SendLine("https://10.10.10.10"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for username :"); err != nil {
					return err
				}

				if _, err := c.SendLine("user"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Is the cluster secure?"); err != nil {
					return err
				}

				if _, err := c.ExpectString("yes"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Which authentication technique you want to use?"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for password :"); err != nil {
					return err
				}

				if _, err := c.SendLine("abcd#@123"); err != nil {
					return err
				}

				if _, err := c.ExpectString("*********"); err != nil {
					return err
				}

				if _, err := c.ExpectString("How do you want to set cadata?"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Passing plain text as parameters"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Using existing kubernetes secrets"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Secret Key for cadata :"); err != nil {
					return err
				}

				if _, err := c.SendLine("cadataKey"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Secret Name for cadata :"); err != nil {
					return err
				}

				if _, err := c.SendLine("cadataName"); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				// check if the resource is created
				res, err := cs.Pipeline.Tekton().PipelineResources("namespace").Get("cluster-res", metav1.GetOptions{})
				if err != nil {
					return err
				}

				if res.Name != "cluster-res" {
					return errors.New("unexpected error")
				}

				return nil
			},
		},
	}
	res := resOpts("namespace", cs)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_clusterResource_secure_token_secret(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
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
			name: "clusterResource-secureTokenSecrets",

			procedure: func(c *goexpect.Console) error {
				if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
					return err
				}

				if _, err := c.SendLine("cluster-res"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Select a resource type to create :"); err != nil {
					return err
				}

				if _, err := c.ExpectString("cloudEvent"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("cluster"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for name :"); err != nil {
					return err
				}

				if _, err := c.SendLine("some-cluster"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for url :"); err != nil {
					return err
				}

				if _, err := c.SendLine("https://10.10.10.10"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Enter a value for username :"); err != nil {
					return err
				}

				if _, err := c.SendLine("user"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Is the cluster secure?"); err != nil {
					return err
				}

				if _, err := c.ExpectString("yes"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Which authentication technique you want to use?"); err != nil {
					return err
				}

				if _, err := c.ExpectString("password"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("token"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("How do you want to set cluster credentials?"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Using existing kubernetes secrets"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
					return err
				}

				if _, err := c.ExpectString("Secret Key for token :"); err != nil {
					return err
				}

				if _, err := c.SendLine("tokenKey"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Secret Name for token :"); err != nil {
					return err
				}

				if _, err := c.SendLine("tokenName"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Secret Key for cadata :"); err != nil {
					return err
				}

				if _, err := c.SendLine("cadataKey"); err != nil {
					return err
				}

				if _, err := c.ExpectString("Secret Name for cadata :"); err != nil {
					return err
				}

				if _, err := c.SendLine("cadataName"); err != nil {
					return err
				}

				if _, err := c.ExpectEOF(); err != nil {
					return err
				}

				// check if the resource is created
				res, err := cs.Pipeline.Tekton().PipelineResources("namespace").Get("cluster-res", metav1.GetOptions{})
				if err != nil {
					return err
				}

				if res.Name != "cluster-res" {
					return errors.New("unexpected error")
				}

				return nil
			},
		},
	}
	res := resOpts("namespace", cs)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_pullRequestResource(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
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
			name: "pullRequestResource",

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

				if _, err := c.ExpectString("cloudEvent"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("cluster"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
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
				res, err := cs.Pipeline.Tekton().PipelineResources("namespace").Get("pullRequest-res", metav1.GetOptions{})
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
		t.Run(test.name, func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_gcsStorageResource(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
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
			name: "gcsStorageResource",

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

				if _, err := c.ExpectString("cloudEvent"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("cluster"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
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
				res, err := cs.Pipeline.Tekton().PipelineResources("namespace").Get("storage-res", metav1.GetOptions{})
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
		t.Run(test.name, func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func TestPipelineResource_create_buildGCSstorageResource(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
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
			name: "buildGCSstorageResource",

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

				if _, err := c.ExpectString("cloudEvent"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
					return err
				}

				if _, err := c.ExpectString("cluster"); err != nil {
					return err
				}

				if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
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
				res, err := cs.Pipeline.Tekton().PipelineResources("namespace").Get("storage-res", metav1.GetOptions{})
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
		t.Run(test.name, func(t *testing.T) {
			res.RunPromptTest(t, test)
		})
	}
}

func Test_Pipeline_Resource_Create(t *testing.T) {

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seeds := make([]pipelinetest.Clients, 0)
	for i := 0; i < 1; i++ {
		cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
		seeds = append(seeds, cs)
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
			name:        "Create pipeline resource successfully",
			command:     []string{"create", "--from", "./testdata/pipelineresource.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "PipelineResource created: test-resource\n",
		},
		{
			name:        "Filename does not exist",
			command:     []string{"create", "-f", "./testdata/notexist.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "open ./testdata/notexist.yaml: no such file or directory",
		},
		{
			name:        "Unsupported file type",
			command:     []string{"create", "-f", "./testdata/pipelineresource.txt", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "inavlid file format for ./testdata/pipelineresource.txt: .yaml or .yml file extension and format required",
		},
		{
			name:        "Mismatched resource file",
			command:     []string{"create", "-f", "./testdata/pipelinerun.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "provided kind PipelineRun instead of kind PipelineResource",
		},
		{
			name:        "Existing pipeline",
			command:     []string{"create", "-f", "./testdata/pipelineresource.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   true,
			want:        "failed to create pipeline resource \"test-resource\": pipelineresources.tekton.dev \"test-resource\" already exists",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline, Kube: tp.input.Kube}
			resource := Command(p)

			if tp.inputStream != nil {
				resource.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(resource, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("Error expected here")
				} else {
					test.AssertOutput(t, tp.want, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}

func resOpts(ns string, cs pipelinetest.Clients) *Resource {

	p := test.Params{
		Kube:   cs.Kube,
		Tekton: cs.Pipeline,
	}
	out := new(bytes.Buffer)
	p.SetNamespace(ns)
	resOp := Resource{
		Params: &p,
		stream: &cli.Stream{Out: out, Err: out},
	}

	return &resOp
}
