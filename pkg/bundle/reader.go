package bundle

import (
	"archive/tar"
	"fmt"
	"io"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned/scheme"
	tkremote "github.com/tektoncd/pipeline/pkg/remote/oci"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ObjectVisitor is an input function that callers of this file's methods can implement to act on the read contents of a
// Tekton bundle.
type ObjectVisitor func(gvk schema.GroupVersionKind, name string, element runtime.Object, raw []byte)

// List will call visitor for every single layer in the img.
func List(img v1.Image, visitor ObjectVisitor) error {
	manifest, err := img.Manifest()
	if err != nil {
		return err
	}

	layers, err := img.Layers()
	if err != nil {
		return err
	}

	layerMap := map[string]v1.Layer{}
	for _, l := range layers {
		digest, err := l.Digest()
		if err != nil {
			return err
		}
		layerMap[digest.String()] = l
	}

	for _, l := range manifest.Layers {
		rawLayer, ok := layerMap[l.Digest.String()]
		if !ok {
			return fmt.Errorf("no image layer with digest %s exists in the bundle", l.Digest.String())
		}

		contents, err := readTarLayer(rawLayer)
		if err != nil {
			return fmt.Errorf("failed to read layer %s: %w", l.Digest, err)
		}

		obj, gvk, err := scheme.Codecs.UniversalDeserializer().Decode(contents, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to decode layer %s to a Tekton object: %w", l.Digest, err)
		}
		visitor(*gvk, l.Annotations[tkremote.TitleAnnotation], obj, contents)
	}

	return nil
}

// ListKind is like #List but only returns elements of a single kind.
func ListKind(img v1.Image, kind string, visitor ObjectVisitor) error {
	listedItems := 0
	if err := List(img, func(gvk schema.GroupVersionKind, name string, element runtime.Object, raw []byte) {
		if gvk.Kind == kind {
			listedItems++
			visitor(gvk, name, element, raw)
		}
	}); err != nil {
		return err
	}

	if listedItems == 0 {
		return fmt.Errorf("no objects of kind %s found in img", kind)
	}
	return nil
}

// Get returns a single named element of a specific kind from the Tekton Bundle.
func Get(img v1.Image, kind, name string, visitor ObjectVisitor) error {
	objectFound := false
	if err := ListKind(img, kind, func(gvk schema.GroupVersionKind, foundName string, element runtime.Object, raw []byte) {
		if foundName == name {
			objectFound = true
			visitor(gvk, foundName, element, raw)
		}
	}); err != nil {
		return err
	}

	if !objectFound {
		return fmt.Errorf("no objects of kind %s named %s found in img", kind, name)
	}
	return nil
}

// readTarLayer is a helper function to read the contents of a tar'ed layer.
func readTarLayer(l v1.Layer) ([]byte, error) {
	rc, err := l.Uncompressed()
	if err != nil {
		return nil, fmt.Errorf("Failed to read image layer: %w", err)
	}
	defer rc.Close()

	// If the user bundled this up as a tar file then we need to untar it.
	treader := tar.NewReader(rc)
	header, err := treader.Next()
	if err != nil {
		return nil, fmt.Errorf("layer is not a tarball")
	}

	contents := make([]byte, header.Size)
	if _, err := treader.Read(contents); err != nil && err != io.EOF {
		// We only allow 1 resource per layer so this tar bundle should have one and only one file.
		return nil, fmt.Errorf("failed to read tar bundle: %w", err)
	}
	return contents, nil
}
