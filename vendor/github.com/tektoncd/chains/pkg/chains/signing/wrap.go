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
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"fmt"
	"io"

	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/secure-systems-lab/go-securesystemslib/dsse"
	"github.com/sigstore/sigstore/pkg/signature"

	"golang.org/x/crypto/ssh"
)

func Wrap(s Signer) (Signer, error) {
	pub, err := s.PublicKey()
	if err != nil {
		return nil, err
	}

	// Generate public key fingerprint
	sshpk, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, err
	}
	fingerprint := ssh.FingerprintSHA256(sshpk)

	adapter := sslAdapter{
		wrapped: s,
		keyID:   fingerprint,
		pk:      sshpk,
	}

	envelope, err := dsse.NewEnvelopeSigner(&adapter)
	if err != nil {
		return nil, err
	}
	return &sslSigner{
		wrapper: envelope,
		typ:     s.Type(),
		pub:     pub,
		cert:    s.Cert(),
		chain:   s.Chain(),
	}, nil
}

// sslAdapter converts our signing objects into the type expected by the Envelope signer for wrapping.
type sslAdapter struct {
	wrapped Signer
	keyID   string
	pk      crypto.PublicKey
}

func (w *sslAdapter) Sign(ctx context.Context, data []byte) ([]byte, error) {
	sig, err := w.wrapped.SignMessage(bytes.NewReader(data))
	return sig, err
}

func (w *sslAdapter) KeyID() (string, error) {
	return w.keyID, nil
}

func (w *sslAdapter) Public() crypto.PublicKey {
	return w.pk
}

func (w *sslAdapter) Verify(_ context.Context, data, sig []byte) error {
	panic("unimplemented")
}

// sslSigner converts the EnvelopeSigners back into our types, after wrapping.
type sslSigner struct {
	wrapper *dsse.EnvelopeSigner
	typ     string
	pub     crypto.PublicKey
	cert    string
	chain   string
}

func (w *sslSigner) Type() string {
	return w.typ
}
func (w *sslSigner) PublicKey(opts ...signature.PublicKeyOption) (crypto.PublicKey, error) {
	return w.pub, nil
}

func (w *sslSigner) Sign(ctx context.Context, payload []byte) ([]byte, []byte, error) {
	env, err := w.wrapper.SignPayload(ctx, in_toto.PayloadType, payload)
	if err != nil {
		return nil, nil, err
	}
	b, err := json.Marshal(env)
	if err != nil {
		return nil, nil, err
	}
	return b, []byte(env.Payload), nil
}

func (w *sslSigner) SignMessage(payload io.Reader, opts ...signature.SignOption) ([]byte, error) {
	m, err := io.ReadAll(payload)
	if err != nil {
		return nil, err
	}
	env, err := w.wrapper.SignPayload(context.TODO(), in_toto.PayloadType, m)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (w *sslSigner) Cert() string {
	return w.cert
}

func (w *sslSigner) Chain() string {
	return w.chain
}

func (w *sslSigner) VerifySignature(signature, message io.Reader, opts ...signature.VerifyOption) error {
	return fmt.Errorf("not implemented")
}
