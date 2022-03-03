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
	"io/ioutil"

	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/secure-systems-lab/go-securesystemslib/dsse"
	"github.com/sigstore/sigstore/pkg/signature"

	"golang.org/x/crypto/ssh"
)

func Wrap(ctx context.Context, s Signer) (Signer, error) {
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
		KeyID:   fingerprint,
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
	KeyID   string
}

func (w *sslAdapter) Sign(data []byte) ([]byte, string, error) {
	sig, err := w.wrapped.SignMessage(bytes.NewReader(data))
	return sig, w.KeyID, err
}

func (w *sslAdapter) Verify(keyID string, data, sig []byte) error {
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

func (s *sslSigner) Type() string {
	return s.typ
}
func (s *sslSigner) PublicKey(opts ...signature.PublicKeyOption) (crypto.PublicKey, error) {
	return s.pub, nil
}

func (s *sslSigner) Sign(ctx context.Context, payload []byte) ([]byte, []byte, error) {
	env, err := s.wrapper.SignPayload(in_toto.PayloadType, payload)
	if err != nil {
		return nil, nil, err
	}
	b, err := json.Marshal(env)
	if err != nil {
		return nil, nil, err
	}
	return b, []byte(env.Payload), nil
}

func (s *sslSigner) SignMessage(payload io.Reader, opts ...signature.SignOption) ([]byte, error) {
	m, err := ioutil.ReadAll(payload)
	if err != nil {
		return nil, err
	}
	env, err := s.wrapper.SignPayload(in_toto.PayloadType, m)
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
