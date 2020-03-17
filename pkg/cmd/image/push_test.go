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

package image

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1remote "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tb "github.com/tektoncd/pipeline/test/builder"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getTaskFromImage(image, task string) ([]byte, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, err
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, fmt.Errorf("Error pulling %q: %w", ref, err)
	}

	m, err := img.Manifest()
	if err != nil {
		return nil, err
	}
	ls, err := img.Layers()
	if err != nil {
		return nil, err
	}
	var layer v1remote.Layer
	for idx, l := range m.Layers {
		if l.Annotations["org.opencontainers.image.title"] == "task/"+task {
			layer = ls[idx]
			break
		}
	}
	if layer == nil {
		return nil, fmt.Errorf("Resource %s/%s not found", "task", task)
	}
	rc, err := layer.Uncompressed()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return ioutil.ReadAll(rc)
}

func TestPushImage(t *testing.T) {
	// For some test cases, we need an image registry to push up an image with a task to reference.
	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}

	simpleTask := tb.Task("simple", "default", tb.TaskSpec(tb.Step("hello", tb.StepCommand("echo 'Hello world'"))))
	simpleTask.TypeMeta = v1.TypeMeta{
		APIVersion: "tekton.dev/v1alpha1",
		Kind:       "Task",
	}
	dummyTask := tb.Task("dummy", "default", tb.TaskSpec(tb.Step("hello", tb.StepCommand("echo 'Hello world'"))))
	dummyTask.TypeMeta = v1.TypeMeta{
		APIVersion: "tekton.dev/v1alpha1",
		Kind:       "Task",
	}
	clusterTask := tb.ClusterTask("cluster", tb.ClusterTaskSpec(tb.Step("hello", tb.StepCommand("echo 'Hello world'"))))
	clusterTask.TypeMeta = v1.TypeMeta{
		APIVersion: "tekton.dev/v1alpha1",
		Kind:       "ClusterTask",
	}

	testParams := []struct {
		name      string
		command   []string
		objects   []v1alpha1.TaskInterface
		wantError string
	}{
		{
			name:      "task-image",
			command:   []string{"push", fmt.Sprintf("%s/repo/task-image", u.Host)},
			objects:   []v1alpha1.TaskInterface{simpleTask, dummyTask},
			wantError: "",
		},
		{
			name:      "cluster-task-image",
			command:   []string{"push", fmt.Sprintf("%s/repo/cluster-task-image", u.Host)},
			objects:   []v1alpha1.TaskInterface{clusterTask},
			wantError: "",
		},
		{
			name:      "no-reference",
			command:   []string{"push"},
			objects:   []v1alpha1.TaskInterface{},
			wantError: "no image specified",
		},
		{
			name:      "invalid-reference",
			command:   []string{"push", "fake.com/NotAValidReference"},
			objects:   []v1alpha1.TaskInterface{},
			wantError: "could not parse reference: fake.com/NotAValidReference",
		},
	}
	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{}
			c := Command(p)

			// Write each object to a file and add each file as a flag input to the command.
			for _, obj := range tp.objects {
				f, err := ioutil.TempFile("", tp.name+"-*.yaml")
				if err != nil {
					t.Errorf("unexpected error writing tmp file: %w", err)
				}
				defer f.Close()

				data, err := yaml.Marshal(obj)
				if err != nil {
					t.Errorf("unexpected error marshalling test data to bytes: %w", err)
				}

				if err := ioutil.WriteFile(f.Name(), data, 0777); err != nil {
					t.Errorf("failed to write tmp file: %w", err)
				}

				tp.command = append(tp.command, "-f", f.Name())
			}

			_, err := test.ExecuteCommand(c, tp.command...)
			switch {
			case tp.wantError != "" && err == nil:
				t.Errorf("command did not fail, expected %s", tp.wantError)
			case tp.wantError != "" && err != nil:
				test.AssertOutput(t, tp.wantError, err.Error())
			case tp.wantError == "" && err != nil:
				t.Errorf("command failed unexpectedly: %w", err)
			}

			// Fetch the image and check to see if all the contents are there.
			for _, obj := range tp.objects {
				expected, err := yaml.Marshal(obj)
				if err != nil {
					t.Errorf("unexpected error marshalling test data to bytes: %w", err)
				}

				actual, err := getTaskFromImage(tp.command[1], obj.TaskMetadata().Name)
				if err != nil {
					t.Errorf("unexpected error retrieving task %s: %w", obj.TaskMetadata().Name, err)
				}

				if diff := cmp.Diff(string(expected), string(actual)); diff != "" {
					t.Error(diff)
				}
			}
		})
	}
}
