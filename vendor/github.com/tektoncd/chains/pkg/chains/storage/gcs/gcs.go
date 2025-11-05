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

package gcs

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"knative.dev/pkg/logging"

	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/storage/api"
	"github.com/tektoncd/chains/pkg/config"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	StorageBackendGCS = "gcs"
	// taskrun-$namespace-$name/$key.<type>
	SignatureNameFormatTaskRun = "taskrun-%s-%s/%s.signature"
	PayloadNameFormatTaskRun   = "taskrun-%s-%s/%s.payload"
	// pipelinerun-$namespace-$name/$key.<type>
	SignatureNameFormatPipelineRun = "pipelinerun-%s-%s/%s.signature"
	PayloadNameFormatPipelineRun   = "pipelinerun-%s-%s/%s.payload"
)

// Backend is a storage backend that stores signed payloads in the TaskRun metadata as an annotation.
// It is stored as base64 encoded JSON.
// Deprecated: Use TaskRunStorer instead.
type Backend struct {
	writer gcsWriter
	reader gcsReader
	cfg    config.Config
}

// NewStorageBackend returns a new Tekton StorageBackend that stores signatures on a TaskRun
func NewStorageBackend(ctx context.Context, cfg config.Config) (*Backend, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	bucket := cfg.Storage.GCS.Bucket
	return &Backend{
		writer: &writer{client: client, bucket: bucket},
		reader: &reader{client: client, bucket: bucket},
		cfg:    cfg,
	}, nil
}

// StorePayload implements the storage.Backend interface.
func (b *Backend) StorePayload(ctx context.Context, obj objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error {
	logger := logging.FromContext(ctx)

	if tr, isTaskRun := obj.GetObject().(*v1.TaskRun); isTaskRun {
		store := &TaskRunStorer{
			writer: b.writer,
			key:    opts.ShortKey,
		}
		if _, err := store.Store(ctx, &api.StoreRequest[*v1.TaskRun, *in_toto.Statement]{
			Object:   obj,
			Artifact: tr,
			// We don't actually use payload - we store the raw bundle values directly.
			Payload: nil,
			Bundle: &signing.Bundle{
				Content:   rawPayload,
				Signature: []byte(signature),
				Cert:      []byte(opts.Cert),
				Chain:     []byte(opts.Chain),
			},
		}); err != nil {
			logger.Errorf("error writing to GCS: %w", err)
			return err
		}
	} else if pr, isPipelineRun := obj.GetObject().(*v1.PipelineRun); isPipelineRun {
		store := &PipelineRunStorer{
			writer: b.writer,
			key:    opts.ShortKey,
		}
		if _, err := store.Store(ctx, &api.StoreRequest[*v1.PipelineRun, *in_toto.Statement]{ //nolint:staticcheck
			Object:   obj,
			Artifact: pr,
			// We don't actually use payload - we store the raw bundle values directly.
			Payload: nil,
			Bundle: &signing.Bundle{
				Content:   rawPayload,
				Signature: []byte(signature),
				Cert:      []byte(opts.Cert),
				Chain:     []byte(opts.Chain),
			},
		}); err != nil {
			logger.Errorf("error writing to GCS: %w", err)
			return err
		}
	} else {
		return fmt.Errorf("type %T not supported - supported types: [*v1.TaskRun, *v1.PipelineRun]", obj.GetObject())
	}
	return nil
}

func (b *Backend) Type() string {
	return StorageBackendGCS
}

type gcsWriter interface {
	GetWriter(ctx context.Context, object string) io.WriteCloser
}

type writer struct {
	client *storage.Client
	bucket string
}

type gcsReader interface {
	GetReader(ctx context.Context, object string) (io.ReadCloser, error)
}

type reader struct {
	client *storage.Client
	bucket string
}

func (r *writer) GetWriter(ctx context.Context, object string) io.WriteCloser {
	return r.client.Bucket(r.bucket).Object(object).NewWriter(ctx)
}

func (r *reader) GetReader(ctx context.Context, object string) (io.ReadCloser, error) {
	return r.client.Bucket(r.bucket).Object(object).NewReader(ctx)
}

func (b *Backend) RetrieveSignatures(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string][]string, error) {
	var object string

	switch t := obj.GetObject().(type) {
	case *v1.TaskRun:
		object = taskRunSigNameV1(t, opts)
	case *v1.PipelineRun:
		object = pipelineRunSignameV1(t, opts)
	default:
		return nil, fmt.Errorf("unsupported TektonObject type: %T", t)
	}

	signature, err := b.retrieveObject(ctx, object)
	if err != nil {
		return nil, err
	}

	m := make(map[string][]string)
	m[object] = []string{signature}
	return m, nil
}

