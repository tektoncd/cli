// Copyright © 2022 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package trustedresources

import (
	"bytes"
	"context"
	"crypto"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	cosignsignature "github.com/sigstore/cosign/v2/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/kms"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Verify the crd
func Verify(o metav1.Object, keyfile, kmsKey string) error {
	var verifier signature.Verifier
	var err error
	if keyfile != "" {
		// Load signer from key files
		ctx := context.Background()
		verifier, err = cosignsignature.LoadPublicKey(ctx, keyfile)
		if err != nil {
			return fmt.Errorf("error getting verifier from key file: %v", err)
		}

	}
	if kmsKey != "" {
		ctx := context.Background()
		verifier, err = kms.Get(ctx, kmsKey, crypto.SHA256)
		if err != nil {
			return fmt.Errorf("error getting kms verifier: %v", err)
		}
	}

	// Get raw signature
	a := o.GetAnnotations()
	sigString := a[SignatureAnnotation]
	signatureBytes, err := base64.StdEncoding.DecodeString(sigString)
	if err != nil {
		return err
	}
	delete(a, SignatureAnnotation)
	o.SetAnnotations(a)
	// Verify signature
	return VerifyInterface(o, verifier, signatureBytes)
}

// VerifyInterface get the checksum of json marshalled object and verify it.
func VerifyInterface(
	obj interface{},
	verifier signature.Verifier,
	signature []byte,
) error {
	ts, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	h := sha256.New()
	h.Write(ts)
	return verifier.VerifySignature(bytes.NewReader(signature), bytes.NewReader(h.Sum(nil)))
}
