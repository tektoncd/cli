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

package oci

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/storage/api"

	"knative.dev/pkg/logging"

	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/secure-systems-lab/go-securesystemslib/dsse"

	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/v2/pkg/oci"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats/simple"
	"github.com/tektoncd/chains/pkg/config"
	"k8s.io/client-go/kubernetes"
)

const StorageBackendOCI = "oci"

// Backend implements a storage backend for OCI artifacts.
// Deprecated: Use SimpleStorer and AttestationStorer instead.
type Backend struct {
	cfg              config.Config
	client           kubernetes.Interface
	getAuthenticator func(ctx context.Context, obj objects.TektonObject, client kubernetes.Interface) (remote.Option, error)
}

// NewStorageBackend returns a new OCI StorageBackend that stores signatures in an OCI registry
func NewStorageBackend(ctx context.Context, client kubernetes.Interface, cfg config.Config) *Backend {
	return &Backend{
		cfg: cfg,

		client: client,
		getAuthenticator: func(ctx context.Context, obj objects.TektonObject, client kubernetes.Interface) (remote.Option, error) {
			kc, err := k8schain.New(ctx, client,
				k8schain.Options{
					Namespace:          obj.GetNamespace(),
					ServiceAccountName: obj.GetServiceAccountName(),
					UseMountSecrets:    true,
				})
			if err != nil {
				return nil, errors.Wrapf(err, "creating new keychain from serviceaccount %s/%s", obj.GetNamespace(), obj.GetServiceAccountName())
			}
			return remote.WithAuthFromKeychain(kc), nil
		},
	}
}

// StorePayload implements the storage.Backend interface.
func (b *Backend) StorePayload(ctx context.Context, obj objects.TektonObject, rawPayload []byte, signature string, storageOpts config.StorageOpts) error {
	logger := logging.FromContext(ctx)
	auth, err := b.getAuthenticator(ctx, obj, b.client)
	if err != nil {
		return errors.Wrap(err, "getting oci authenticator")
	}

	logger.Infof("Storing payload on %s/%s/%s", obj.GetGVK(), obj.GetNamespace(), obj.GetName())

	if storageOpts.PayloadFormat == formats.PayloadTypeSimpleSigning {
		format := simple.SimpleContainerImage{}
		if err := json.Unmarshal(rawPayload, &format); err != nil {
			return errors.Wrap(err, "unmarshal simplesigning")
		}
		return b.uploadSignature(ctx, format, rawPayload, signature, storageOpts, auth)
	}

	if _, ok := formats.IntotoAttestationSet[storageOpts.PayloadFormat]; ok {
		attestation := intoto.Statement{}
		if err := json.Unmarshal(rawPayload, &attestation); err != nil {
			return errors.Wrap(err, "unmarshal attestation")
		}

		// This can happen if the Task/TaskRun does not adhere to specific naming conventions
		// like *IMAGE_URL that would serve as hints. This may be intentional for a Task/TaskRun
		// that is not intended to produce an image, e.g. git-clone.
		if len(attestation.Subject) == 0 {
			logger.Infof(
				"No image subject to attest for %s/%s/%s. Skipping upload to registry", obj.GetGVK(), obj.GetNamespace(), obj.GetName())
			return nil
		}

		return b.uploadAttestation(ctx, &attestation, signature, storageOpts, auth)
	}

	// Fallback in case unsupported payload format is used or the deprecated "tekton" format
	logger.Info("Skipping upload to OCI registry, OCI storage backend is only supported for OCI images and in-toto attestations")
	return nil
}

func (b *Backend) uploadSignature(ctx context.Context, format simple.SimpleContainerImage, rawPayload []byte, signature string, storageOpts config.StorageOpts, remoteOpts ...remote.Option) error {
	logger := logging.FromContext(ctx)

	imageName := format.ImageName()
	logger.Infof("Uploading %s signature", imageName)

	ref, err := name.NewDigest(imageName)
	if err != nil {
		return errors.Wrap(err, "getting digest")
	}

	repo, err := newRepo(b.cfg, ref)
	if err != nil {
		return errors.Wrapf(err, "getting storage repo for sub %s", imageName)
	}

	store, err := NewSimpleStorerFromConfig(WithTargetRepository(repo))
	if err != nil {
		return err
	}
	// TODO: make these creation opts.
	store.remoteOpts = remoteOpts
	if _, err := store.Store(ctx, &api.StoreRequest[name.Digest, simple.SimpleContainerImage]{
		Object:   nil,
		Artifact: ref,
		Payload:  format,
		Bundle: &signing.Bundle{
			Content:   rawPayload,
			Signature: []byte(signature),
			Cert:      []byte(storageOpts.Cert),
			Chain:     []byte(storageOpts.Chain),
		},
	}); err != nil {
		return err
	}
	return nil
}

