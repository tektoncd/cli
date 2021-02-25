package bundle

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	remoteimg "github.com/google/go-containerregistry/pkg/v1/remote"
)

// Write will publish an OCI image to a remote registry using the provided options and reference.
func Write(img v1.Image, ref name.Reference, opts ...remoteimg.Option) (string, error) {
	if err := remoteimg.Write(ref, img, opts...); err != nil {
		return "", fmt.Errorf("could not push image to registry as %q: %w", ref.String(), err)
	}

	digest, err := img.Digest()
	if err != nil {
		return "", fmt.Errorf("could not read image digest: %w", err)
	}

	return ref.Context().Digest(digest.String()).String(), nil
}
