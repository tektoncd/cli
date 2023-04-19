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
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/signing/kms"
	"github.com/tektoncd/chains/pkg/chains/signing/x509"
	"github.com/tektoncd/chains/pkg/chains/storage"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	versioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
)

type Signer interface {
	Sign(ctx context.Context, obj objects.TektonObject) error
}

type ObjectSigner struct {
	// Backends: store payload and signature
	// The keys are different storage option's name. {docdb, gcs, grafeas, oci, tekton}
	// The values are the actual storage backends that will be used to store and retrieve provenance.
	Backends          map[string]storage.Backend
	SecretPath        string
	Pipelineclientset versioned.Interface
}

func allSigners(ctx context.Context, sp string, cfg config.Config, l *zap.SugaredLogger) map[string]signing.Signer {
	all := map[string]signing.Signer{}
	neededSigners := map[string]struct{}{
		cfg.Artifacts.OCI.Signer:          {},
		cfg.Artifacts.TaskRuns.Signer:     {},
		cfg.Artifacts.PipelineRuns.Signer: {},
	}

	for _, s := range signing.AllSigners {
		if _, ok := neededSigners[s]; !ok {
			continue
		}
		switch s {
		case signing.TypeX509:
			signer, err := x509.NewSigner(ctx, sp, cfg, l)
			if err != nil {
				l.Warnf("error configuring x509 signer: %s", err)
				continue
			}
			all[s] = signer
		case signing.TypeKMS:
			signer, err := kms.NewSigner(ctx, cfg.Signers.KMS, l)
			if err != nil {
				l.Warnf("error configuring kms signer with config %v: %s", cfg.Signers.KMS, err)
				continue
			}
			all[s] = signer
		default:
			// This should never happen, so panic
			l.Panicf("unsupported signer: %s", s)
		}
	}
	return all
}

// TODO: Hook this up to config.
func getSignableTypes(obj objects.TektonObject, logger *zap.SugaredLogger) ([]artifacts.Signable, error) {
	switch v := obj.GetObject().(type) {
	case *v1beta1.TaskRun:
		return []artifacts.Signable{
			&artifacts.TaskRunArtifact{Logger: logger},
			&artifacts.OCIArtifact{Logger: logger},
		}, nil
	case *v1beta1.PipelineRun:
		return []artifacts.Signable{
			&artifacts.PipelineRunArtifact{Logger: logger},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported type of object to be signed: %s", v)
	}
}

// Signs TaskRun and PipelineRun objects, as well as generates attesations for each
// Follows process of extract payload, sign payload, store payload and signature
func (o *ObjectSigner) Sign(ctx context.Context, tektonObj objects.TektonObject) error {
	cfg := *config.FromContext(ctx)
	logger := logging.FromContext(ctx)

	signableTypes, err := getSignableTypes(tektonObj, logger)
	if err != nil {
		return err
	}

	signers := allSigners(ctx, o.SecretPath, cfg, logger)

	var merr *multierror.Error
	extraAnnotations := map[string]string{}
	for _, signableType := range signableTypes {
		if !signableType.Enabled(cfg) {
			continue
		}
		payloadFormat := signableType.PayloadFormat(cfg)
		// Find the right payload format and format the object
		payloader, err := formats.GetPayloader(payloadFormat, cfg)
		if err != nil {
			logger.Warnf("Format %s configured for %s: %v was not found", payloadFormat, tektonObj.GetGVK(), signableType.Type())
			continue
		}

		// Extract all the "things" to be signed.
		// We might have a few of each type (several binaries, or images)
		objects := signableType.ExtractObjects(tektonObj)

		// Go through each object one at a time.
		for _, obj := range objects {

			payload, err := payloader.CreatePayload(ctx, obj)
			if err != nil {
				logger.Error(err)
				continue
			}
			logger.Infof("Created payload of type %s for %s %s/%s", string(payloadFormat), tektonObj.GetGVK(), tektonObj.GetNamespace(), tektonObj.GetName())

			// Sign it!
			signerType := signableType.Signer(cfg)
			signer, ok := signers[signerType]
			if !ok {
				logger.Warnf("No signer %s configured for %s", signerType, signableType.Type())
				continue
			}

			if payloader.Wrap() {
				wrapped, err := signing.Wrap(ctx, signer)
				if err != nil {
					return err
				}
				logger.Infof("Using wrapped envelope signer for %s", payloader.Type())
				signer = wrapped
			}

			logger.Infof("Signing object with %s", signerType)
			rawPayload, err := json.Marshal(payload)
			if err != nil {
				logger.Warnf("Unable to marshal payload: %v", signerType, obj)
				continue
			}

			signature, err := signer.SignMessage(bytes.NewReader(rawPayload))
			if err != nil {
				logger.Error(err)
				continue
			}

			// Now store those!
			for _, backend := range signableType.StorageBackend(cfg).List() {
				b := o.Backends[backend]
				storageOpts := config.StorageOpts{
					ShortKey:      signableType.ShortKey(obj),
					FullKey:       signableType.FullKey(obj),
					Cert:          signer.Cert(),
					Chain:         signer.Chain(),
					PayloadFormat: payloadFormat,
				}
				if err := b.StorePayload(ctx, tektonObj, rawPayload, string(signature), storageOpts); err != nil {
					logger.Error(err)
					merr = multierror.Append(merr, err)
				}
			}

			if shouldUploadTlog(cfg, tektonObj) {
				rekorClient, err := getRekor(cfg.Transparency.URL, logger)
				if err != nil {
					return err
				}

				entry, err := rekorClient.UploadTlog(ctx, signer, signature, rawPayload, signer.Cert(), string(payloadFormat))
				if err != nil {
					logger.Warnf("error uploading entry to tlog: %v", err)
					merr = multierror.Append(merr, err)
				} else {
					logger.Infof("Uploaded entry to %s with index %d", cfg.Transparency.URL, *entry.LogIndex)

					extraAnnotations[ChainsTransparencyAnnotation] = fmt.Sprintf("%s/api/v1/log/entries?logIndex=%d", cfg.Transparency.URL, *entry.LogIndex)
				}
			}

		}
		if merr.ErrorOrNil() != nil {
			if err := HandleRetry(ctx, tektonObj, o.Pipelineclientset, extraAnnotations); err != nil {
				logger.Warnf("error handling retry: %v", err)
				merr = multierror.Append(merr, err)
			}
			return merr
		}
	}

	// Now mark the TektonObject as signed
	if err := MarkSigned(ctx, tektonObj, o.Pipelineclientset, extraAnnotations); err != nil {
		return err
	}

	return nil
}

func HandleRetry(ctx context.Context, obj objects.TektonObject, ps versioned.Interface, annotations map[string]string) error {
	if RetryAvailable(obj) {
		return AddRetry(ctx, obj, ps, annotations)
	}
	return MarkFailed(ctx, obj, ps, annotations)
}