func (b *Backend) uploadAttestation(ctx context.Context, attestation *intoto.Statement, signature string, storageOpts config.StorageOpts, remoteOpts ...remote.Option) error {
	logger := logging.FromContext(ctx)
	// upload an attestation for each subject
	logger.Info("Starting to upload attestations to OCI ...")
	for _, subj := range attestation.Subject {
		imageName := fmt.Sprintf("%s@sha256:%s", subj.Name, subj.Digest["sha256"])
		logger.Infof("Starting attestation upload to OCI for %s...", imageName)

		ref, err := name.NewDigest(imageName)
		if err != nil {
			return errors.Wrapf(err, "getting digest for subj %s", imageName)
		}

		repo, err := newRepo(b.cfg, ref)
		if err != nil {
			return errors.Wrapf(err, "getting storage repo for sub %s", imageName)
		}

		store, err := NewAttestationStorer(WithTargetRepository(repo))
		if err != nil {
			return err
		}
		// TODO: make these creation opts.
		store.remoteOpts = remoteOpts
		if _, err := store.Store(ctx, &api.StoreRequest[name.Digest, *intoto.Statement]{
			Object:   nil,
			Artifact: ref,
			Payload:  attestation,
			Bundle: &signing.Bundle{
				Content:   nil,
				Signature: []byte(signature),
				Cert:      []byte(storageOpts.Cert),
				Chain:     []byte(storageOpts.Chain),
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (b *Backend) Type() string {
	return StorageBackendOCI
}

func (b *Backend) RetrieveSignatures(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string][]string, error) {
	images, err := b.RetrieveArtifact(ctx, obj, opts)
	if err != nil {
		return nil, err
	}

	m := make(map[string][]string)
	for ref, img := range images {
		sigImage, err := img.Signatures()
		if err != nil {
			return nil, err
		}
		sigs, err := sigImage.Get()
		if err != nil {
			return nil, err
		}

		signatures := []string{}
		for _, s := range sigs {
			if sig, err := s.Base64Signature(); err == nil {
				signatures = append(signatures, sig)
			}
		}
		m[ref] = signatures
	}
	return m, nil
}

func (b *Backend) RetrievePayloads(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string]string, error) {
	var err error
	images, err := b.RetrieveArtifact(ctx, obj, opts)
	if err != nil {
		return nil, err
	}

	m := make(map[string]string)
	var attImage oci.Signatures
	for ref, img := range images {
		if opts.PayloadFormat == formats.PayloadTypeSimpleSigning {
			attImage, err = img.Signatures()
		} else {
			attImage, err = img.Attestations()
		}
		if err != nil {
			return nil, err
		}
		atts, err := attImage.Get()
		if err != nil {
			return nil, err
		}

		for _, s := range atts {
			if payload, err := s.Payload(); err == nil {
				envelope := dsse.Envelope{}
				if err := json.Unmarshal(payload, &envelope); err != nil {
					return nil, fmt.Errorf("cannot decode the envelope: %s", err)
				}

				var decodedPayload []byte
				decodedPayload, err = base64.StdEncoding.DecodeString(envelope.Payload)
				if err != nil {
					return nil, fmt.Errorf("error decoding the payload: %s", err)
				}

				m[ref] = string(decodedPayload)
			}

		}
	}
	return m, nil
}

func (b *Backend) RetrieveArtifact(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string]oci.SignedImage, error) {
	// Given the TaskRun, retrieve the OCI images.
	images := artifacts.ExtractOCIImagesFromResults(ctx, obj.GetResults())
	m := make(map[string]oci.SignedImage)

	for _, image := range images {
		ref, ok := image.(name.Digest)
		if !ok {
			return nil, errors.New("error parsing image")
		}
		img, err := ociremote.SignedImage(ref)
		if err != nil {
			return nil, err
		}
		m[ref.DigestStr()] = img
	}

	return m, nil
}

func newRepo(cfg config.Config, imageName name.Digest) (name.Repository, error) {
	var opts []name.Option
	if cfg.Storage.OCI.Insecure {
		opts = append(opts, name.Insecure)
	}

	if storageOCIRepository := cfg.Storage.OCI.Repository; storageOCIRepository != "" {
		return name.NewRepository(storageOCIRepository, opts...)
	}
	return name.NewRepository(imageName.Repository.Name(), opts...)
}
