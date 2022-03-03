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

package chains

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sigstore/cosign/pkg/cosign"
	rc "github.com/sigstore/rekor/pkg/client"
	"github.com/sigstore/rekor/pkg/generated/client"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
)

const (
	RekorAnnotation = "chains.tekton.dev/transparency-upload"
)

var (
	// using cosign default
	timeout = 30 * time.Second
)

type rekor struct {
	c      *client.Rekor
	logger *zap.SugaredLogger
}

type rekorClient interface {
	UploadTlog(ctx context.Context, signer signing.Signer, signature, rawPayload []byte, cert, payloadFormat string) (*models.LogEntryAnon, error)
}

func (r *rekor) UploadTlog(ctx context.Context, signer signing.Signer, signature, rawPayload []byte, cert, payloadFormat string) (*models.LogEntryAnon, error) {
	pkoc, err := publicKeyOrCert(signer, cert)
	if err != nil {
		return nil, errors.Wrap(err, "public key or cert")
	}
	if payloadFormat == "in-toto" || payloadFormat == "tekton-provenance" {
		return cosign.TLogUploadInTotoAttestation(ctx, r.c, signature, pkoc)
	}
	return cosign.TLogUpload(ctx, r.c, signature, rawPayload, pkoc)
}

// return the cert if we have it, otherwise return public key
func publicKeyOrCert(signer signing.Signer, cert string) ([]byte, error) {
	if cert != "" {
		return []byte(cert), nil
	}
	pub, err := signer.PublicKey()
	if err != nil {
		return nil, errors.Wrap(err, "getting public key")
	}
	pem, err := cryptoutils.MarshalPublicKeyToPEM(pub)
	if err != nil {
		return nil, errors.Wrap(err, "key to pem")
	}
	return pem, nil
}

// for testing
var getRekor = func(url string, l *zap.SugaredLogger) (rekorClient, error) {
	rekorClient, err := rc.GetRekorClient(url)
	if err != nil {
		return nil, err
	}
	return &rekor{
		c:      rekorClient,
		logger: l,
	}, nil
}

func shouldUploadTlog(cfg config.Config, tr *v1beta1.TaskRun) bool {
	// if transparency isn't enabled, return false
	if !cfg.Transparency.Enabled {
		return false
	}
	// if transparency is enabled and verification is disabled, return true
	if !cfg.Transparency.VerifyAnnotation {
		return true
	}

	// Already uploaded, don't do it again
	if _, ok := tr.Annotations[ChainsTransparencyAnnotation]; ok {
		return false
	}
	// verify the annotation
	return tr.Annotations[RekorAnnotation] == "true"
}
