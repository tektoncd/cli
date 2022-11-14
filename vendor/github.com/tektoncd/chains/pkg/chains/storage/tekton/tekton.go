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

package tekton

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"

	"github.com/tektoncd/chains/pkg/patch"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
)

const (
	StorageBackendTekton      = "tekton"
	PayloadAnnotationFormat   = "chains.tekton.dev/payload-%s"
	SignatureAnnotationFormat = "chains.tekton.dev/signature-%s"
	CertAnnotationsFormat     = "chains.tekton.dev/cert-%s"
	ChainAnnotationFormat     = "chains.tekton.dev/chain-%s"
)

// Backend is a storage backend that stores signed payloads in the TaskRun metadata as an annotation.
// It is stored as base64 encoded JSON.
type Backend struct {
	pipelineclientset versioned.Interface
	logger            *zap.SugaredLogger
}

// NewStorageBackend returns a new Tekton StorageBackend that stores signatures on a TaskRun
func NewStorageBackend(ps versioned.Interface, logger *zap.SugaredLogger) *Backend {
	return &Backend{
		pipelineclientset: ps,
		logger:            logger,
	}
}

// StorePayload implements the Payloader interface.
func (b *Backend) StorePayload(ctx context.Context, obj objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error {
	b.logger.Infof("Storing payload on %s/%s/%s", obj.GetGVK(), obj.GetNamespace(), obj.GetName())

	// Use patch instead of update to prevent race conditions.
	patchBytes, err := patch.GetAnnotationsPatch(map[string]string{
		// Base64 encode both the signature and the payload
		fmt.Sprintf(PayloadAnnotationFormat, opts.ShortKey):   base64.StdEncoding.EncodeToString(rawPayload),
		fmt.Sprintf(SignatureAnnotationFormat, opts.ShortKey): base64.StdEncoding.EncodeToString([]byte(signature)),
		fmt.Sprintf(CertAnnotationsFormat, opts.ShortKey):     base64.StdEncoding.EncodeToString([]byte(opts.Cert)),
		fmt.Sprintf(ChainAnnotationFormat, opts.ShortKey):     base64.StdEncoding.EncodeToString([]byte(opts.Chain)),
	})
	if err != nil {
		return err
	}

	patchErr := obj.Patch(ctx, b.pipelineclientset, patchBytes)
	if patchErr != nil {
		return patchErr
	}
	return nil
}

func (b *Backend) Type() string {
	return StorageBackendTekton
}

// retrieveAnnotationValue retrieve the value of an annotation and base64 decode it if needed.
func (b *Backend) retrieveAnnotationValue(ctx context.Context, obj objects.TektonObject, annotationKey string, decode bool) (string, error) {
	b.logger.Infof("Retrieving annotation %q on %s/%s/%s", annotationKey, obj.GetGVK(), obj.GetNamespace(), obj.GetName())

	var annotationValue string
	annotations, err := obj.GetLatestAnnotations(ctx, b.pipelineclientset)
	if err != nil {
		return "", fmt.Errorf("error retrieving the annotation value for the key %q: %s", annotationKey, err)
	}
	val, ok := annotations[annotationKey]

	// Ensure it exists.
	if ok {
		// Decode it if needed.
		if decode {
			decodedAnnotation, err := base64.StdEncoding.DecodeString(val)
			if err != nil {
				return "", fmt.Errorf("error decoding the annotation value for the key %q: %s", annotationKey, err)
			}
			annotationValue = string(decodedAnnotation)
		} else {
			annotationValue = val
		}
	}

	return annotationValue, nil
}

// RetrieveSignature retrieve the signature stored in the taskrun.
func (b *Backend) RetrieveSignatures(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string][]string, error) {
	b.logger.Infof("Retrieving signature on %s/%s/%s", obj.GetGVK(), obj.GetNamespace(), obj.GetName())
	signatureAnnotation := sigName(opts)
	signature, err := b.retrieveAnnotationValue(ctx, obj, signatureAnnotation, true)
	if err != nil {
		return nil, err
	}
	m := make(map[string][]string)
	m[signatureAnnotation] = []string{signature}
	return m, nil
}

// RetrievePayload retrieve the payload stored in the taskrun.
func (b *Backend) RetrievePayloads(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string]string, error) {
	b.logger.Infof("Retrieving payload on %s/%s/%s", obj.GetGVK(), obj.GetNamespace(), obj.GetName())
	payloadAnnotation := payloadName(opts)
	payload, err := b.retrieveAnnotationValue(ctx, obj, payloadAnnotation, true)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
	m[payloadAnnotation] = payload
	return m, nil
}

func sigName(opts config.StorageOpts) string {
	return fmt.Sprintf(SignatureAnnotationFormat, opts.ShortKey)
}

func payloadName(opts config.StorageOpts) string {
	return fmt.Sprintf(PayloadAnnotationFormat, opts.ShortKey)
}
