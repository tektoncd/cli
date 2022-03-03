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
	"fmt"

	"github.com/sigstore/sigstore/pkg/signature/payload"
	"github.com/tektoncd/chains/pkg/chains/formats"

	"github.com/google/go-containerregistry/pkg/name"
)

// SimpleSigning is a formatter that uses the RedHat simple signing format
// https://www.redhat.com/en/blog/container-image-signing
type SimpleSigning struct {
}

type SimpleContainerImage payload.SimpleContainerImage

// CreatePayload implements the Payloader interface.
func (i *SimpleSigning) CreatePayload(obj interface{}) (interface{}, error) {
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

func NewFormatter() (formats.Payloader, error) {
	return &SimpleSigning{}, nil
}

func NewSimpleStruct(img name.Digest) SimpleContainerImage {
	cosign := payload.Cosign{Image: img}
	return SimpleContainerImage(cosign.SimpleContainerImage())
}

func (i SimpleContainerImage) ImageName() string {
	return fmt.Sprintf("%s@%s", i.Critical.Identity.DockerReference, i.Critical.Image.DockerManifestDigest)
}

func (i *SimpleSigning) Type() formats.PayloadType {
	return formats.PayloadTypeSimpleSigning
}
