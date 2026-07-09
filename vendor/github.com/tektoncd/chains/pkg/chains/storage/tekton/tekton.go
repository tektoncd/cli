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

	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/tektoncd/chains/pkg/chains/annotations"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/storage/api"
	"github.com/tektoncd/chains/pkg/config"
	"knative.dev/pkg/logging"

	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
)

//nolint:revive,exported
const (
	// StorageBackendTekton is the identifier for the Tekton storage backend
	StorageBackendTekton      = "tekton"
	PayloadAnnotationFormat   = annotations.ChainsAnnotationPrefix + "payload-%s"
	SignatureAnnotationFormat = annotations.ChainsAnnotationPrefix + "signature-%s"
	CertAnnotationsFormat     = annotations.ChainsAnnotationPrefix + "cert-%s"
	ChainAnnotationFormat     = annotations.ChainsAnnotationPrefix + "chain-%s"
)

// Backend is a storage backend that stores signed payloads in the TaskRun metadata as an annotation.
// It is stored as base64 encoded JSON.
// Deprecated: use Storer instead.
type Backend struct {
	pipelineclientset versioned.Interface
}

// NewStorageBackend returns a new Tekton StorageBackend that stores signatures on a TaskRun
func NewStorageBackend(ps versioned.Interface) *Backend {
	return &Backend{
		pipelineclientset: ps,
	}
}

// StorePayload implements the Payloader interface.
func (b *Backend) StorePayload(ctx context.Context, obj objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error {
	logger := logging.FromContext(ctx)

	store := &Storer{
		client: b.pipelineclientset,
		key:    opts.ShortKey,
	}
	if _, err := store.Store(ctx, &api.StoreRequest[objects.TektonObject, *intoto.Statement]{
		Object:   obj,
		Artifact: obj,
		// We don't actually use payload - we store the raw bundle values directly.
		Payload: nil,
		Bundle: &signing.Bundle{
			Content:   rawPayload,
			Signature: []byte(signature),
			Cert:      []byte(opts.Cert),
			Chain:     []byte(opts.Chain),
		},
	}); err != nil {
		logger.Errorf("error writing to Tekton object: %w", err)
		return err
	}
	return nil
}

func (b *Backend) Type() string {
	return StorageBackendTekton
}

// retrieveAnnotationValue retrieve the value of an annotation and base64 decode it if needed.
func (b *Backend) retrieveAnnotationValue(ctx context.Context, obj objects.TektonObject, annotationKey string, decode bool) (string, error) {
	logger := logging.FromContext(ctx)
	logger.Infof("Retrieving annotation %q on %s/%s/%s", annotationKey, obj.GetGVK(), obj.GetNamespace(), obj.GetName())

	var annotationValue string
	annotations, err := obj.GetLatestAnnotations(ctx, b.pipelineclientset)
	if err != nil {
		return "", fmt.Errorf("error retrieving the annotation value for the key %q: %w", annotationKey, err)
	}
	val, ok := annotations[annotationKey]

	// Ensure it exists.
	if ok {
		// Decode it if needed.
		if decode {
			decodedAnnotation, err := base64.StdEncoding.DecodeString(val)
			if err != nil {
				return "", fmt.Errorf("error decoding the annotation value for the key %q: %w", annotationKey, err)
			}
			annotationValue = string(decodedAnnotation)
		} else {
			annotationValue = val
		}
	}

	return annotationValue, nil
}

// RetrieveSignatures retrieve the signature stored in the taskrun.
func (b *Backend) RetrieveSignatures(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string][]string, error) {
	logger := logging.FromContext(ctx)
	logger.Infof("Retrieving signature on %s/%s/%s", obj.GetGVK(), obj.GetNamespace(), obj.GetName())
	signatureAnnotation := sigName(opts)
	signature, err := b.retrieveAnnotationValue(ctx, obj, signatureAnnotation, true)
	if err != nil {
		return nil, err
	}
	m := make(map[string][]string)
	m[signatureAnnotation] = []string{signature}
	return m, nil
}

// RetrievePayloads retrieve the payload stored in the taskrun.
func (b *Backend) RetrievePayloads(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string]string, error) {
	logger := logging.FromContext(ctx)
	logger.Infof("Retrieving payload on %s/%s/%s", obj.GetGVK(), obj.GetNamespace(), obj.GetName())
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

type Storer struct {
	client versioned.Interface
	// optional key override. If not specified, the UID of the object is used.
	key string
}

var (
	_ api.Storer[objects.TektonObject, *intoto.Statement] = &Storer{}
)

// Store stores the statement in the TaskRun metadata as an annotation.
func (s *Storer) Store(ctx context.Context, req *api.StoreRequest[objects.TektonObject, *intoto.Statement]) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx)

	obj := req.Object
	logger.Infof("Storing payload on %s/%s/%s", obj.GetGVK(), obj.GetNamespace(), obj.GetName())

	key := s.key
	if key == "" {
		key = string(obj.GetUID())
	}

	storedAnnotations := map[string]string{
		fmt.Sprintf(PayloadAnnotationFormat, key):   base64.StdEncoding.EncodeToString(req.Bundle.Content),
		fmt.Sprintf(SignatureAnnotationFormat, key): base64.StdEncoding.EncodeToString(req.Bundle.Signature),
		fmt.Sprintf(CertAnnotationsFormat, key):     base64.StdEncoding.EncodeToString(req.Bundle.Cert),
		fmt.Sprintf(ChainAnnotationFormat, key):     base64.StdEncoding.EncodeToString(req.Bundle.Chain),
	}

	if err := annotations.AddAnnotations(ctx, obj, s.client, storedAnnotations); err != nil {
		return nil, err
	}

	return &api.StoreResponse{}, nil
}
