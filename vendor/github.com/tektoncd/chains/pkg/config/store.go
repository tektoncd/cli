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

package config

import (
	"context"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/reconciler"
)

type cfgKey struct{}

// ConfigStore is the configuration from a ConfigMap
// +k8s:deepcopy-gen=false
type ConfigStore struct {
	*configmap.UntypedStore
}

var _ reconciler.ConfigStore = (*ConfigStore)(nil)

// FromContext fetch config from context.
func FromContext(ctx context.Context) *Config {
	return ctx.Value(cfgKey{}).(*Config)
}

// ToContext adds config to given context.
func ToContext(ctx context.Context, c *Config) context.Context {
	return context.WithValue(ctx, cfgKey{}, c)
}

// ToContext adds Store contents to given context.
func (s *ConfigStore) ToContext(ctx context.Context) context.Context {
	return ToContext(ctx, s.Load())
}

// Load fetches config from Store.
func (s *ConfigStore) Load() *Config {
	return s.UntypedLoad(ChainsConfig).(*Config).DeepCopy()
}

// NewConfigStore returns a reconciler.ConfigStore for the chains configuration data.
func NewConfigStore(logger configmap.Logger, onAfterStore ...func(name string, value interface{})) *ConfigStore {
	return &ConfigStore{
		UntypedStore: configmap.NewUntypedStore(
			"chains",
			logger,
			configmap.Constructors{
				ChainsConfig: NewConfigFromConfigMap,
			},
			onAfterStore...,
		),
	}
}
