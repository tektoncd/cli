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

package timestamp

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/digitorus/pkcs7"
	"github.com/digitorus/timestamp"
	"github.com/in-toto/go-witness/cryptoutil"
)

type TSPTimestamper struct {
	url                string
	hash               crypto.Hash
	requestCertificate bool
}

type TSPTimestamperOption func(*TSPTimestamper)

func TimestampWithUrl(url string) TSPTimestamperOption {
	return func(t *TSPTimestamper) {
		t.url = url
	}
}

func TimestampWithHash(h crypto.Hash) TSPTimestamperOption {
	return func(t *TSPTimestamper) {
		t.hash = h
	}
}

func TimestampWithRequestCertificate(requestCertificate bool) TSPTimestamperOption {
	return func(t *TSPTimestamper) {
		t.requestCertificate = requestCertificate
	}
}

func NewTimestamper(opts ...TSPTimestamperOption) TSPTimestamper {
	t := TSPTimestamper{
		hash:               crypto.SHA256,
		requestCertificate: true,
	}

	for _, opt := range opts {
		opt(&t)
	}

	return t
}

func (t TSPTimestamper) Timestamp(ctx context.Context, r io.Reader) ([]byte, error) {
	tsq, err := timestamp.CreateRequest(r, &timestamp.RequestOptions{
		Hash:         t.hash,
		Certificates: t.requestCertificate,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.url, bytes.NewReader(tsq))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/timestamp-query")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
	default:
		return nil, fmt.Errorf("request to timestamp authority failed: %v", resp.Status)
	}

	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	timestamp, err := timestamp.ParseResponse(bodyBytes)
	if err != nil {
		return nil, err
	}

	return timestamp.RawToken, nil
}

type TSPVerifier struct {
	certChain *x509.CertPool
	hash      crypto.Hash
}

type TSPVerifierOption func(*TSPVerifier)

func VerifyWithCerts(certs []*x509.Certificate) TSPVerifierOption {
	return func(t *TSPVerifier) {
		t.certChain = x509.NewCertPool()
		for _, cert := range certs {
			t.certChain.AddCert(cert)
		}
	}
}

func VerifyWithHash(h crypto.Hash) TSPVerifierOption {
	return func(t *TSPVerifier) {
		t.hash = h
	}
}

func NewVerifier(opts ...TSPVerifierOption) TSPVerifier {
	v := TSPVerifier{
		hash: crypto.SHA256,
	}

	for _, opt := range opts {
		opt(&v)
	}

	return v
}

func (v TSPVerifier) Verify(ctx context.Context, tsrData, signedData io.Reader) (time.Time, error) {
	tsrBytes, err := io.ReadAll(tsrData)
	if err != nil {
		return time.Time{}, err
	}

	ts, err := timestamp.Parse(tsrBytes)
	if err != nil {
		return time.Time{}, err
	}

	hashedData, err := cryptoutil.Digest(signedData, v.hash)
	if err != nil {
		return time.Time{}, err
	}

	if !bytes.Equal(ts.HashedMessage, hashedData) {
		return time.Time{}, fmt.Errorf("signed payload does not match timestamped payload")
	}

	p7, err := pkcs7.Parse(tsrBytes)
	if err != nil {
		return time.Time{}, err
	}

	if err := p7.VerifyWithChain(v.certChain); err != nil {
		return time.Time{}, err
	}

	return ts.Time, nil
}
