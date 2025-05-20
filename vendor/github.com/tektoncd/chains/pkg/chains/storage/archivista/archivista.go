/*
Copyright 2025 The Tekton Authors
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

package archivista

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	archivistaClient "github.com/in-toto/archivista/pkg/http-client"
	"github.com/in-toto/go-witness/dsse"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"knative.dev/pkg/logging"
)

const (
	// StorageBackendArchivista is the name of the Archivista storage backend
	StorageBackendArchivista = "archivista"
)

// Backend is a storage backend that is capable of storing Payloaders that are signed and wrapped
// with a DSSE envelope. Archivista is an in-toto attestation storage service.
type Backend struct {
	client *archivistaClient.ArchivistaClient
	url    string
	cfg    config.ArchivistaStorageConfig
}

// NewStorageBackend returns a new Archivista StorageBackend that can store Payloaders that are signed
// and wrapped in a DSSE envelope
func NewStorageBackend(cfg config.Config) (*Backend, error) {
	archCfg := cfg.Storage.Archivista
	if strings.TrimSpace(archCfg.URL) == "" {
		return nil, fmt.Errorf("missing archivista URL in storage configuration")
	}

	client, err := archivistaClient.CreateArchivistaClient(&http.Client{}, archCfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Archivista client: %w", err)
	}

	return &Backend{
		client: client,
		url:    archCfg.URL,
		cfg:    archCfg,
	}, nil
}

// StorePayload attempts to parse `signature` as a DSSE envelope, and if successful
// sends it to an Archivista server for storage.
func (b *Backend) StorePayload(ctx context.Context, _ objects.TektonObject, _ []byte, signature string, _ config.StorageOpts) error {
	logger := logging.FromContext(ctx)
	var env dsse.Envelope
	if err := json.Unmarshal([]byte(signature), &env); err != nil {
		logger.Errorf("Failed to parse DSSE envelope: %w", err)
		return errors.Join(errors.New("Failed to parse DSSE envelope"), err)
	}

	uploadResp, err := b.client.Store(ctx, env)
	if err != nil {
		logger.Errorw("Failed to upload DSSE envelope to Archivista", "error", err)
		return err
	}
	logger.Infof("Successfully uploaded DSSE envelope to Archivista, response: %+v", uploadResp)
	return nil
}

// RetrievePayload is not implemented for Archivista.
func (b *Backend) RetrievePayload(_ context.Context, _ string) ([]byte, []byte, error) {
	return nil, nil, fmt.Errorf("RetrievePayload not implemented for Archivista")
}

// RetrievePayloads is not implemented for Archivista.
func (b *Backend) RetrievePayloads(_ context.Context, _ objects.TektonObject, _ config.StorageOpts) (map[string]string, error) {
	return nil, fmt.Errorf("RetrievePayloads not implemented for Archivista")
}

// RetrieveSignatures is not implemented for Archivista.
func (b *Backend) RetrieveSignatures(_ context.Context, _ objects.TektonObject, _ config.StorageOpts) (map[string][]string, error) {
	return nil, fmt.Errorf("RetrieveSignatures not implemented for Archivista")
}

// Type returns the name of the storage backend
func (b *Backend) Type() string {
	return StorageBackendArchivista
}
