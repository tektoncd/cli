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

package config

// PayloadType specifies the format to store payload in.
// - For OCI artifact, Chains only supports `simplesigning` format. https://www.redhat.com/en/blog/container-image-signing
// - For Tekton artifacts, Chains supports `tekton` and `in-toto` format. https://slsa.dev/provenance/v0.2
type PayloadType string

// StorageOpts contains additional information required when storing signatures
type StorageOpts struct {
	// FullKey stands for the identifier of an artifact.
	// - For OCI artifact, it is the full representation in the format of `<NAME>@sha256:<DIGEST>`.
	// - For TaskRun/PipelineRun artifact, it is `<GROUP>-<VERSION>-<KIND>-<UID>`
	FullKey string

	// ShortKey is the short version of an artifact identifier. This is useful for annotation based storage
	// because annotation key has limitations (https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/#syntax-and-character-set).
	// - For OCI artifact, it is first 12 chars of the image digest.
	// - For TaskRun/PipelineRun artifact, it is `<KIND>-<UID>`.
	ShortKey string

	// Cert is an OPTIONAL property that contains a PEM-encoded x509 certificate.
	// If present, this certificate MUST embed the public key that can be used to verify the signature.
	// https://github.com/sigstore/cosign/blob/main/specs/SIGNATURE_SPEC.md
	Cert string

	// Chain string is an OPTIONAL property that contains a PEM-encoded, DER-formatted, ASN.1 x509 certificate chain.
	// The certificate property MUST be present if this property is present.
	// This chain MAY be used by implementations to verify the certificate property.
	// https://github.com/sigstore/cosign/blob/main/specs/SIGNATURE_SPEC.md
	Chain string

	// PayloadFormat is the format to store payload in.
	PayloadFormat PayloadType
}
