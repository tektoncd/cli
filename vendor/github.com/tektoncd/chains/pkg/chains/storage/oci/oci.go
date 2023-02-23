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

	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/secure-systems-lab/go-securesystemslib/dsse"

	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/pkg/oci"
	"github.com/sigstore/cosign/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
	"github.com/sigstore/cosign/pkg/oci/static"
	"github.com/sigstore/cosign/pkg/types"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats/simple"
	"github.com/tektoncd/chains/pkg/config"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

const StorageBackendOCI = "oci"

type Backend struct {
	logger           *zap.SugaredLogger
	cfg              config.Config
	client           kubernetes.Interface
	getAuthenticator func(ctx context.Context, obj objects.TektonObject, client kubernetes.Interface) (remote.Option, error)
}

// NewStorageBackend returns a new OCI StorageBackend that stores signatures in an OCI registry
func NewStorageBackend(ctx context.Context, logger *zap.SugaredLogger, client kubernetes.Interface, cfg config.Config) *Backend {
	return &Backend{
		logger: logger,
		cfg:    cfg,
		client: client,
		getAuthenticator: func(ctx context.Context, obj objects.TektonObject, client kubernetes.Interface) (remote.Option, error) {
			kc, err := k8schain.New(ctx, client,
				k8schain.Options{
					Namespace:          obj.GetNamespace(),
					ServiceAccountName: obj.GetServiceAccountName(),
					ImagePullSecrets:   obj.GetPullSecrets(),
				})
			if err != nil {
				return nil, err
			}
			return remote.WithAuthFromKeychain(kc), nil
		},
	}
}

// StorePayload implements the storage.Backend interface.
func (b *Backend) StorePayload(ctx context.Context, obj objects.TektonObject, rawPayload []byte, signature string, storageOpts config.StorageOpts) error {
	auth, err := b.getAuthenticator(ctx, obj, b.client)
	if err != nil {
		return err
	}

	b.logger.Infof("Storing payload on %s/%s/%s", obj.GetGVK(), obj.GetNamespace(), obj.GetName())

	if storageOpts.PayloadFormat == formats.PayloadTypeSimpleSigning {
		format := simple.SimpleContainerImage{}
		if err := json.Unmarshal(rawPayload, &format); err != nil {
			return errors.Wrap(err, "unmarshal simplesigning")
		}
		return b.uploadSignature(format, rawPayload, signature, storageOpts, auth)
	}

	if _, ok := formats.IntotoAttestationSet[storageOpts.PayloadFormat]; ok {
		attestation := in_toto.Statement{}
		if err := json.Unmarshal(rawPayload, &attestation); err != nil {
			return errors.Wrap(err, "unmarshal attestation")
		}

		// This can happen if the Task/TaskRun does not adhere to specific naming conventions
		// like *IMAGE_URL that would serve as hints. This may be intentional for a Task/TaskRun
		// that is not intended to produce an image, e.g. git-clone.
		if len(attestation.Subject) == 0 {
			b.logger.Infof(
				"No image subject to attest for %s/%s/%s. Skipping upload to registry", obj.GetGVK(), obj.GetNamespace(), obj.GetName())
			return nil
		}

		return b.uploadAttestation(attestation, signature, storageOpts, auth)
	}

	// Fallback in case unsupported payload format is used or the deprecated "tekton" format
	b.logger.Info("Skipping upload to OCI registry, OCI storage backend is only supported for OCI images and in-toto attestations")
	return nil
}

func (b *Backend) uploadSignature(format simple.SimpleContainerImage, rawPayload []byte, signature string, storageOpts config.StorageOpts, remoteOpts ...remote.Option) error {
	imageName := format.ImageName()
	b.logger.Infof("Uploading %s signature", imageName)
	var opts []name.Option
	if b.cfg.Storage.OCI.Insecure {
		opts = append(opts, name.Insecure)
	}
	ref, err := name.NewDigest(imageName, opts...)
	if err != nil {
		return errors.Wrap(err, "getting digest")
	}
	se, err := ociremote.SignedEntity(ref, ociremote.WithRemoteOptions(remoteOpts...))
	if err != nil {
		return errors.Wrap(err, "getting signed image")
	}

	sigOpts := []static.Option{}
	if storageOpts.Cert != "" {
		sigOpts = append(sigOpts, static.WithCertChain([]byte(storageOpts.Cert), []byte(storageOpts.Chain)))
	}
	// Create the new signature for this entity.
	b64sig := base64.StdEncoding.EncodeToString([]byte(signature))
	sig, err := static.NewSignature(rawPayload, b64sig, sigOpts...)
	if err != nil {
		return err
	}
	// Attach the signature to the entity.
	newSE, err := mutate.AttachSignatureToEntity(se, sig)
	if err != nil {
		return err
	}
	repo := ref.Repository
	if b.cfg.Storage.OCI.Repository != "" {
		repo, err = name.NewRepository(b.cfg.Storage.OCI.Repository)
		if err != nil {
			return errors.Wrapf(err, "%s is not a valid repository", b.cfg.Storage.OCI.Repository)
		}
	}
	// Publish the signatures associated with this entity
	if err := ociremote.WriteSignatures(repo, newSE, ociremote.WithRemoteOptions(remoteOpts...)); err != nil {
		return err
	}
	b.logger.Infof("Successfully uploaded signature for %s", imageName)
	return nil
}

func (b *Backend) uploadAttestation(attestation in_toto.Statement, signature string, storageOpts config.StorageOpts, remoteOpts ...remote.Option) error {
	// upload an attestation for each subject
	b.logger.Info("Starting to upload attestations to OCI ...")
	for _, subj := range attestation.Subject {
		imageName := fmt.Sprintf("%s@sha256:%s", subj.Name, subj.Digest["sha256"])
		b.logger.Infof("Starting attestation upload to OCI for %s...", imageName)
		var opts []name.Option
		if b.cfg.Storage.OCI.Insecure {
			opts = append(opts, name.Insecure)
		}
		ref, err := name.NewDigest(imageName, opts...)
		if err != nil {
			return errors.Wrapf(err, "getting digest for subj %s", imageName)
		}
		repo := ref.Repository
		if b.cfg.Storage.OCI.Repository != "" {
			repo, err = name.NewRepository(b.cfg.Storage.OCI.Repository)
			if err != nil {
				return errors.Wrapf(err, "%s is not a valid repository", b.cfg.Storage.OCI.Repository)
			}
		}
		se, err := ociremote.SignedEntity(ref, ociremote.WithRemoteOptions(remoteOpts...))
		if err != nil {
			return errors.Wrap(err, "getting signed image")
		}
		// Create the new attestation for this entity.
		attOpts := []static.Option{static.WithLayerMediaType(types.DssePayloadType)}
		if storageOpts.Cert != "" {
			attOpts = append(attOpts, static.WithCertChain([]byte(storageOpts.Cert), []byte(storageOpts.Chain)))
		}
		att, err := static.NewAttestation([]byte(signature), attOpts...)
		if err != nil {
			return err
		}
		newImage, err := mutate.AttachAttestationToEntity(se, att)
		if err != nil {
			return err
		}
		// Publish the signatures associated with this entity
		if err := ociremote.WriteAttestations(repo, newImage, ociremote.WithRemoteOptions(remoteOpts...)); err != nil {
			return err
		}
		b.logger.Infof("Successfully uploaded attestation for %s", imageName)
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
	images := artifacts.ExtractOCIImagesFromResults(obj, b.logger)
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
