package bundle

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	tkremote "github.com/tektoncd/pipeline/pkg/remote/oci"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// BuildTektonBundle will return a complete OCI Image usable as a Tekton Bundle built by parsing, decoding, and
// compressing the provided contents as Tekton objects.
func BuildTektonBundle(contents []string, log io.Writer) (v1.Image, error) {
	img := empty.Image

	if len(contents) > tkremote.MaximumBundleObjects {
		return nil, fmt.Errorf("bundle contains more than the maximum %d allow objects", tkremote.MaximumBundleObjects)
	}

	fmt.Fprint(log, "Creating Tekton Bundle:\n")

	// For each block of input, attempt to parse all of the YAML/JSON objects as Tekton objects and compress them into
	// the OCI image as a tar layer.
	for _, content := range contents {
		if err := decodeObjects(content, func(gvr *schema.GroupVersionKind, element runtime.Object, raw []byte) error {
			name, err := getObjectName(element)
			if err != nil {
				return err
			}

			// Tar up this object before writing it to the layer.
			var tarbundle bytes.Buffer
			writer := tar.NewWriter(&tarbundle)
			if err := writer.WriteHeader(&tar.Header{
				Name:     name,
				Mode:     0600,
				Size:     int64(len(raw)),
				Typeflag: tar.TypeReg,
			}); err != nil {
				return err
			}
			if _, err := writer.Write(raw); err != nil {
				return err
			}
			if err := writer.Close(); err != nil {
				return err
			}

			// nolint: staticcheck
			l, err := tarball.LayerFromReader(&tarbundle)
			if err != nil {
				return err
			}

			// Add this layer to the image with all of the required annotations.
			img, err = mutate.Append(img, mutate.Addendum{
				Layer: l,
				Annotations: map[string]string{
					tkremote.APIVersionAnnotation: gvr.Version,
					tkremote.KindAnnotation:       strings.ToLower(gvr.Kind),
					tkremote.TitleAnnotation:      name,
				},
			})
			if err != nil {
				return fmt.Errorf("failed to add %q to Tekton Bundle: %w", string(raw), err)
			}

			fmt.Fprintf(log, "\t- Added %s: %s to image\n", gvr.Kind, name)

			return nil
		}); err != nil {
			return nil, err
		}
	}

	// Set created time for bundle image
	img, err := mutate.CreatedAt(img, v1.Time{Time: time.Now()})
	if err != nil {
		return nil, fmt.Errorf("failed to add created time to image: %w", err)
	}

	return img, nil
}

// Return the ObjectMetadata.Name field which every resource should have.
func getObjectName(obj runtime.Object) (string, error) {
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return "", errors.New("object is not a registered kubernetes resource")
	}
	name := metaObj.GetName()
	if name == "" {
		return "", errors.New("kubernetes resources should have a name")
	}
	return name, nil
}
