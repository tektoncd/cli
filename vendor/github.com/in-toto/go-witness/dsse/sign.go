// Copyright 2022 The Witness Contributors
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

package dsse

import (
	"bytes"
	"context"
	"encoding/pem"
	"fmt"
	"io"

	"github.com/in-toto/go-witness/cryptoutil"
	"github.com/in-toto/go-witness/timestamp"
)

type signOptions struct {
	signers      []cryptoutil.Signer
	timestampers []timestamp.Timestamper
}

type SignOption func(*signOptions)

func SignWithSigners(signers ...cryptoutil.Signer) SignOption {
	return func(so *signOptions) {
		so.signers = signers
	}
}

func SignWithTimestampers(timestampers ...timestamp.Timestamper) SignOption {
	return func(so *signOptions) {
		so.timestampers = timestampers
	}
}

func Sign(bodyType string, body io.Reader, opts ...SignOption) (Envelope, error) {
	so := &signOptions{}
	env := Envelope{}
	for _, opt := range opts {
		opt(so)
	}

	if len(so.signers) == 0 {
		return env, fmt.Errorf("must have at least one signer, have %v", len(so.signers))
	}

	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return env, err
	}

	env.PayloadType = bodyType
	env.Payload = bodyBytes
	env.Signatures = make([]Signature, 0)
	pae := preauthEncode(bodyType, bodyBytes)
	for _, signer := range so.signers {
		if signer == nil {
			continue
		}

		sig, err := signer.Sign(bytes.NewReader(pae))
		if err != nil {
			return env, err
		}

		keyID, err := signer.KeyID()
		if err != nil {
			return env, err
		}

		dsseSig := Signature{
			KeyID:     keyID,
			Signature: sig,
		}

		for _, timestamper := range so.timestampers {
			timestamp, err := timestamper.Timestamp(context.TODO(), bytes.NewReader(sig))
			if err != nil {
				return env, err
			}

			dsseSig.Timestamps = append(dsseSig.Timestamps, SignatureTimestamp{
				Type: TimestampRFC3161,
				Data: timestamp,
			})
		}

		if trustBundler, ok := signer.(cryptoutil.TrustBundler); ok {
			leaf := trustBundler.Certificate()
			intermediates := trustBundler.Intermediates()
			if leaf != nil {
				dsseSig.Certificate = pem.EncodeToMemory(&pem.Block{Type: PemTypeCertificate, Bytes: leaf.Raw})
			}

			for _, intermediate := range intermediates {
				dsseSig.Intermediates = append(dsseSig.Intermediates, pem.EncodeToMemory(&pem.Block{Type: PemTypeCertificate, Bytes: intermediate.Raw}))
			}
		}

		env.Signatures = append(env.Signatures, dsseSig)
	}

	return env, nil
}
