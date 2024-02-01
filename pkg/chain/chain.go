// Copyright © 2022 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package chain

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap/zapcore"

	"github.com/tektoncd/chains/pkg/chains/storage"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/cli/pkg/cli"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMapToContext returns a context initialized with the Chains ConfigMap.
func ConfigMapToContext(cs *cli.Clients, namespace string) (context.Context, error) {
	cfg, err := getChainsConfig(cs, namespace)
	if err != nil {
		return nil, err
	}
	return config.ToContext(context.Background(), cfg), nil
}

func GetTaskRunBackends(cs *cli.Clients, namespace string, tr *v1.TaskRun) (map[string]storage.Backend, config.StorageOpts, error) {
	// Prepare the logger.
	encoderCfg := zapcore.EncoderConfig{
		MessageKey: "msg",
	}
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderCfg), os.Stderr, zapcore.DebugLevel)
	logger := zap.New(core).WithOptions()

	// flushes buffer, if any
	defer func() {
		// intentionally ignoring error here, see https://github.com/uber-go/zap/issues/328
		_ = logger.Sync()
	}()

	// Get the storage backend.
	backends, err := initializeBackends(cs, namespace)
	if err != nil {
		return nil, config.StorageOpts{}, fmt.Errorf("failed to retrieve the backend storage: %v", err)
	}

	// Initialize the storage options.
	opts := config.StorageOpts{
		ShortKey: fmt.Sprintf("taskrun-%s", tr.UID),
	}

	return backends, opts, nil
}

func initializeBackends(cs *cli.Clients, namespace string) (map[string]storage.Backend, error) {
	// Retrieve the Chains configuration.
	cfg, err := getChainsConfig(cs, namespace)
	if err != nil {
		return nil, err
	}

	// Initialize the backend.
	backends, err := storage.InitializeBackends(context.Background(), cs.Tekton, cs.Kube, *cfg)
	if err != nil {
		return nil, fmt.Errorf("error initializing backends: %s", err)
	}

	// Return the configured backend.
	return backends, nil
}

// getChainsConfig returns the chains config configmap
func getChainsConfig(cs *cli.Clients, namespace string) (*config.Config, error) {
	chainsConfig, err := cs.Kube.CoreV1().ConfigMaps(namespace).Get(context.Background(), "chains-config", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error retrieving tekton chains configmap: %s", err)
	}
	cfg, err := config.NewConfigFromConfigMap(chainsConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating tekton chains configuration: %s", err)
	}
	return cfg, nil
}
