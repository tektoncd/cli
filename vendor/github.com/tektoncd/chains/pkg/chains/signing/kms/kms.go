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

// Package kms creates a signer using a key management server
package kms

import (
	"context"
	"crypto"

	"github.com/tektoncd/chains/pkg/config"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/kms"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/aws"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/azure"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/gcp"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/hashivault"
	"github.com/sigstore/sigstore/pkg/signature/options"

	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"github.com/tektoncd/chains/pkg/chains/signing"
)

// Signer exposes methods to sign payloads using a KMS
type Signer struct {
	signature.SignerVerifier
}

// NewSigner returns a configured Signer
func NewSigner(ctx context.Context, cfg config.KMSSigner) (*Signer, error) {
	kmsOpts := []signature.RPCOption{}
	// pass through configuration options to RPCAuth used by KMS in sigstore
	rpcAuth := options.RPCAuth{
		Address: cfg.Auth.Address,
		Token:   cfg.Auth.Token,
		OIDC: options.RPCAuthOIDC{
			Role: cfg.Auth.OIDC.Role,
			Path: cfg.Auth.OIDC.Path,
		},
	}
	// get token from spire
	if cfg.Auth.Spire.Sock != "" {
		token, err := newSpireToken(ctx, cfg)
		if err != nil {
			return nil, err
		}
		rpcAuth.OIDC.Token = token
	}
	kmsOpts = append(kmsOpts, options.WithRPCAuthOpts(rpcAuth))
	// get the signer/verifier from sigstore
	k, err := kms.Get(ctx, cfg.KMSRef, crypto.SHA256, kmsOpts...)
	if err != nil {
		return nil, err
	}
	return &Signer{
		SignerVerifier: k,
	}, nil
}

// newSpireToken retrieves an SVID token from Spire
func newSpireToken(ctx context.Context, cfg config.KMSSigner) (string, error) {
	jwtSource, err := workloadapi.NewJWTSource(
		ctx,
		workloadapi.WithClientOptions(workloadapi.WithAddr(cfg.Auth.Spire.Sock)),
	)
	if err != nil {
		return "", err
	}
	svid, err := jwtSource.FetchJWTSVID(ctx, jwtsvid.Params{Audience: cfg.Auth.Spire.Audience})
	if err != nil {
		return "", err
	}
	return svid.Marshal(), nil
}

// Type returns the type of the signer
func (s *Signer) Type() string {
	return signing.TypeKMS
}

// Cert there is no cert, return nothing
func (s *Signer) Cert() string {
	return ""
}

// Chain there is no chain, return nothing
func (s *Signer) Chain() string {
	return ""
}
