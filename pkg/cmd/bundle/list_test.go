package bundle

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/tektoncd/cli/pkg/bundle"
	"github.com/tektoncd/cli/pkg/test"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"gotest.tools/assert"
)

const (
	examplePullTask = `apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  creationTimestamp: null
  name: foobar
spec:
  params:
  - name: someparam
`
	examplePullPipeline = `apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  creationTimestamp: null
  name: foobar
spec:
  params:
  - name: someparam
`
)

func TestListCommand(t *testing.T) {
	testcases := []struct {
		name           string
		additionalArgs []string
		format         string
		expectedStdout string
		expectedErr    string
	}{
		{
			name:           "no-format",
			format:         "",
			expectedStdout: "task.tekton.dev/foobar\npipeline.tekton.dev/foobar\n",
		}, {
			name:           "name-format",
			format:         "name",
			expectedStdout: "task.tekton.dev/foobar\npipeline.tekton.dev/foobar\n",
		}, {
			name:           "yaml-format",
			format:         "yaml",
			expectedStdout: examplePullTask + examplePullPipeline,
		}, {
			name:           "specify-kind-task",
			format:         "name",
			expectedStdout: "task.tekton.dev/foobar\n",
			additionalArgs: []string{"Task"},
		}, {
			name:           "specify-kind-task-lowercase-plural",
			format:         "name",
			expectedStdout: "task.tekton.dev/foobar\n",
			additionalArgs: []string{"tasks"},
		}, {
			name:           "specify-kind-task-lowercase-singular",
			format:         "name",
			expectedStdout: "task.tekton.dev/foobar\n",
			additionalArgs: []string{"task"},
		}, {
			name:           "specify-kind-pipeline",
			format:         "name",
			expectedStdout: "pipeline.tekton.dev/foobar\n",
			additionalArgs: []string{"Pipeline"},
		}, {
			name:           "specify-kind-name-dne",
			format:         "name",
			additionalArgs: []string{"Pipeline", "does-not-exist"},
			expectedErr:    `no objects of kind "pipeline" named "does-not-exist" found in img`,
		}, {
			name:           "specify-kind-name",
			format:         "name",
			expectedStdout: "pipeline.tekton.dev/foobar\n",
			additionalArgs: []string{"Pipeline", "foobar"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(registry.New())
			defer s.Close()
			u, err := url.Parse(s.URL)
			if err != nil {
				t.Fatal(err)
			}

			ref := fmt.Sprintf("%s/test-img-namespace/%s:1.0", u.Host, tc.name)
			parsedRef, err := name.ParseReference(ref)
			if err != nil {
				t.Fatal(err)
			}

			img, err := bundle.BuildTektonBundle([]string{examplePullTask, examplePullPipeline}, &bytes.Buffer{})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := bundle.Write(img, parsedRef); err != nil {
				t.Fatal(err)
			}

			cs, _ := test.SeedTestData(t, pipelinetest.Data{})
			tdc := testDynamic.Options{}
			dc, _ := tdc.Client()

			p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
			task := Command(p)

			args := []string{"list", ref}
			args = append(args, tc.additionalArgs...)
			if tc.format != "" {
				args = append(args, "-o", tc.format)
			}

			output, err := test.ExecuteCommand(task, args...)
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
				return
			} else if err != nil {
				t.Error(err)
			}

			test.AssertOutput(t, tc.expectedStdout, output)
		})
	}
}
