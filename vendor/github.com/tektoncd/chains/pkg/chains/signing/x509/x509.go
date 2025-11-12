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
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/fulcio"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/providers"
	"knative.dev/pkg/logging"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/tuf"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/config"
)

const (
	defaultOIDCClientID = "sigstore"
)

// Signer exposes methods to sign payloads.
type Signer struct {
	cert  string
	chain string
	signature.SignerVerifier
}

// NewSigner returns a configured Signer
func NewSigner(ctx context.Context, secretPath string, cfg config.Config) (*Signer, error) {
	x509PrivateKeyPath := filepath.Join(secretPath, "x509.pem")
	cosignPrivateKeypath := filepath.Join(secretPath, "cosign.key")

	if cfg.Signers.X509.FulcioEnabled {
		return fulcioSigner(ctx, cfg.Signers.X509)
	} else if contents, err := os.ReadFile(x509PrivateKeyPath); err == nil {
		return x509Signer(ctx, contents)
	} else if contents, err := os.ReadFile(cosignPrivateKeypath); err == nil {
		return cosignSigner(ctx, secretPath, contents)
	}
	return nil, errors.New("no valid private key found, looked for: [x509.pem, cosign.key]")
}

func fulcioSigner(ctx context.Context, cfg config.X509Signer) (*Signer, error) {
	logger := logging.FromContext(ctx)

	providersEnabled := providers.Enabled(ctx)

	if cfg.IdentityTokenFile != "" {
		FilesystemTokenPath = cfg.IdentityTokenFile
		providersEnabled = true
	}
	if !providersEnabled {
		return nil, fmt.Errorf("no auth provider for fulcio is enabled")
	}
	var tok string
	var err error
	if cfg.TUFMirrorURL != tuf.DefaultRemoteRoot {
		if err = initializeTUF(ctx, cfg.TUFMirrorURL); err != nil {
			return nil, errors.Wrap(err, "initialize tuf")
		}
	}

	if cfg.IdentityTokenFile != "" {
		switch cfg.FulcioProvider {
		// cosign providers package hardcodes the token path value
		// "filesystem-custom-path" accepts a variable for the token path
		case fsDefaultCosignTokenPathProvider, "", fsCustomTokenPathProvider:
			cfg.FulcioProvider = fsCustomTokenPathProvider
		}
	}

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
		// if FulcioProvider is not set, all will be tried
		tok, err = providers.Provide(ctx, defaultOIDCClientID)
	}
	if err != nil {
		return nil, errors.Wrap(err, "getting provider")
	}

	logger.Info("Signing with fulcio ...")
	priv, err := cosign.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("error generating keypair: %w", err)
	}
	signer, err := signature.LoadECDSASignerVerifier(priv, crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("error loading sigstore signer: %w", err)
	}
	k, err := fulcio.NewSigner(ctx, options.KeyOpts{
		FulcioURL:    cfg.FulcioAddr,
		IDToken:      tok,
		OIDCIssuer:   cfg.FulcioOIDCIssuer,
		OIDCClientID: defaultOIDCClientID,
	}, signer)
	if err != nil {
		return nil, errors.Wrap(err, "new signer")
	}
	return &Signer{
		SignerVerifier: signer,
		cert:           string(k.Cert),
		chain:          string(k.Chain),
	}, nil
}

// root: TUF_URL/root.json
// mirror: TUF_URL
func initializeTUF(ctx context.Context, mirror string) error {
	logger := logging.FromContext(ctx)
	// Get the initial trusted root contents.
	root, err := url.JoinPath(mirror, "root.json")
	if err != nil {
		return err
	}
	rootFileBytes, err := loadRootFromURL(root)
	if err != nil {
		return err
	}
	if err = tuf.Initialize(ctx, mirror, rootFileBytes); err != nil {
		return err
	}
	status, err := tuf.GetRootStatus(ctx)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(status, "", "\t")
	if err != nil {
		return err
	}
	logger.Infof("Root status: %s", string(b))
	return nil
}

func loadRootFromURL(root string) ([]byte, error) {
	resp, err := http.Get(root)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func x509Signer(ctx context.Context, privateKey []byte) (*Signer, error) {
	logger := logging.FromContext(ctx)
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
	return &Signer{SignerVerifier: signer}, nil
}

func cosignSigner(ctx context.Context, secretPath string, privateKey []byte) (*Signer, error) {
	logger := logging.FromContext(ctx)
	logger.Info("Found cosign key...")
	cosignPasswordPath := filepath.Join(secretPath, "cosign.password")
	password, err := os.ReadFile(cosignPasswordPath)
	if err != nil {
		return nil, errors.Wrap(err, "reading cosign.password file")
	}
	signer, err := cosign.LoadPrivateKey(privateKey, password, nil)
	if err != nil {
		return nil, err
	}
	return &Signer{SignerVerifier: signer}, nil
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
