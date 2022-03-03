package bundle

import (
	"archive/tar"
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tkremote "github.com/tektoncd/pipeline/pkg/remote/oci"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func writeTarLayer(t *testing.T, img v1.Image, obj runtime.Object) (v1.Image, error) {
	t.Helper()

	data, err := yaml.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("error serializing object: %w", err)
	}

	name, err := getObjectName(obj)
	if err != nil {
		return nil, err
	}

	// Compress the data into a tarball.
	var tarbundle bytes.Buffer
	writer := tar.NewWriter(&tarbundle)
	if err := writer.WriteHeader(&tar.Header{
		Name:     name,
		Mode:     0600,
		Size:     int64(len(data)),
		Typeflag: tar.TypeReg,
	}); err != nil {
		return nil, err
	}
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	// nolint: staticcheck
	layer, err := tarball.LayerFromReader(&tarbundle)
	if err != nil {
		return nil, fmt.Errorf("unexpected error adding layer to image %w", err)
	}

	return mutate.Append(img, mutate.Addendum{
		Layer: layer,
		Annotations: map[string]string{
			tkremote.TitleAnnotation:      name,
			tkremote.KindAnnotation:       strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind),
			tkremote.APIVersionAnnotation: obj.GetObjectKind().GroupVersionKind().Version,
		},
	})
}

func TestReader(t *testing.T) {
	task1 := v1beta1.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Task",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "foo-1"},
		Spec:       v1beta1.TaskSpec{Description: "foobar"},
	}
	task2 := v1beta1.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Task",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "foo-2"},
		Spec:       v1beta1.TaskSpec{Description: "foobar"},
	}
	pipeline := v1beta1.Pipeline{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Pipeline",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "foo"},
		Spec:       v1beta1.PipelineSpec{Description: "foobar"},
	}

	var err error
	img := empty.Image
	if img, err = writeTarLayer(t, img, &task1); err != nil {
		t.Fatal(err)
	}
	if img, err = writeTarLayer(t, img, &task2); err != nil {
		t.Fatal(err)
	}
	if img, err = writeTarLayer(t, img, &pipeline); err != nil {
		t.Fatal(err)
	}

	// Expect list to return all the elements.
	if err := List(img, func(_, _, _ string, element runtime.Object, _ []byte) {
		test.Contains(t, []runtime.Object{&task1, &task2, &pipeline}, element)
	}); err != nil {
		t.Error(err)
	}

	// Expect ListKind to return the two tasks.
	if err := ListKind(img, strings.ToLower(task1.GroupVersionKind().Kind), func(_, _, _ string, element runtime.Object, _ []byte) {
		test.Contains(t, []runtime.Object{&task1, &task2}, element)
	}); err != nil {
		t.Error(err)
	}

	// Expect Get to return a single object.
	if err := Get(img, strings.ToLower(task1.GroupVersionKind().Kind), task1.GetName(), func(_, _, _ string, element runtime.Object, _ []byte) {
		test.Contains(t, []runtime.Object{&task1}, element)
	}); err != nil {
		t.Error(err)
	}
}
