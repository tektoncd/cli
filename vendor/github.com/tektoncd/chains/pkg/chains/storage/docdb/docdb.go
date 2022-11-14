/*
Copyright 2021 The Tekton Authors
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

package docdb

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"go.uber.org/zap"
	"gocloud.dev/docstore"
	_ "gocloud.dev/docstore/awsdynamodb"
	_ "gocloud.dev/docstore/gcpfirestore"
	_ "gocloud.dev/docstore/mongodocstore"
)

const (
	StorageTypeDocDB = "docdb"
)

// Backend is a storage backend that stores signed payloads in the TaskRun metadata as an annotation.
// It is stored as base64 encoded JSON.
type Backend struct {
	logger *zap.SugaredLogger
	coll   *docstore.Collection
}

type SignedDocument struct {
	Signed    []byte
	Signature string
	Cert      string
	Chain     string
	Object    interface{}
	Name      string
}

// NewStorageBackend returns a new Tekton StorageBackend that stores signatures on a TaskRun
func NewStorageBackend(ctx context.Context, logger *zap.SugaredLogger, cfg config.Config) (*Backend, error) {
	url := cfg.Storage.DocDB.URL
	coll, err := docstore.OpenCollection(ctx, url)
	if err != nil {
		return nil, err
	}

	return &Backend{
		logger: logger,
		coll:   coll,
	}, nil
}

// StorePayload implements the Payloader interface.
func (b *Backend) StorePayload(ctx context.Context, _ objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error {
	var obj interface{}
	if err := json.Unmarshal(rawPayload, &obj); err != nil {
		return err
	}

	entry := SignedDocument{
		Signed:    rawPayload,
		Signature: base64.StdEncoding.EncodeToString([]byte(signature)),
		Object:    obj,
		Name:      opts.ShortKey,
		Cert:      opts.Cert,
		Chain:     opts.Chain,
	}

	if err := b.coll.Put(ctx, &entry); err != nil {
		return err
	}

	return nil
}

func (b *Backend) Type() string {
	return StorageTypeDocDB
}

func (b *Backend) RetrieveSignatures(ctx context.Context, _ objects.TektonObject, opts config.StorageOpts) (map[string][]string, error) {
	// Retrieve the document.
	documents, err := b.retrieveDocuments(ctx, opts)
	if err != nil {
		return nil, err
	}

	m := make(map[string][]string)
	for _, d := range documents {
		// Extract and decode the signature.
		sig, err := base64.StdEncoding.DecodeString(d.Signature)
		if err != nil {
			return nil, err
		}
		m[d.Name] = []string{string(sig)}
	}
	return m, nil
}

func (b *Backend) RetrievePayloads(ctx context.Context, _ objects.TektonObject, opts config.StorageOpts) (map[string]string, error) {
	documents, err := b.retrieveDocuments(ctx, opts)
	if err != nil {
		return nil, err
	}

	m := make(map[string]string)
	for _, d := range documents {
		m[d.Name] = string(d.Signed)
	}

	return m, nil
}

func (b *Backend) retrieveDocuments(ctx context.Context, opts config.StorageOpts) ([]SignedDocument, error) {
	d := SignedDocument{Name: opts.ShortKey}
	if err := b.coll.Get(ctx, &d); err != nil {
		return []SignedDocument{}, err
	}
	return []SignedDocument{d}, nil
}
