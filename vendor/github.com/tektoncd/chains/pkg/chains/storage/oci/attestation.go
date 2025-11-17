// Copyright 2023 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oci

import (
	"context"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	"github.com/sigstore/cosign/v2/pkg/types"
	"github.com/tektoncd/chains/pkg/chains/storage/api"
	"knative.dev/pkg/logging"
)

var (
	_ api.Storer[name.Digest, *intoto.Statement] = &AttestationStorer{}
)

// AttestationStorer stores in-toto Attestation payloads in OCI registries.
type AttestationStorer struct {
	// repo configures the repo where data should be stored.
	// If empty, the repo is inferred from the Artifact.
	repo *name.Repository
	// remoteOpts are additional remote options (i.e. auth) to use for client operations.
	remoteOpts []remote.Option
}

func NewAttestationStorer(opts ...AttestationStorerOption) (*AttestationStorer, error) {
	s := &AttestationStorer{}
	for _, o := range opts {
		if err := o.applyAttestationStorer(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Store saves the given statement.
func (s *AttestationStorer) Store(ctx context.Context, req *api.StoreRequest[name.Digest, *intoto.Statement]) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx)

	repo := req.Artifact.Repository
	if s.repo != nil {
		repo = *s.repo
	}
	se, err := ociremote.SignedEntity(req.Artifact, ociremote.WithRemoteOptions(s.remoteOpts...))
	var entityNotFoundError *ociremote.EntityNotFoundError
	if errors.As(err, &entityNotFoundError) {
		se = ociremote.SignedUnknown(req.Artifact)
	} else if err != nil {
		return nil, errors.Wrap(err, "getting signed image")
	}

	// Create the new attestation for this entity.
	attOpts := []static.Option{static.WithLayerMediaType(types.DssePayloadType)}
	if req.Bundle.Cert != nil {
		attOpts = append(attOpts, static.WithCertChain(req.Bundle.Cert, req.Bundle.Chain))
	}
	att, err := static.NewAttestation(req.Bundle.Signature, attOpts...)
	if err != nil {
		return nil, err
	}
	newImage, err := mutate.AttachAttestationToEntity(se, att)
	if err != nil {
		return nil, err
	}

	// Publish the signatures associated with this entity
	if err := ociremote.WriteAttestations(repo, newImage, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return nil, err
	}
	logger.Infof("Successfully uploaded attestation for %s", req.Artifact.String())

	return &api.StoreResponse{}, nil
}
