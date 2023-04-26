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

package storage

import (
	"context"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/storage/docdb"
	"github.com/tektoncd/chains/pkg/chains/storage/gcs"
	"github.com/tektoncd/chains/pkg/chains/storage/grafeas"
	"github.com/tektoncd/chains/pkg/chains/storage/oci"
	"github.com/tektoncd/chains/pkg/chains/storage/pubsub"
	"github.com/tektoncd/chains/pkg/chains/storage/tekton"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
)

// Backend is an interface to store a chains Payload
type Backend interface {
	StorePayload(ctx context.Context, obj objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error
	// RetrievePayloads maps [ref]:[payload] for a TaskRun
	RetrievePayloads(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string]string, error)
	// RetrieveSignatures maps [ref]:[list of signatures] for a TaskRun
	RetrieveSignatures(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string][]string, error)
	// Type is the string representation of the backend
	Type() string
}

// InitializeBackends creates and initializes every configured storage backend.
func InitializeBackends(ctx context.Context, ps versioned.Interface, kc kubernetes.Interface, cfg config.Config) (map[string]Backend, error) {
	// Add an entry here for every configured backend
	configuredBackends := []string{}
	if cfg.Artifacts.TaskRuns.Enabled() {
		configuredBackends = append(configuredBackends, sets.List[string](cfg.Artifacts.TaskRuns.StorageBackend)...)
	}
	if cfg.Artifacts.OCI.Enabled() {
		configuredBackends = append(configuredBackends, sets.List[string](cfg.Artifacts.OCI.StorageBackend)...)
	}
	if cfg.Artifacts.PipelineRuns.Enabled() {
		configuredBackends = append(configuredBackends, sets.List[string](cfg.Artifacts.PipelineRuns.StorageBackend)...)
	}

	// Now only initialize and return the configured ones.
	backends := map[string]Backend{}
	for _, backendType := range configuredBackends {
		switch backendType {
		case gcs.StorageBackendGCS:
			gcsBackend, err := gcs.NewStorageBackend(ctx, cfg)
			if err != nil {
				return nil, err
			}
			backends[backendType] = gcsBackend
		case tekton.StorageBackendTekton:
			backends[backendType] = tekton.NewStorageBackend(ps)
		case oci.StorageBackendOCI:
			ociBackend := oci.NewStorageBackend(ctx, kc, cfg)
			backends[backendType] = ociBackend
		case docdb.StorageTypeDocDB:
			docdbBackend, err := docdb.NewStorageBackend(ctx, cfg)
			if err != nil {
				return nil, err
			}
			backends[backendType] = docdbBackend
		case grafeas.StorageBackendGrafeas:
			grafeasBackend, err := grafeas.NewStorageBackend(ctx, cfg)
			if err != nil {
				return nil, err
			}
			backends[backendType] = grafeasBackend
		case pubsub.StorageBackendPubSub:
			pubsubBackend, err := pubsub.NewStorageBackend(ctx, cfg)
			if err != nil {
				return nil, err
			}
			backends[backendType] = pubsubBackend
		}

	}
	return backends, nil
}
