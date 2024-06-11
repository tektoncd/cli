/*
Copyright 2020 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package simple

import (
	"context"
	"fmt"

	"github.com/sigstore/sigstore/pkg/signature/payload"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/config"

	"github.com/google/go-containerregistry/pkg/name"
)

const (
	PayloadTypeSimpleSigning = formats.PayloadTypeSimpleSigning
)

func init() {
	formats.RegisterPayloader(PayloadTypeSimpleSigning, NewFormatter)
}

// SimpleSigning is a formatter that uses the RedHat simple signing format
// https://www.redhat.com/en/blog/container-image-signing
type SimpleSigning struct{}

type SimpleContainerImage payload.SimpleContainerImage

// CreatePayload implements the Payloader interface.
func (i *SimpleSigning) CreatePayload(ctx context.Context, obj interface{}) (interface{}, error) {
	switch v := obj.(type) {
	case name.Digest:
		format := NewSimpleStruct(v)
		return format, nil
	default:
		return nil, fmt.Errorf("unsupported type %s", v)
	}
}

func (i *SimpleSigning) Wrap() bool {
	return false
}

func NewFormatter(config.Config) (formats.Payloader, error) {
	return &SimpleSigning{}, nil
}

func NewSimpleStruct(img name.Digest) SimpleContainerImage {
	cosign := payload.Cosign{Image: img}
	return SimpleContainerImage(cosign.SimpleContainerImage())
}

func (i SimpleContainerImage) ImageName() string {
	return fmt.Sprintf("%s@%s", i.Critical.Identity.DockerReference, i.Critical.Image.DockerManifestDigest)
}

func (i *SimpleSigning) Type() config.PayloadType {
	return formats.PayloadTypeSimpleSigning
}

// RetrieveAllArtifactURIs returns always an error, feature not available for simplesigning formatter.
func (i *SimpleSigning) RetrieveAllArtifactURIs(_ context.Context, _ interface{}) ([]string, error) {
	return nil, fmt.Errorf("RetrieveAllArtifactURIs not supported for simeplesining formatter")
}
