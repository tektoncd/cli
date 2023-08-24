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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const (
	StorageBackendGCS = "gcs"
	// taskrun-$namespace-$name/$key.<type>
	SignatureNameFormat = "taskrun-%s-%s/%s.signature"
	PayloadNameFormat   = "taskrun-%s-%s/%s.payload"
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

	// TODO(https://github.com/tektoncd/chains/issues/852): Support PipelineRuns
	tr, ok := obj.GetObject().(*v1beta1.TaskRun)
	if !ok {
		return fmt.Errorf("type %T not supported - supported types: [*v1beta1.TaskRun]", obj.GetObject())
	}

	store := &TaskRunStorer{
		writer: b.writer,
		key:    opts.ShortKey,
	}
	if _, err := store.Store(ctx, &api.StoreRequest[*v1beta1.TaskRun, *in_toto.Statement]{
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
	// TODO: Handle unsupported type gracefully
	tr := obj.GetObject().(*v1beta1.TaskRun)
	object := sigName(tr, opts)
	signature, err := b.retrieveObject(ctx, object)
	if err != nil {
		return nil, err
	}

	m := make(map[string][]string)
	m[object] = []string{signature}
	return m, nil
}

func (b *Backend) RetrievePayloads(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string]string, error) {
	// TODO: Handle unsupported type gracefully
	tr := obj.GetObject().(*v1beta1.TaskRun)
	object := payloadName(tr, opts)
	m := make(map[string]string)
	payload, err := b.retrieveObject(ctx, object)
	if err != nil {
		return nil, err
	}

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

func sigName(tr *v1beta1.TaskRun, opts config.StorageOpts) string {
	return fmt.Sprintf(SignatureNameFormat, tr.Namespace, tr.Name, opts.ShortKey)
}

func payloadName(tr *v1beta1.TaskRun, opts config.StorageOpts) string {
	return fmt.Sprintf(PayloadNameFormat, tr.Namespace, tr.Name, opts.ShortKey)
}

var (
	_ api.Storer[*v1beta1.TaskRun, *in_toto.Statement] = &TaskRunStorer{}
)

// TaskRunStorer stores TaskRuns in GCS.
// TODO(https://github.com/tektoncd/chains/issues/852): implement PipelineRun support (nothing in here is particularly TaskRun specific, but needs tests).
type TaskRunStorer struct {
	writer gcsWriter

	// Optional key to store objects as. If not set, the object UID will be used.
	// The resulting name will look like: $bucket/taskrun-$namespace-$name/$key.signature
	key string
}

// Store stores the
func (s *TaskRunStorer) Store(ctx context.Context, req *api.StoreRequest[*v1beta1.TaskRun, *in_toto.Statement]) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx)

	tr := req.Artifact
	// We need multiple objects: the signature and the payload. We want to make these unique to the UID, but easy to find based on the
	// name/namespace as well.
	// $bucket/taskrun-$namespace-$name/$key.signature
	// $bucket/taskrun-$namespace-$name/$key.payload
	key := s.key
	if key == "" {
		key = string(tr.GetUID())
	}
	prefix := fmt.Sprintf("taskrun-%s-%s/%s", tr.GetNamespace(), tr.GetName(), key)

	// Write signature
	sigName := prefix + ".signature"
	logger.Infof("Storing signature at %s", sigName)
	if _, err := write(ctx, s.writer, sigName, req.Bundle.Signature); err != nil {
		return nil, err
	}

	// Write payload
	if _, err := write(ctx, s.writer, prefix+".payload", req.Bundle.Content); err != nil {
		return nil, err
	}

	// Only write cert+chain if it is present.
	if req.Bundle.Cert == nil {
		return nil, nil
	}
	if _, err := write(ctx, s.writer, prefix+".cert", req.Bundle.Cert); err != nil {
		return nil, err
	}
	if _, err := write(ctx, s.writer, prefix+".chain", req.Bundle.Chain); err != nil {
		return nil, err
	}

	return &api.StoreResponse{}, nil
}

func write(ctx context.Context, client gcsWriter, name string, content []byte) (int, error) {
	w := client.GetWriter(ctx, name)
	defer w.Close()
	return w.Write(content)
}
