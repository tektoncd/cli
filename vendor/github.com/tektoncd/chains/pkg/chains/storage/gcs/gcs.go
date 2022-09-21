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
	"io/ioutil"

	"cloud.google.com/go/storage"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
)

const (
	StorageBackendGCS = "gcs"
	// taskrun-$namespace-$name/$key.<type>
	SignatureNameFormat = "taskrun-%s-%s/%s.signature"
	PayloadNameFormat   = "taskrun-%s-%s/%s.payload"
	CertNameFormat      = "taskrun-%s-%s/%s.cert"
	ChainNameFormat     = "taskrun-%s-%s/%s.chain"
)

// Backend is a storage backend that stores signed payloads in the TaskRun metadata as an annotation.
// It is stored as base64 encoded JSON.
type Backend struct {
	logger *zap.SugaredLogger
	writer gcsWriter
	reader gcsReader
	cfg    config.Config
}

// NewStorageBackend returns a new Tekton StorageBackend that stores signatures on a TaskRun
func NewStorageBackend(ctx context.Context, logger *zap.SugaredLogger, cfg config.Config) (*Backend, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	bucket := cfg.Storage.GCS.Bucket
	return &Backend{
		logger: logger,
		writer: &writer{client: client, bucket: bucket},
		reader: &reader{client: client, bucket: bucket},
		cfg:    cfg,
	}, nil
}

// StorePayload implements the storage.Backend interface.
func (b *Backend) StorePayload(ctx context.Context, obj objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error {
	// TODO: Handle unsupported type gracefully
	tr := obj.GetObject().(*v1beta1.TaskRun)
	// We need multiple objects: the signature and the payload. We want to make these unique to the UID, but easy to find based on the
	// name/namespace as well.
	// $bucket/taskrun-$namespace-$name/$key.signature
	// $bucket/taskrun-$namespace-$name/$key.payload
	sigName := sigName(tr, opts)
	b.logger.Infof("Storing signature at %s", sigName)

	sigObj := b.writer.GetWriter(ctx, sigName)
	if _, err := sigObj.Write([]byte(signature)); err != nil {
		return err
	}
	if err := sigObj.Close(); err != nil {
		return err
	}

	payloadObj := b.writer.GetWriter(ctx, payloadName(tr, opts))
	defer payloadObj.Close()
	if _, err := payloadObj.Write(rawPayload); err != nil {
		return err
	}
	if err := payloadObj.Close(); err != nil {
		return err
	}

	if opts.Cert == "" {
		return nil
	}
	certName := certName(tr, opts)
	certObj := b.writer.GetWriter(ctx, certName)
	defer certObj.Close()
	if _, err := certObj.Write([]byte(opts.Cert)); err != nil {
		return err
	}
	if err := certObj.Close(); err != nil {
		return err
	}

	chainName := chainName(tr, opts)
	chainObj := b.writer.GetWriter(ctx, chainName)
	defer chainObj.Close()
	if _, err := chainObj.Write([]byte(opts.Chain)); err != nil {
		return err
	}
	if err := chainObj.Close(); err != nil {
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
	payload, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func sigName(tr *v1beta1.TaskRun, opts config.StorageOpts) string {
	return fmt.Sprintf(SignatureNameFormat, tr.Namespace, tr.Name, opts.Key)
}

func payloadName(tr *v1beta1.TaskRun, opts config.StorageOpts) string {
	return fmt.Sprintf(PayloadNameFormat, tr.Namespace, tr.Name, opts.Key)
}

func certName(tr *v1beta1.TaskRun, opts config.StorageOpts) string {
	return fmt.Sprintf(CertNameFormat, tr.Namespace, tr.Name, opts.Key)
}

func chainName(tr *v1beta1.TaskRun, opts config.StorageOpts) string {
	return fmt.Sprintf(ChainNameFormat, tr.Namespace, tr.Name, opts.Key)
}
