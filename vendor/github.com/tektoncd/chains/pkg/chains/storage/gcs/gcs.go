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
	tr     *v1beta1.TaskRun
	writer gcsWriter
	reader gcsReader
	cfg    config.Config
}

// NewStorageBackend returns a new Tekton StorageBackend that stores signatures on a TaskRun
func NewStorageBackend(logger *zap.SugaredLogger, tr *v1beta1.TaskRun, cfg config.Config) (*Backend, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	bucket := cfg.Storage.GCS.Bucket
	return &Backend{
		logger: logger,
		tr:     tr,
		writer: &writer{client: client, bucket: bucket},
		reader: &reader{client: client, bucket: bucket},
		cfg:    cfg,
	}, nil
}

// StorePayload implements the storage.Backend interface.
func (b *Backend) StorePayload(rawPayload []byte, signature string, opts config.StorageOpts) error {
	// We need multiple objects: the signature and the payload. We want to make these unique to the UID, but easy to find based on the
	// name/namespace as well.
	// $bucket/taskrun-$namespace-$name/$key.signature
	// $bucket/taskrun-$namespace-$name/$key.payload
	sigName := b.sigName(opts)
	b.logger.Infof("Storing signature at %s", sigName)

	sigObj := b.writer.GetWriter(sigName)
	if _, err := sigObj.Write([]byte(signature)); err != nil {
		return err
	}
	if err := sigObj.Close(); err != nil {
		return err
	}

	payloadObj := b.writer.GetWriter(b.payloadName(opts))
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
	certName := b.certName(opts)
	certObj := b.writer.GetWriter(certName)
	defer certObj.Close()
	if _, err := certObj.Write([]byte(opts.Cert)); err != nil {
		return err
	}
	if err := certObj.Close(); err != nil {
		return err
	}

	chainName := b.chainName(opts)
	chainObj := b.writer.GetWriter(chainName)
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
	GetWriter(object string) io.WriteCloser
}

type writer struct {
	client *storage.Client
	bucket string
}

type gcsReader interface {
	GetReader(object string) (io.ReadCloser, error)
}

type reader struct {
	client *storage.Client
	bucket string
}

func (r *writer) GetWriter(object string) io.WriteCloser {
	ctx := context.Background()
	return r.client.Bucket(r.bucket).Object(object).NewWriter(ctx)
}

func (r *reader) GetReader(object string) (io.ReadCloser, error) {
	ctx := context.Background()
	return r.client.Bucket(r.bucket).Object(object).NewReader(ctx)
}

func (b *Backend) RetrieveSignatures(opts config.StorageOpts) (map[string][]string, error) {
	object := b.sigName(opts)
	signature, err := b.retrieveObject(object)
	if err != nil {
		return nil, err
	}

	m := make(map[string][]string)
	m[object] = []string{signature}
	return m, nil
}

func (b *Backend) RetrievePayloads(opts config.StorageOpts) (map[string]string, error) {
	object := b.payloadName(opts)
	m := make(map[string]string)
	payload, err := b.retrieveObject(object)
	if err != nil {
		return nil, err
	}

	m[object] = payload
	return m, nil
}

func (b *Backend) retrieveObject(object string) (string, error) {
	reader, err := b.reader.GetReader(object)
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

func (b *Backend) sigName(opts config.StorageOpts) string {
	return fmt.Sprintf(SignatureNameFormat, b.tr.Namespace, b.tr.Name, opts.Key)
}

func (b *Backend) payloadName(opts config.StorageOpts) string {
	return fmt.Sprintf(PayloadNameFormat, b.tr.Namespace, b.tr.Name, opts.Key)
}

func (b *Backend) certName(opts config.StorageOpts) string {
	return fmt.Sprintf(CertNameFormat, b.tr.Namespace, b.tr.Name, opts.Key)
}

func (b *Backend) chainName(opts config.StorageOpts) string {
	return fmt.Sprintf(ChainNameFormat, b.tr.Namespace, b.tr.Name, opts.Key)
}