func (b *Backend) RetrievePayloads(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string]string, error) {
	var object string

	switch t := obj.GetObject().(type) {
	case *v1.TaskRun:
		object = taskRunPayloadNameV1(t, opts)
	case *v1.PipelineRun:
		object = pipelineRunPayloadNameV1(t, opts)
	default:
		return nil, fmt.Errorf("unsupported TektonObject type: %T", t)
	}

	payload, err := b.retrieveObject(ctx, object)
	if err != nil {
		return nil, err
	}

	m := make(map[string]string)
	m[object] = payload
	return m, nil
}

func (b *Backend) retrieveObject(ctx context.Context, object string) (string, error) {
	reader, err := b.reader.GetReader(ctx, object)
	if err != nil {
		return "", err
	}

	defer reader.Close()
	payload, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func taskRunSigNameV1(tr *v1.TaskRun, opts config.StorageOpts) string {
	return fmt.Sprintf(SignatureNameFormatTaskRun, tr.Namespace, tr.Name, opts.ShortKey)
}

func taskRunPayloadNameV1(tr *v1.TaskRun, opts config.StorageOpts) string {
	return fmt.Sprintf(PayloadNameFormatTaskRun, tr.Namespace, tr.Name, opts.ShortKey)
}

func pipelineRunSignameV1(pr *v1.PipelineRun, opts config.StorageOpts) string {
	return fmt.Sprintf(SignatureNameFormatPipelineRun, pr.Namespace, pr.Name, opts.ShortKey)
}

func pipelineRunPayloadNameV1(pr *v1.PipelineRun, opts config.StorageOpts) string {
	return fmt.Sprintf(PayloadNameFormatPipelineRun, pr.Namespace, pr.Name, opts.ShortKey)
}

//nolint:staticcheck
var (
	_ api.Storer[*v1.TaskRun, *in_toto.Statement]     = &TaskRunStorer{}
	_ api.Storer[*v1.PipelineRun, *in_toto.Statement] = &PipelineRunStorer{}
)

// TaskRunStorer stores TaskRuns in GCS.
type TaskRunStorer struct {
	writer gcsWriter

	// Optional key to store objects as. If not set, the object UID will be used.
	// The resulting name will look like: $bucket/taskrun-$namespace-$name/$key.signature
	key string
}

// Store stores the TaskRun chains information in GCS
//
//nolint:staticcheck
func (s *TaskRunStorer) Store(ctx context.Context, req *api.StoreRequest[*v1.TaskRun, *in_toto.Statement]) (*api.StoreResponse, error) {
	tr := req.Artifact
	key := s.key
	if key == "" {
		key = string(tr.GetUID())
	}
	prefix := fmt.Sprintf("%s-%s-%s/%s", "taskrun", tr.GetNamespace(), tr.GetName(), key)

	return store(ctx, s.writer, prefix,
		req.Bundle.Signature, req.Bundle.Content, req.Bundle.Cert, req.Bundle.Chain)
}

// PipelineRunStorer stores PipelineRuns in GCS.
type PipelineRunStorer struct {
	writer gcsWriter

	// Optional key to store objects as. If not set, the object UID will be used.
	// The resulting name will look like: $bucket/pipelinerun-$namespace-$name/$key.signature
	key string
}

// Store stores the PipelineRun chains information in GCS
//
//nolint:staticcheck
func (s *PipelineRunStorer) Store(ctx context.Context, req *api.StoreRequest[*v1.PipelineRun, *in_toto.Statement]) (*api.StoreResponse, error) {
	pr := req.Artifact
	key := s.key
	if key == "" {
		key = string(pr.GetUID())
	}
	prefix := fmt.Sprintf("%s-%s-%s/%s", "pipelinerun", pr.GetNamespace(), pr.GetName(), key)

	return store(ctx, s.writer, prefix,
		req.Bundle.Signature, req.Bundle.Content, req.Bundle.Cert, req.Bundle.Chain)
}

func store(ctx context.Context, writer gcsWriter, prefix string,
	signature, content, cert, chain []byte) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx)

	// Write signature
	sigName := prefix + ".signature"
	logger.Infof("Storing signature at %s", sigName)
	if _, err := write(ctx, writer, sigName, signature); err != nil {
		return nil, err
	}

	// Write payload
	payloadName := prefix + ".payload"
	if _, err := write(ctx, writer, payloadName, content); err != nil {
		return nil, err
	}

	// Only write cert+chain if it is present.
	if cert == nil {
		return nil, nil
	}
	certName := prefix + ".cert"
	if _, err := write(ctx, writer, certName, cert); err != nil {
		return nil, err
	}

	chainName := prefix + ".chain"
	if _, err := write(ctx, writer, chainName, chain); err != nil {
		return nil, err
	}

	return &api.StoreResponse{}, nil
}

func write(ctx context.Context, client gcsWriter, name string, content []byte) (int, error) {
	w := client.GetWriter(ctx, name)
	defer w.Close()
	return w.Write(content)
}
