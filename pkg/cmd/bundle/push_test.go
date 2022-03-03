package bundle

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/tektoncd/cli/pkg/bundle"
	"github.com/tektoncd/cli/pkg/cli"
	tkremote "github.com/tektoncd/pipeline/pkg/remote/oci"
)

type expected struct {
	name       string
	kind       string
	apiVersion string
	raw        string
}

const (
	exampleTask = `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: foobar
spec:
  params:
    - name: someparam
`
)

var (
	exampleTaskExpected = expected{
		name:       "foobar",
		kind:       "Task",
		apiVersion: "v1beta1",
		raw:        exampleTask,
	}
)

func TestPushCommand(t *testing.T) {
	testcases := []struct {
		name             string
		files            map[string]string
		stdin            string
		expectedContents map[string]expected
	}{
		{
			name: "single-input",
			files: map[string]string{
				"simple.yaml": exampleTask,
			},
			expectedContents: map[string]expected{exampleTaskExpected.name: exampleTaskExpected},
		},
		{
			name: "stdin-input",
			files: map[string]string{
				"-": "",
			},
			stdin:            exampleTask,
			expectedContents: map[string]expected{exampleTaskExpected.name: exampleTaskExpected},
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

			testDir, err := ioutil.TempDir(os.TempDir(), tc.name+"-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(testDir)

			var paths []string
			for name, contents := range tc.files {
				if name == "-" {
					paths = append(paths, name)
					continue
				}

				filename := path.Join(testDir, name)
				if err := ioutil.WriteFile(filename, []byte(contents), os.ModePerm); err != nil {
					t.Fatalf("failed to write file %s: %s", filename, err)
				}
				paths = append(paths, filename)
			}

			opts := pushOptions{
				cliparams: nil,
				stream: &cli.Stream{
					In:  bytes.NewBuffer([]byte(tc.stdin)),
					Out: &bytes.Buffer{},
					Err: &bytes.Buffer{},
				},
				bundleContentPaths: paths,
				remoteOptions:      bundle.RemoteOptions{},
			}
			if err := opts.Run([]string{ref}); err != nil {
				t.Errorf("Unexpected failure calling run: %v", err)
			}

			// Fetch and verify the image was published as expected.
			img, err := remote.Image(opts.ref)
			if err != nil {
				t.Fatal(err)
			}

			manifest, err := img.Manifest()
			if err != nil {
				t.Fatal(err)
			}

			layers, err := img.Layers()
			if err != nil {
				t.Fatal(err)
			}

			if len(manifest.Layers) != len(tc.expectedContents) {
				t.Errorf("Expected %d layers but found %d", len(tc.expectedContents), len(manifest.Layers))
			}

			for i, l := range manifest.Layers {
				title, ok := l.Annotations[tkremote.TitleAnnotation]
				if !ok {
					t.Errorf("layer %d did not have a title annotation", i)
				}

				layer, ok := tc.expectedContents[title]
				if !ok {
					t.Errorf("layer %d with title %s does not match an expected layer", i, title)
				}

				kind := normalizeKind(layer.kind)
				if l.Annotations[tkremote.KindAnnotation] != kind ||
					l.Annotations[tkremote.APIVersionAnnotation] != layer.apiVersion {
					t.Errorf("layer annotations (%s, %s) do not match expected (%s, %s)", l.Annotations[tkremote.KindAnnotation], l.Annotations[tkremote.APIVersionAnnotation], kind, layer.apiVersion)
				}

				actual := readTarLayer(t, layers[i])
				expected, err := yaml.YAMLToJSON([]byte(layer.raw))
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(actual, string(expected)); diff != "" {
					t.Error(diff)
				}
			}
		})
	}
}

func readTarLayer(t *testing.T, layer v1.Layer) string {
	t.Helper()

	rc, err := layer.Uncompressed()
	if err != nil {
		t.Errorf("Failed to read image layer: %v", err)
	}
	defer rc.Close()

	// If the user bundled this up as a tar file then we need to untar it.
	treader := tar.NewReader(rc)
	header, err := treader.Next()
	if err != nil {
		t.Errorf("layer is not a tarball")

	}

	contents := make([]byte, header.Size)
	if _, err := treader.Read(contents); err != nil && err != io.EOF {
		// We only allow 1 resource per layer so this tar bundle should have one and only one file.
		t.Errorf("failed to read tar bundle: %v", err)
	}
	return string(contents)
}
