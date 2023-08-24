/*
Copyright 2021 The Tekton Authors
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

package signing

import (
	"github.com/sigstore/sigstore/pkg/signature"
)

type Signer interface {
	signature.SignerVerifier
	Type() string
	Cert() string
	Chain() string
}

const (
	TypeX509 = "x509"
	TypeKMS  = "kms"
)

var AllSigners = []string{TypeX509, TypeKMS}

// Bundle represents the output of a signing operation.
type Bundle struct {
	// Content is the raw content that was signed.
	Content []byte
	// Signature is the content signature.
	Signature []byte
	// Cert is an optional PEM encoded x509 certificate, if one was used for signing.
	Cert []byte
	// Cert is an optional PEM encoded x509 certificate chain, if one was used for signing.
	Chain []byte
}
