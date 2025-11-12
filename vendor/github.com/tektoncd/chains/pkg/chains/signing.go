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
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/signing/kms"
	"github.com/tektoncd/chains/pkg/chains/signing/x509"
	"github.com/tektoncd/chains/pkg/chains/storage"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/metrics"
	versioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/util/sets"
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

	Recorder metrics.Recorder
}

func allSigners(ctx context.Context, sp string, cfg config.Config) map[string]signing.Signer {
	l := logging.FromContext(ctx)
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
			signer, err := x509.NewSigner(ctx, sp, cfg)
			if err != nil {
				l.Warnf("error configuring x509 signer: %s", err)
				continue
			}
			all[s] = signer
		case signing.TypeKMS:
			signer, err := kms.NewSigner(ctx, cfg.Signers.KMS)
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
func getSignableTypes(ctx context.Context, obj objects.TektonObject) ([]artifacts.Signable, error) {
	var types []artifacts.Signable

	if obj.SupportsTaskRunArtifact() {
		types = append(types, &artifacts.TaskRunArtifact{})
	}

	if obj.SupportsPipelineRunArtifact() {
		types = append(types, &artifacts.PipelineRunArtifact{})
	}

	if obj.SupportsOCIArtifact() {
		types = append(types, &artifacts.OCIArtifact{})
	}

	if len(types) == 0 {
		return nil, fmt.Errorf("no signable artifacts found for %v", obj)
	}

	return types, nil
}

// Sign TaskRun and PipelineRun objects, as well as generates attestations for each.
// Follows process of extract payload, sign payload, store payload and signature.
func (o *ObjectSigner) Sign(ctx context.Context, tektonObj objects.TektonObject) error {
	cfg := *config.FromContext(ctx)
	logger := logging.FromContext(ctx)

	signableTypes, err := getSignableTypes(ctx, tektonObj)
	if err != nil {
		return err
	}

	signers := allSigners(ctx, o.SecretPath, cfg)

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
		objects := signableType.ExtractObjects(ctx, tektonObj)
		// Go through each object one at a time.
		for _, obj := range objects {

			payload, err := payloader.CreatePayload(ctx, obj)
			if err != nil {
				logger.Error(err)
				o.recordError(ctx, signableType.Type(), metrics.PayloadCreationError)
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
				wrapped, err := signing.Wrap(signer)
				if err != nil {
					return err
				}
				logger.Infof("Using wrapped envelope signer for %s", payloader.Type())
				signer = wrapped
			}

			logger.Infof("Signing object with %s", signerType)
			rawPayload, err := getRawPayload(payload)
			if err != nil {
				logger.Warnf("Unable to marshal payload for %s: %v", signerType, err)
				o.recordError(ctx, signableType.Type(), metrics.MarshalPayloadError)
				continue
			}

			signature, err := signer.SignMessage(bytes.NewReader(rawPayload))
			if err != nil {
				logger.Error(err)
				o.recordError(ctx, signableType.Type(), metrics.SigningError)
				continue
			}
			measureMetrics(ctx, metrics.SignedMessagesCount, o.Recorder)

			// Now store those!
			for _, backend := range sets.List[string](signableType.StorageBackend(cfg)) {
				b, ok := o.Backends[backend]
				if !ok {
					backendErr := fmt.Errorf("could not find backend '%s' in configured backends (%v) while trying sign: %s/%s", backend, maps.Keys(o.Backends), tektonObj.GetKindName(), tektonObj.GetName())
					logger.Error(backendErr)
					o.recordError(ctx, signableType.Type(), metrics.StorageError)
					merr = multierror.Append(merr, backendErr)
					continue
				}

				storageOpts := config.StorageOpts{
					ShortKey:      signableType.ShortKey(obj),
					FullKey:       signableType.FullKey(obj),
					Cert:          signer.Cert(),
					Chain:         signer.Chain(),
					PayloadFormat: payloadFormat,
				}
				if err := b.StorePayload(ctx, tektonObj, rawPayload, string(signature), storageOpts); err != nil {
					logger.Error(err)
					o.recordError(ctx, signableType.Type(), metrics.StorageError)
					merr = multierror.Append(merr, err)
				} else {
					measureMetrics(ctx, metrics.SignsStoredCount, o.Recorder)
				}
			}

			if shouldUploadTlog(cfg, tektonObj) {
				rekorClient, err := getRekor(cfg.Transparency.URL)
				if err != nil {
					return err
				}

				entry, err := rekorClient.UploadTlog(ctx, signer, signature, rawPayload, signer.Cert(), string(payloadFormat))
				if err != nil {
					logger.Warnf("error uploading entry to tlog: %v", err)
					o.recordError(ctx, signableType.Type(), metrics.TlogError)
					merr = multierror.Append(merr, err)
				} else {
					logger.Infof("Uploaded entry to %s with index %d", cfg.Transparency.URL, *entry.LogIndex)
					extraAnnotations[ChainsTransparencyAnnotation] = fmt.Sprintf("%s/api/v1/log/entries?logIndex=%d", cfg.Transparency.URL, *entry.LogIndex)
					measureMetrics(ctx, metrics.PayloadUploadeCount, o.Recorder)
				}
			}

		}
		if merr.ErrorOrNil() != nil {
			if retryErr := HandleRetry(ctx, tektonObj, o.Pipelineclientset, extraAnnotations); retryErr != nil {
				logger.Warnf("error handling retry: %v", retryErr)
				merr = multierror.Append(merr, retryErr)
			}
			return merr
		}
	}

	// Now mark the TektonObject as signed
	if err := MarkSigned(ctx, tektonObj, o.Pipelineclientset, extraAnnotations); err != nil {
		return err
	}
	measureMetrics(ctx, metrics.MarkedAsSignedCount, o.Recorder)
	return nil
}

func measureMetrics(ctx context.Context, metrictype metrics.Metric, mtr metrics.Recorder) {
	if mtr != nil {
		mtr.RecordCountMetrics(ctx, metrictype)
	}
}

// recordError abstracts the check and calls RecordErrorMetric if appropriate.
func (o *ObjectSigner) recordError(ctx context.Context, kind string, errType metrics.MetricErrorType) {
	shouldRecordError := kind == "TaskRunArtifact" || kind == "PipelineRunArtifact"
	if shouldRecordError && o.Recorder != nil {
		o.Recorder.RecordErrorMetric(ctx, errType)
	}
}

func HandleRetry(ctx context.Context, obj objects.TektonObject, ps versioned.Interface, annotations map[string]string) error {
	if RetryAvailable(obj) {
		return AddRetry(ctx, obj, ps, annotations)
	}
	return MarkFailed(ctx, obj, ps, annotations)
}

// getRawPayload returns the payload as a json string. If the given payload is a intoto.Statement type, protojson.Marshal
// is used to get the proper labels/field names in the resulting json.
func getRawPayload(payload interface{}) ([]byte, error) {
	switch payloadObj := payload.(type) {
	case intoto.Statement:
		return protojson.Marshal(&payloadObj)
	case *intoto.Statement:
		if payloadObj == nil {
			return json.Marshal(payload)
		}
		return protojson.Marshal(payloadObj)
	default:
		return json.Marshal(payload)
	}
}
