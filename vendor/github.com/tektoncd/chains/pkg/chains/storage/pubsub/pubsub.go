/*
Copyright 2022 The Tekton Authors
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

package pubsub

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"gocloud.dev/pubsub/kafkapubsub"
	"knative.dev/pkg/logging"

	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/mempubsub"
)

const (
	StorageBackendPubSub   = "pubsub"
	PubSubProviderInMemory = "inmemory"
	PubSubProviderKafka    = "kafka"
)

// Backend is a storage backend that stores signed payloads in the TaskRun metadata as an annotation.
// It is stored as base64 encoded JSON.
type Backend struct {
	cfg config.Config
}

// NewStorageBackend returns a new Tekton StorageBackend that stores signatures on a TaskRun
func NewStorageBackend(ctx context.Context, cfg config.Config) (*Backend, error) {
	return &Backend{
		cfg: cfg,
	}, nil
}

func (b *Backend) Type() string {
	return StorageBackendPubSub
}

func (b *Backend) StorePayload(ctx context.Context, obj objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error {
	logger := logging.FromContext(ctx)
	logger.Infof("Storing payload on Object %s/%s", obj.GetNamespace(), obj.GetName())

	// Construct a *pubsub.Topic.
	topic, err := b.NewTopic(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := topic.Shutdown(ctx); err != nil {
			logger.Error(err)
		}
	}()

	// Send the message with the DSSE signature.
	err = topic.Send(ctx, &pubsub.Message{
		Body: []byte(signature),
		Metadata: map[string]string{
			"payload":   base64.StdEncoding.EncodeToString(rawPayload),
			"signature": signature,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (b *Backend) RetrievePayloads(ctx context.Context, _ objects.TektonObject, opts config.StorageOpts) (map[string]string, error) {
	return nil, fmt.Errorf("not implemented for this storage backend: %s", b.Type())
}

func (b *Backend) RetrieveSignatures(ctx context.Context, _ objects.TektonObject, opts config.StorageOpts) (map[string][]string, error) {
	return nil, fmt.Errorf("not implemented for this storage backend: %s", b.Type())
}

func (b *Backend) NewTopic(ctx context.Context) (*pubsub.Topic, error) {
	logger := logging.FromContext(ctx)
	provider := b.cfg.Storage.PubSub.Provider
	topic := b.cfg.Storage.PubSub.Topic
	logger.Infof("Creating new %q pubsub producer for %q topic", provider, topic)
	switch provider {
	case PubSubProviderKafka:
		// The set of brokers in the Kafka cluster.
		addrs := []string{b.cfg.Storage.PubSub.Kafka.BootstrapServers}
		logger.Infof("Configuring Kafka brokers: %s", addrs)
		// The Kafka client configuration to use.
		config := kafkapubsub.MinimalConfig()
		return kafkapubsub.OpenTopic(addrs, config, topic, nil)
	case PubSubProviderInMemory:
		addr := fmt.Sprintf("mem://%s", b.cfg.Storage.PubSub.Topic)
		logger.Infof("Configuring in-memory producer: %s", addr)
		return pubsub.OpenTopic(context.TODO(), addr)
	default:
		return nil, fmt.Errorf("invalid provider: %q", provider)
	}
}
