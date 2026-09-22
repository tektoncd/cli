package bundle

import (
	"bytes"
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestWriteAndRead(t *testing.T) {
	tempDir, err := os.MkdirTemp(os.TempDir(), "write-and-read-test")
	if err != nil {
		t.Fatal(err)
	}

	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}

	img := empty.Image
	// nolint: staticcheck
	testLayer, err := tarball.LayerFromReader(bytes.NewReader([]byte("some-contents")))
	if err != nil {
		t.Fatal(err)
	}

	img, err = mutate.Append(img, mutate.Addendum{Layer: testLayer})
	if err != nil {
		t.Fatal(err)
	}

	imgName := rand.String(6)
	imgRef, err := name.ParseReference(fmt.Sprintf("%s/testimg/%s:1.0", u.Host, imgName))
	if err != nil {
		t.Fatal(err)
	}

	digest, err := Write(img, imgRef)
	if err != nil {
		t.Fatal(err)
	}

	actualRef, err := name.ParseReference(digest)
	if err != nil {
		t.Fatal(err)
	}

	cacheOptions := CacheOptions{
		cacheDir: tempDir,
		noCache:  false,
	}
	actualImg, err := Read(actualRef, &cacheOptions)
	if err != nil {
		t.Fatal(err)
	}

	manifest, err := actualImg.Manifest()
	if err != nil {
		t.Fatal(err)
	}

	if len(manifest.Layers) != 1 {
		t.Error("Image does not contain expected number of layers")
	}

	layers, err := actualImg.Layers()
	if err != nil {
		t.Fatal(err)
	}

	reader, _ := layers[0].Uncompressed()
	remoteContents, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	if string(remoteContents) != "some-contents" {
		t.Errorf("Expected image contents to be \"some-contents\" but found %s", string(remoteContents))
	}

	t.Run("default cache directory selection", func(t *testing.T) {
		testcases := []struct {
			name      string
			useLegacy bool
		}{
			{name: "XDG cache"},
			{name: "legacy Tekton directory", useLegacy: true},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
				xdgCacheHome := filepath.Join(home, "xdg-cache")
				t.Setenv("XDG_CACHE_HOME", xdgCacheHome)

				wantCacheDir := filepath.Join(xdgCacheHome, "tkn", "bundles")
				if tc.useLegacy {
					legacyDir := filepath.Join(home, ".tekton")
					if err := os.Mkdir(legacyDir, 0o700); err != nil {
						t.Fatal(err)
					}
					wantCacheDir = filepath.Join(legacyDir, "bundles")
				}

				options := CacheOptions{}
				cachedImg, err := Read(actualRef, &options)
				if err != nil {
					t.Fatal(err)
				}
				cachedLayers, err := cachedImg.Layers()
				if err != nil {
					t.Fatal(err)
				}
				cachedContents, err := cachedLayers[0].Uncompressed()
				if err != nil {
					t.Fatal(err)
				}
				if _, err := io.Copy(io.Discard, cachedContents); err != nil {
					t.Fatal(err)
				}
				if err := cachedContents.Close(); err != nil {
					t.Fatal(err)
				}

				entries, err := os.ReadDir(wantCacheDir)
				if err != nil {
					t.Fatal(err)
				}
				if len(entries) == 0 {
					t.Fatalf("cache directory %q is empty", wantCacheDir)
				}
				if tc.useLegacy {
					xdgBundleCache := filepath.Join(xdgCacheHome, "tkn", "bundles")
					if _, err := os.Stat(xdgBundleCache); err == nil || !os.IsNotExist(err) {
						t.Fatalf("XDG cache %q should not be used, stat error = %v", xdgBundleCache, err)
					}
				}
			})
		}
	})

	t.Run("no-cache does not resolve cache directory", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_CACHE_HOME", "relative/xdg-cache")
		noCacheOptions := CacheOptions{noCache: true}
		if _, err := Read(actualRef, &noCacheOptions); err != nil {
			t.Fatal(err)
		}
	})
}
