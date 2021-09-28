package bundle

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestWriteAndRead(t *testing.T) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "write-and-read-test")
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
	remoteContents, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	if string(remoteContents) != "some-contents" {
		t.Errorf("Expected image contents to be \"some-contents\" but found %s", string(remoteContents))
	}
}
