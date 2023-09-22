package bundle

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/tektoncd/cli/pkg/bundle"
	"github.com/tektoncd/cli/pkg/cli"
	tkremote "github.com/tektoncd/pipeline/pkg/remote/oci"
	"sigs.k8s.io/yaml"
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
		name                string
		files               map[string]string
		stdin               string
		annotations         []string
		expectedContents    map[string]expected
		expectedAnnotations map[string]string
		ctime               time.Time
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
		{
			name: "with-annotations",
			files: map[string]string{
				"simple.yaml": exampleTask,
			},
			annotations:      []string{"org.opencontainers.image.license=Apache-2.0", "org.opencontainers.image.url = https://example.org"},
			expectedContents: map[string]expected{exampleTaskExpected.name: exampleTaskExpected},
			expectedAnnotations: map[string]string{
				"org.opencontainers.image.license": "Apache-2.0",
				"org.opencontainers.image.url":     "https://example.org",
			},
		},
		{
			name: "with-ctime",
			files: map[string]string{
				"simple.yaml": exampleTask,
			},
			expectedContents: map[string]expected{exampleTaskExpected.name: exampleTaskExpected},
			ctime:            time.Now(),
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

			testDir, err := os.MkdirTemp(os.TempDir(), tc.name+"-")
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
				if err := os.WriteFile(filename, []byte(contents), os.ModePerm); err != nil {
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
				annotationParams:   tc.annotations,
				remoteOptions:      bundle.RemoteOptions{},
				ctime:              tc.ctime,
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

			config, err := img.ConfigFile()
			if err != nil {
				t.Fatal(err)
			}
			if config.Created.Time.Unix() != tc.ctime.Unix() {
				t.Errorf("Expected created time to be %s, but it was %s", tc.ctime, config.Created.Time)
			}

			layers, err := img.Layers()
			if err != nil {
				t.Fatal(err)
			}

			if len(manifest.Layers) != len(tc.expectedContents) {
				t.Errorf("Expected %d layers but found %d", len(tc.expectedContents), len(manifest.Layers))
			}

			if len(manifest.Annotations) != len(tc.expectedAnnotations) || fmt.Sprint(manifest.Annotations) != fmt.Sprint(tc.expectedAnnotations) {
				t.Errorf("Requested annotations were not set wanted: %s, got %s", tc.expectedAnnotations, manifest.Annotations)
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

func TestParseTime(t *testing.T) {
	now = func() time.Time {
		return time.Date(2023, 9, 22, 1, 2, 3, 0, time.UTC)
	}

	cases := []struct {
		name     string
		given    string
		err      string
		expected time.Time
	}{
		{name: "now", expected: now()},
		{name: "date", given: "2023-09-22", expected: time.Date(2023, 9, 22, 0, 0, 0, 0, time.UTC)},
		{name: "date and time", given: "2023-09-22T01:02:03", expected: time.Date(2023, 9, 22, 1, 2, 3, 0, time.UTC)},
		{name: "utc with fraction", given: "2023-09-22T01:02:03.45Z", expected: time.Date(2023, 9, 22, 1, 2, 3, 45, time.UTC)},
		{name: "full", given: "2023-09-22T01:02:03+04:30", expected: time.Date(2023, 9, 22, 1, 2, 3, 0, time.FixedZone("", 16200))},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseTime(c.given)

			if err != nil {
				if err.Error() != c.err {
					t.Errorf("expected error %q, got %q", c.err, err)
				} else {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if got.Unix() != c.expected.Unix() {
				t.Errorf("expected parsed time to be %s, got %s", c.expected, got)
			}
		})
	}
}
