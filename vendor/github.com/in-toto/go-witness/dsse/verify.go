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
	"crypto/x509"
	"fmt"
	"time"

	"github.com/in-toto/go-witness/cryptoutil"
	"github.com/in-toto/go-witness/log"
	"github.com/in-toto/go-witness/timestamp"
)

type verificationOptions struct {
	roots              []*x509.Certificate
	intermediates      []*x509.Certificate
	verifiers          []cryptoutil.Verifier
	threshold          int
	timestampVerifiers []timestamp.TimestampVerifier
}

type VerificationOption func(*verificationOptions)

func VerifyWithRoots(roots ...*x509.Certificate) VerificationOption {
	return func(vo *verificationOptions) {
		vo.roots = roots
	}
}

func VerifyWithIntermediates(intermediates ...*x509.Certificate) VerificationOption {
	return func(vo *verificationOptions) {
		vo.intermediates = intermediates
	}
}

func VerifyWithVerifiers(verifiers ...cryptoutil.Verifier) VerificationOption {
	return func(vo *verificationOptions) {
		vo.verifiers = verifiers
	}
}

func VerifyWithThreshold(threshold int) VerificationOption {
	return func(vo *verificationOptions) {
		vo.threshold = threshold
	}
}

func VerifyWithTimestampVerifiers(verifiers ...timestamp.TimestampVerifier) VerificationOption {
	return func(vo *verificationOptions) {
		vo.timestampVerifiers = verifiers
	}
}

type CheckedVerifier struct {
	Verifier           cryptoutil.Verifier
	TimestampVerifiers []timestamp.TimestampVerifier
	Error              error
}

func (e Envelope) Verify(opts ...VerificationOption) ([]CheckedVerifier, error) {
	options := &verificationOptions{
		threshold: 1,
	}

	for _, opt := range opts {
		opt(options)
	}

	if options.threshold <= 0 {
		return nil, ErrInvalidThreshold(options.threshold)
	}

	pae := preauthEncode(e.PayloadType, e.Payload)
	if len(e.Signatures) == 0 {
		return nil, ErrNoSignatures{}
	}

	checkedVerifiers := make([]CheckedVerifier, 0)
	verified := 0
	for _, sig := range e.Signatures {
		if len(sig.Certificate) > 0 {
			cert, err := cryptoutil.TryParseCertificate(sig.Certificate)
			if err != nil {
				continue
			}

			sigIntermediates := make([]*x509.Certificate, 0)
			for _, int := range sig.Intermediates {
				intCert, err := cryptoutil.TryParseCertificate(int)
				if err != nil {
					continue
				}

				sigIntermediates = append(sigIntermediates, intCert)
			}

			sigIntermediates = append(sigIntermediates, options.intermediates...)
			if len(options.timestampVerifiers) == 0 {
				if verifier, err := verifyX509Time(cert, sigIntermediates, options.roots, pae, sig.Signature, time.Now()); err == nil {
					checkedVerifiers = append(checkedVerifiers, CheckedVerifier{Verifier: verifier})
					verified += 1
				} else {
					checkedVerifiers = append(checkedVerifiers, CheckedVerifier{Verifier: verifier, Error: err})
					log.Debugf("failed to verify with timestamp verifier: %w", err)
				}
			} else {
				var passedVerifier cryptoutil.Verifier
				failed := []cryptoutil.Verifier{}
				passedTimestampVerifiers := []timestamp.TimestampVerifier{}
				failedTimestampVerifiers := []timestamp.TimestampVerifier{}

				for _, timestampVerifier := range options.timestampVerifiers {
					for _, sigTimestamp := range sig.Timestamps {
						timestamp, err := timestampVerifier.Verify(context.TODO(), bytes.NewReader(sigTimestamp.Data), bytes.NewReader(sig.Signature))
						if err != nil {
							continue
						}

						if verifier, err := verifyX509Time(cert, sigIntermediates, options.roots, pae, sig.Signature, timestamp); err == nil {
							// NOTE: do we not want to save all the passed verifiers?
							passedVerifier = verifier
							passedTimestampVerifiers = append(passedTimestampVerifiers, timestampVerifier)
						} else {
							failed = append(failed, verifier)
							failedTimestampVerifiers = append(failedTimestampVerifiers, timestampVerifier)
							log.Debugf("failed to verify with timestamp verifier: %w", err)
						}

					}
				}

				if len(passedTimestampVerifiers) > 0 {
					verified += 1
					checkedVerifiers = append(checkedVerifiers, CheckedVerifier{
						Verifier:           passedVerifier,
						TimestampVerifiers: passedTimestampVerifiers,
					})
				} else {
					for _, v := range failed {
						checkedVerifiers = append(checkedVerifiers, CheckedVerifier{
							Verifier:           v,
							TimestampVerifiers: failedTimestampVerifiers,
							Error:              fmt.Errorf("no valid timestamps found"),
						})
					}
				}
			}
		}

		for _, verifier := range options.verifiers {
			if verifier != nil {
				kid, err := verifier.KeyID()
				if err != nil {
					log.Warn("failed to get key id from verifier: %v", err)
				}
				log.Debug("verifying with verifier with KeyID ", kid)

				if err := verifier.Verify(bytes.NewReader(pae), sig.Signature); err == nil {
					verified += 1
					checkedVerifiers = append(checkedVerifiers, CheckedVerifier{Verifier: verifier})
				} else {
					checkedVerifiers = append(checkedVerifiers, CheckedVerifier{Verifier: verifier, Error: err})
				}
			}
		}
	}

	if verified == 0 {
		return nil, ErrNoMatchingSigs{Verifiers: checkedVerifiers}
	} else if verified < options.threshold {
		return checkedVerifiers, ErrThresholdNotMet{Theshold: options.threshold, Actual: verified}
	}

	return checkedVerifiers, nil
}

func verifyX509Time(cert *x509.Certificate, sigIntermediates, roots []*x509.Certificate, pae, sig []byte, trustedTime time.Time) (cryptoutil.Verifier, error) {
	verifier, err := cryptoutil.NewX509Verifier(cert, sigIntermediates, roots, trustedTime)
	if err != nil {
		return nil, err
	}

	err = verifier.Verify(bytes.NewReader(pae), sig)

	return verifier, err
}
