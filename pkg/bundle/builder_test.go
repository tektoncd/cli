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
	// Build the bundle twice with the same inputs (but shuffled order)
	img1, err := BuildTektonBundle(threeTasks, nil, nil, time.Now(), &bytes.Buffer{})
	if err != nil {
		t.Errorf("unexpected error building first bundle: %v", err)
	}

	img2, err := BuildTektonBundle(threeTasks, nil, nil, time.Now(), &bytes.Buffer{})
	if err != nil {
		t.Errorf("unexpected error building second bundle: %v", err)
	}

	layers1, err := img1.Layers()
	if err != nil {
		t.Errorf("unexpected error getting layers from first image: %v", err)
	}

	layers2, err := img2.Layers()
	if err != nil {
		t.Errorf("unexpected error getting layers from second image: %v", err)
	}

	if l := len(layers1); l != 3 {
		t.Errorf("expecting 3 layers in first image, got: %d", l)
	}

	if l := len(layers2); l != 3 {
		t.Errorf("expecting 3 layers in second image, got: %d", l)
	}

	// Verify that both builds produce identical layer digests (proving determinism)
	for i := 0; i < 3; i++ {
		digest1, err := layers1[i].Digest()
		if err != nil {
			t.Errorf("unexpected error getting digest for layer %d in first image: %v", i, err)
		}

		digest2, err := layers2[i].Digest()
		if err != nil {
			t.Errorf("unexpected error getting digest for layer %d in second image: %v", i, err)
		}

		if digest1.String() != digest2.String() {
			t.Errorf("layer %d digests don't match (non-deterministic build):\n  first:  %s\n  second: %s", i, digest1, digest2)
		}
	}
}

func TestDeterministicManifest(t *testing.T) {
	// Use zero time to ensure manifest digest is deterministic
	fixedTime := time.Time{}

	// Build the bundle twice with the same inputs (but shuffled order)
	img1, err := BuildTektonBundle(threeTasks, nil, nil, fixedTime, &bytes.Buffer{})
	if err != nil {
		t.Errorf("unexpected error building first bundle: %v", err)
	}

	img2, err := BuildTektonBundle(threeTasks, nil, nil, fixedTime, &bytes.Buffer{})
	if err != nil {
		t.Errorf("unexpected error building second bundle: %v", err)
	}

	digest1, err := img1.Digest()
	if err != nil {
		t.Errorf("unexpected error getting digest from first image: %v", err)
	}

	digest2, err := img2.Digest()
	if err != nil {
		t.Errorf("unexpected error getting digest from second image: %v", err)
	}

	if digest1.String() != digest2.String() {
		t.Errorf("image manifests don't match (non-deterministic build):\n  first:  %s\n  second: %s", digest1, digest2)
	}
}
