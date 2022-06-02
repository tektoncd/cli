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

package x509

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	cx509 "crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sigstore/cosign/cmd/cosign/cli/fulcio"
	"github.com/sigstore/cosign/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/cosign/pkg/providers"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/config"
	"go.uber.org/zap"
)

const (
	defaultOIDCClientID = "sigstore"
)

// Signer exposes methods to sign payloads.
type Signer struct {
	cert  string
	chain string
	signature.SignerVerifier
	logger *zap.SugaredLogger
}

// NewSigner returns a configured Signer
func NewSigner(ctx context.Context, secretPath string, cfg config.Config, logger *zap.SugaredLogger) (*Signer, error) {
	x509PrivateKeyPath := filepath.Join(secretPath, "x509.pem")
	cosignPrivateKeypath := filepath.Join(secretPath, "cosign.key")

	if cfg.Signers.X509.FulcioEnabled {
		return fulcioSigner(ctx, cfg.Signers.X509, logger)
	} else if contents, err := ioutil.ReadFile(x509PrivateKeyPath); err == nil {
		return x509Signer(contents, logger)
	} else if contents, err := ioutil.ReadFile(cosignPrivateKeypath); err == nil {
		return cosignSigner(secretPath, contents, logger)
	}
	return nil, errors.New("no valid private key found, looked for: [x509.pem, cosign.key]")
}

func fulcioSigner(ctx context.Context, cfg config.X509Signer, logger *zap.SugaredLogger) (*Signer, error) {
	if !providers.Enabled(ctx) {
		return nil, fmt.Errorf("no auth provider for fulcio is enabled")
	}
	var tok string
	var err error
	if cfg.FulcioProvider != "" {
		logger.Infof("Attempting to get id token from provider %s", cfg.FulcioProvider)
		p, err := providers.ProvideFrom(ctx, cfg.FulcioProvider)
		if err != nil {
			return nil, errors.Wrap(err, "provide from")
		}
		tok, err = p.Provide(ctx, defaultOIDCClientID)
		if err != nil {
			return nil, errors.Wrapf(err, "getting token from provider %s", cfg.FulcioProvider)
		}
	} else {
		tok, err = providers.Provide(ctx, defaultOIDCClientID)
	}
	if err != nil {
		return nil, errors.Wrap(err, "getting provider")
	}

	logger.Info("Signing with fulcio ...")
	k, err := fulcio.NewSigner(ctx, options.KeyOpts{
		FulcioURL:    cfg.FulcioAddr,
		IDToken:      tok,
		OIDCIssuer:   cfg.FulcioOIDCIssuer,
		OIDCClientID: defaultOIDCClientID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "new signer")
	}
	return &Signer{
		SignerVerifier: k.ECDSASignerVerifier,
		cert:           string(k.Cert),
		chain:          string(k.Chain),
		logger:         logger,
	}, nil
}

func x509Signer(privateKey []byte, logger *zap.SugaredLogger) (*Signer, error) {
	logger.Info("Found x509 key...")

	p, _ := pem.Decode(privateKey)
	if p.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("expected private key, found object of type %s", p.Type)
	}
	pk, err := cx509.ParsePKCS8PrivateKey(p.Bytes)
	if err != nil {
		return nil, err
	}
	signer, err := signature.LoadECDSASignerVerifier(pk.(*ecdsa.PrivateKey), crypto.SHA256)
	if err != nil {
		return nil, err
	}
	return &Signer{SignerVerifier: signer, logger: logger}, nil
}

func cosignSigner(secretPath string, privateKey []byte, logger *zap.SugaredLogger) (*Signer, error) {
	logger.Info("Found cosign key...")
	cosignPasswordPath := filepath.Join(secretPath, "cosign.password")
	password, err := ioutil.ReadFile(cosignPasswordPath)
	if err != nil {
		return nil, errors.Wrap(err, "reading cosign.password file")
	}
	signer, err := cosign.LoadPrivateKey(privateKey, password)
	if err != nil {
		return nil, err
	}
	return &Signer{SignerVerifier: signer, logger: logger}, nil
}

func (s *Signer) Type() string {
	return signing.TypeX509
}

func (s *Signer) Cert() string {
	return s.cert
}

// there is no cert or chain, return nothing
func (s *Signer) Chain() string {
	return s.chain
}
