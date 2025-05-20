// Copyright 2021 The Witness Contributors
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
	"fmt"

	"github.com/in-toto/go-witness/log"
)

type ErrNoSignatures struct{}

func (e ErrNoSignatures) Error() string {
	return "no signatures in dsse envelope"
}

type ErrNoMatchingSigs struct {
	Verifiers []CheckedVerifier
}

func (e ErrNoMatchingSigs) Error() string {
	mess := "no valid signatures for the provided verifiers found for keyids:\n"
	for _, v := range e.Verifiers {
		if v.Error != nil {
			kid, err := v.Verifier.KeyID()
			if err != nil {
				log.Warnf("failed to get key id from verifier: %w", err)
			}

			s := fmt.Sprintf("  %s: %v\n", kid, v.Error)
			mess += s
		}
	}

	return mess
}

type ErrThresholdNotMet struct {
	Theshold int
	Actual   int
}

func (e ErrThresholdNotMet) Error() string {
	return fmt.Sprintf("envelope did not meet verifier threshold. expected %v valid verifiers but got %v", e.Theshold, e.Actual)
}

type ErrInvalidThreshold int

func (e ErrInvalidThreshold) Error() string {
	return fmt.Sprintf("invalid threshold (%v). thresholds must be greater than 0", int(e))
}

const PemTypeCertificate = "CERTIFICATE"

type Envelope struct {
	Payload     []byte      `json:"payload"`
	PayloadType string      `json:"payloadType"`
	Signatures  []Signature `json:"signatures"`
}

type Signature struct {
	KeyID         string               `json:"keyid"`
	Signature     []byte               `json:"sig"`
	Certificate   []byte               `json:"certificate,omitempty"`
	Intermediates [][]byte             `json:"intermediates,omitempty"`
	Timestamps    []SignatureTimestamp `json:"timestamps,omitempty"`
}

type SignatureTimestampType string

const TimestampRFC3161 SignatureTimestampType = "tsp"

type SignatureTimestamp struct {
	Type SignatureTimestampType `json:"type"`
	Data []byte                 `json:"data"`
}

// preauthEncode wraps the data to be signed or verified and it's type in the DSSE protocol's
// pre-authentication encoding as detailed at https://github.com/secure-systems-lab/dsse/blob/master/protocol.md
// PAE(type, body) = "DSSEv1" + SP + LEN(type) + SP + type + SP + LEN(body) + SP + body
func preauthEncode(bodyType string, body []byte) []byte {
	const dsseVersion = "DSSEv1"
	return []byte(fmt.Sprintf("%s %d %s %d %s", dsseVersion, len(bodyType), bodyType, len(body), body))
}
