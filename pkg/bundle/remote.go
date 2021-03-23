package bundle

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	remoteimg "github.com/google/go-containerregistry/pkg/v1/remote"
	homedir "github.com/mitchellh/go-homedir"
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

// Read looks up an image from a remote registry and fetches layers from a local cache if we have seen it before.
func Read(ref name.Reference, cacheOptions *CacheOptions, opts ...remoteimg.Option) (v1.Image, error) {
	img, err := remoteimg.Image(ref, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}

	if cacheOptions.noCache {
		return img, nil
	}

	// Construct a new cache and wrap this image in that.
	dir, err := homedir.Expand(cacheOptions.cacheDir)
	if err != nil {
		return nil, err
	}
	fsCache := cache.NewFilesystemCache(dir)
	return cache.Image(img, fsCache), nil
}
