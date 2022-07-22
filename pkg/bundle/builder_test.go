package bundle

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned/scheme"
	tkremote "github.com/tektoncd/pipeline/pkg/remote/oci"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	img, err := BuildTektonBundle([]string{string(raw)}, &bytes.Buffer{})
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

	manifest, err := img.Manifest()
	if err != nil {
		t.Error(err)
		return
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
	header, err := treader.Next()
	if err != nil {
		t.Errorf("layer is not a tarball")
	}

	contents := make([]byte, header.Size)
	if _, err := treader.Read(contents); err != nil && err != io.EOF {
		// We only allow 1 resource per layer so this tar bundle should have one and only one file.
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
	_, err = BuildTektonBundle([]string{string(raw)}, &bytes.Buffer{})
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
	_, err = BuildTektonBundle([]string{string(raw)}, &bytes.Buffer{})
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
	_, err := BuildTektonBundle(justEnoughObj, &bytes.Buffer{})
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
	_, err := BuildTektonBundle(toMuchObj, &bytes.Buffer{})
	if err == nil {
		t.Errorf("expected error: %v", toManyObjErr)
	}
}
