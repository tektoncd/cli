package bundle

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func TestWriteAndRead(t *testing.T) {
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

	imgRef, err := name.ParseReference(fmt.Sprintf("%s/testimg/myimg:1.0", u.Host))
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

	actualImg, err := Read(actualRef)
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

	// There should be a remote and cached version of this layer. Ensure the contents match.
	reader, _ := layers[0].Uncompressed()
	remoteContents, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	if string(remoteContents) != "some-contents" {
		t.Errorf("Expected image contents to be \"some-contents\" but found %s", string(remoteContents))
	}

	reader, _ = layers[1].Uncompressed()
	cachedContents, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	if string(cachedContents) != "some-contents" {
		t.Errorf("Expected image contents to be \"some-contents\" but found %s", string(cachedContents))
	}
}
