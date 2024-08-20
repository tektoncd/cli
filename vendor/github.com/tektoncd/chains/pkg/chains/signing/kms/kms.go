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
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/kms"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/aws"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/azure"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/gcp"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/hashivault"
	"github.com/sigstore/sigstore/pkg/signature/options"
	"github.com/tektoncd/chains/pkg/config"

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

	// Checks if the vault address provide by the user is a valid address or not
	if cfg.Auth.Address != "" {
		vaultAddress, err := url.Parse(cfg.Auth.Address)
		if err != nil {
			return nil, err
		}

		var vaultUrl *url.URL
		switch {
		case vaultAddress.Port() != "":
			vaultUrl = vaultAddress
		case vaultAddress.Scheme == "http":
			vaultUrl = &url.URL{
				Scheme: vaultAddress.Scheme,
				Host:   vaultAddress.Host + ":80",
			}
		case vaultAddress.Scheme == "https":
			vaultUrl = &url.URL{
				Scheme: vaultAddress.Scheme,
				Host:   vaultAddress.Host + ":443",
			}
		case vaultAddress.Scheme == "":
			vaultUrl = &url.URL{
				Scheme: "http",
				Host:   cfg.Auth.Address + ":80",
			}
		case vaultAddress.Scheme != "" && vaultAddress.Scheme != "http" && vaultAddress.Scheme != "https":
			vaultUrl = &url.URL{
				Scheme: "http",
				Host:   cfg.Auth.Address,
			}
			if vaultUrl.Port() == "" {
				vaultUrl.Host = cfg.Auth.Address + ":80"
			}
		}

		if vaultUrl != nil {
			conn, err := net.DialTimeout("tcp", vaultUrl.Host, 5*time.Second)
			if err != nil {
				return nil, err
			}
			defer conn.Close()
		} else {
			return nil, fmt.Errorf("Error connecting to URL %s\n", cfg.Auth.Address)
		}
	}

	// pass through configuration options to RPCAuth used by KMS in sigstore
	rpcAuth := options.RPCAuth{
		Address: cfg.Auth.Address,
		OIDC: options.RPCAuthOIDC{
			Role: cfg.Auth.OIDC.Role,
			Path: cfg.Auth.OIDC.Path,
		},
	}

	// get token from file KMS_AUTH_TOKEN, a mounted secret at signers.kms.auth.token-dir or
	// as direct value set from signers.kms.auth.token.
	// If both values are set, priority will be given to token-dir.

	if cfg.Auth.TokenPath != "" {
		rpcAuthToken, err := getKMSAuthToken(cfg.Auth.TokenPath)
		if err != nil {
			return nil, err
		}
		rpcAuth.Token = rpcAuthToken
	} else {
		rpcAuth.Token = cfg.Auth.Token
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

// getKMSAuthToken retreives token from the given mount path
func getKMSAuthToken(path string) (string, error) {
	fileData, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file in %q: %w", path, err)
	}

	// A trailing newline is fairly common in mounted files, so remove it.
	fileDataNormalized := strings.TrimSuffix(string(fileData), "\n")
	return fileDataNormalized, nil
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
