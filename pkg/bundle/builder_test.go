package bundle

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned/scheme"
	tkremote "github.com/tektoncd/pipeline/pkg/remote/oci"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var threeTasks = []string{
	`apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: task1
spec:
  description: task1
`,
	`apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: task2
spec:
  description: task2
`,
	`apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: task3
spec:
  description: task3
`,
}

func init() {
	// shuffle the test tasks
	sort.Slice(threeTasks, func(_, _ int) bool {
		return rand.Intn(2) == 0
	})
}

// Note, that for this test we are only using one object type to precisely test the image contents. The
// #TestDecodeFromRaw tests the general parsing logic.
func TestBuildTektonBundle(t *testing.T) {
	task := v1beta1.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Task",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "foo"},
		Spec:       v1beta1.TaskSpec{Description: "foobar"},
	}

	raw, err := yaml.Marshal(task)
	if err != nil {
		t.Error(err)
		return
	}

	annotations := map[string]string{"org.opencontainers.image.license": "Apache-2.0", "org.opencontainers.image.url": "https://example.org"}
	labels := map[string]string{"version": "1.0", "quay.expires-after": "7d"}
	img, err := BuildTektonBundle([]string{string(raw)}, annotations, labels, time.Now(), &bytes.Buffer{})
	if err != nil {
		t.Error(err)
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		t.Error(err)
		return
	}
	if cfg.Created.IsZero() {
		t.Error("Created time of image was not set")
	}
	if !cmp.Equal(cfg.Config.Labels, labels) {
		t.Errorf("Expected labels were not set got: %+v", cfg.Config.Labels)
	}

	manifest, err := img.Manifest()
	if err != nil {
		t.Error(err)
		return
	}

	ann := manifest.Annotations
	if len(ann) != len(annotations) || fmt.Sprint(ann) != fmt.Sprint(annotations) {
		t.Errorf("Requested annotations were not set wanted: %s, got %s", annotations, ann)
	}

	if len(manifest.Layers) != 1 {
		t.Errorf("Unexpected number of layers %d", len(manifest.Layers))
	}

	l := manifest.Layers[0]
	if apiVersion, ok := l.Annotations[tkremote.APIVersionAnnotation]; !ok || apiVersion != "v1beta1" {
		t.Errorf("Did not receive expected APIVersion v1beta1. Found %s", apiVersion)
	}
	if kind, ok := l.Annotations[tkremote.KindAnnotation]; !ok || kind != "task" {
		t.Errorf("Did not receive expected Kind Task. Found %s", kind)
	}
	if name, ok := l.Annotations[tkremote.TitleAnnotation]; !ok || name != "foo" {
		t.Errorf("Did not receive expected metadata.name \"foo\". Found %s", name)
	}

	layers, err := img.Layers()
	if err != nil {
		t.Error(err)
		return
	}

	if len(layers) != 1 {
		t.Errorf("Unexpected number of layers %d", len(layers))
	}

	rc, err := layers[0].Uncompressed()
	if err != nil {
		t.Errorf("Failed to read image layer: %v", err)
	}
	defer rc.Close()

	// If the user bundled this up as a tar file then we need to untar it.
	treader := tar.NewReader(rc)
	if _, err := treader.Next(); err != nil {
		t.Errorf("layer is not a tarball")
	}

	limited := io.LimitReader(treader, MaxLayerSize+1)
	contents, err := io.ReadAll(limited)
	if err != nil && err != io.EOF {
		t.Errorf("failed to read tar bundle: %v", err)
	}

	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(contents, nil, nil)
	if err != nil {
		t.Errorf("failed to decode layer contents to a Tekton object: %v", err)
	}

	if diff := cmp.Diff(obj, &task); diff != "" {
		t.Error(diff)
	}
}

func TestBadObj(t *testing.T) {
	task := v1beta1.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Task",
		},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       v1beta1.TaskSpec{Description: "foobar"},
	}

	raw, err := yaml.Marshal(task)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = BuildTektonBundle([]string{string(raw)}, nil, nil, time.Now(), &bytes.Buffer{})
	noNameErr := errors.New("kubernetes resources should have a name")
	if err == nil {
		t.Errorf("expected error: %v", noNameErr)
	}
}

func TestLessThenMaxBundle(t *testing.T) {
	task := v1beta1.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Task",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "foo"},
		Spec:       v1beta1.TaskSpec{Description: "foobar"},
	}

	raw, err := yaml.Marshal(task)
	if err != nil {
		t.Error(err)
		return
	}
	// no error for less then max
	_, err = BuildTektonBundle([]string{string(raw)}, nil, nil, time.Now(), &bytes.Buffer{})
	if err != nil {
		t.Error(err)
	}

}

func TestJustEnoughBundleSize(t *testing.T) {
	var justEnoughObj []string
	for i := 0; i == tkremote.MaximumBundleObjects; i++ {
		name := fmt.Sprintf("%d-task", i)
		task := v1beta1.Task{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "Task",
			},
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       v1beta1.TaskSpec{Description: "foobar"},
		}

		raw, err := yaml.Marshal(task)
		if err != nil {
			t.Error(err)
			return
		}
		justEnoughObj = append(justEnoughObj, string(raw))
	}
	// no error for the max
	_, err := BuildTektonBundle(justEnoughObj, nil, nil, time.Now(), &bytes.Buffer{})
	if err != nil {
		t.Error(err)
	}
}

func TestTooManyInBundle(t *testing.T) {
	toManyObjErr := fmt.Sprintf("contained more than the maximum %d allow objects", tkremote.MaximumBundleObjects)
	var toMuchObj []string
	for i := 0; i <= tkremote.MaximumBundleObjects; i++ {
		name := fmt.Sprintf("%d-task", i)
		task := v1beta1.Task{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "Task",
			},
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       v1beta1.TaskSpec{Description: "foobar"},
		}

		raw, err := yaml.Marshal(task)
		if err != nil {
			t.Error(err)
			return
		}
		toMuchObj = append(toMuchObj, string(raw))
	}

	// expect error when we hit the max
	_, err := BuildTektonBundle(toMuchObj, nil, nil, time.Now(), &bytes.Buffer{})
	if err == nil {
		t.Errorf("expected error: %v", toManyObjErr)
	}
}

func TestDeterministicLayers(t *testing.T) {
	img, err := BuildTektonBundle(threeTasks, nil, nil, time.Now(), &bytes.Buffer{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	layers, err := img.Layers()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if l := len(layers); l != 3 {
		t.Errorf("expecting 3 layers got: %d", l)
	}

	compare := func(n int, expected string) {
		digest, err := layers[n].Digest()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		got := digest.String()
		if got != expected {
			t.Errorf("unexpected digest for layer %d: %s, expecting %s", n, got, expected)
		}
	}

	compare(0, "sha256:05ea10ce5b8dbbd638d40b7e09008f1553e37d7146fa7440443f41b54565990c")
	compare(1, "sha256:8163f84b890c6d57307a766098bb24eb3ea93328408cd05a59176d21c5e03f98")
	compare(2, "sha256:f3b8e632fdd3769d9f1c79fe17ddf4bece7f34a605276364a22fcee1d840a9a3")
}

func TestDeterministicManifest(t *testing.T) {
	img, err := BuildTektonBundle(threeTasks, nil, nil, time.Time{}, &bytes.Buffer{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	digest, err := img.Digest()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if expected, got := "sha256:6f74ecf3bf60218d26087e1f2f1aca3f7501cd89d43d88c6de1b20738b89b842", digest.String(); expected != got {
		t.Errorf("unexpected image digest: %s, expecting %s", got, expected)
	}
}
