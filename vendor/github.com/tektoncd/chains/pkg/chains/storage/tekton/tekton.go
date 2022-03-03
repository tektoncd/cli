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
	"strings"

	"github.com/tektoncd/chains/pkg/config"

	"github.com/tektoncd/chains/pkg/patch"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	versioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	pipelienclientset versioned.Interface
	logger            *zap.SugaredLogger
	tr                *v1beta1.TaskRun
}

// NewStorageBackend returns a new Tekton StorageBackend that stores signatures on a TaskRun
func NewStorageBackend(ps versioned.Interface, logger *zap.SugaredLogger, tr *v1beta1.TaskRun) *Backend {
	return &Backend{
		pipelienclientset: ps,
		logger:            logger,
		tr:                tr,
	}
}

// StorePayload implements the Payloader interface.
func (b *Backend) StorePayload(rawPayload []byte, signature string, opts config.StorageOpts) error {
	b.logger.Infof("Storing payload on TaskRun %s/%s", b.tr.Namespace, b.tr.Name)

	// Use patch instead of update to prevent race conditions.
	patchBytes, err := patch.GetAnnotationsPatch(map[string]string{
		// Base64 encode both the signature and the payload
		fmt.Sprintf(PayloadAnnotationFormat, opts.Key):   base64.StdEncoding.EncodeToString(rawPayload),
		fmt.Sprintf(SignatureAnnotationFormat, opts.Key): base64.StdEncoding.EncodeToString([]byte(signature)),
		fmt.Sprintf(CertAnnotationsFormat, opts.Key):     base64.StdEncoding.EncodeToString([]byte(opts.Cert)),
		fmt.Sprintf(ChainAnnotationFormat, opts.Key):     base64.StdEncoding.EncodeToString([]byte(opts.Chain)),
	})
	if err != nil {
		return err
	}
	if _, err := b.pipelienclientset.TektonV1beta1().TaskRuns(b.tr.Namespace).Patch(
		context.TODO(), b.tr.Name, types.MergePatchType, patchBytes, v1.PatchOptions{}); err != nil {
		return err
	}
	return nil
}

func (b *Backend) Type() string {
	return StorageBackendTekton
}

// retrieveAnnotationValue retrieve the value of an annotation and base64 decode it if needed.
func (b *Backend) retrieveAnnotationValue(annotationKey string, decode bool) (string, error) {
	// Retrieve the TaskRun.
	b.logger.Infof("Retrieving annotation %q on TaskRun %s/%s", annotationKey, b.tr.Namespace, b.tr.Name)
	tr, err := b.pipelienclientset.TektonV1beta1().TaskRuns(b.tr.Namespace).Get(context.TODO(), b.tr.Name, v1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error retrieving taskrun: %s", err)
	}

	// Retrieve the annotation.
	var annotationValue string
	rawAnnotationValue, exists := tr.Annotations[annotationKey]

	// Ensure it exists.
	if exists {
		// Decode it if needed.
		if decode {
			decodedAnnotation, err := base64.StdEncoding.DecodeString(rawAnnotationValue)
			if err != nil {
				return "", fmt.Errorf("error decoding the annotation value for the key %q: %s", annotationKey, err)
			}
			annotationValue = string(decodedAnnotation)
		} else {
			annotationValue = rawAnnotationValue
		}
	}

	return annotationValue, nil
}

// RetrieveSignature retrieve the signature stored in the taskrun.
func (b *Backend) RetrieveSignatures(opts config.StorageOpts) (map[string][]string, error) {
	b.logger.Infof("Retrieving signature on TaskRun %s/%s", b.tr.Namespace, b.tr.Name)
	signatureAnnotation := b.SigName(opts)
	signature, err := b.retrieveAnnotationValue(signatureAnnotation, true)
	if err != nil {
		return nil, err
	}

	m := make(map[string][]string)
	for _, res := range b.tr.Status.TaskRunResults {
		if strings.HasSuffix(res.Name, "IMAGE_URL") {
			m[signatureAnnotation] = []string{signature}
			break
		}
	}
	return m, nil
}

// RetrievePayload retrieve the payload stored in the taskrun.
func (b *Backend) RetrievePayloads(opts config.StorageOpts) (map[string]string, error) {
	b.logger.Infof("Retrieving payload on TaskRun %s/%s", b.tr.Namespace, b.tr.Name)
	payloadAnnotation := b.PayloadName(opts)
	payload, err := b.retrieveAnnotationValue(payloadAnnotation, true)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for _, res := range b.tr.Status.TaskRunResults {
		if strings.HasSuffix(res.Name, "IMAGE_URL") {
			m[payloadAnnotation] = payload
			break
		}
	}

	return m, nil
}

func (b *Backend) SigName(opts config.StorageOpts) string {
	return fmt.Sprintf(SignatureAnnotationFormat, opts.Key)
}

func (b *Backend) PayloadName(opts config.StorageOpts) string {
	return fmt.Sprintf(PayloadAnnotationFormat, opts.Key)
}

func (b *Backend) CertName(opts config.StorageOpts) string {
	return fmt.Sprintf(CertAnnotationsFormat, opts.Key)
}

func (b *Backend) ChainName(opts config.StorageOpts) string {
	return fmt.Sprintf(ChainAnnotationFormat, opts.Key)
}
