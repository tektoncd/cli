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

package chains

import (
	"context"
	"strings"

	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	versioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/logging"
)

type Verifier interface {
	VerifyTaskRun(ctx context.Context, tr *v1beta1.TaskRun) error
}

type TaskRunVerifier struct {
	KubeClient        kubernetes.Interface
	Pipelineclientset versioned.Interface
	SecretPath        string
}

func (tv *TaskRunVerifier) VerifyTaskRun(ctx context.Context, tr *v1beta1.TaskRun) error {
	// Get all the things we might need (storage backends, signers and formatters)
	cfg := *config.FromContext(ctx)
	logger := logging.FromContext(ctx)
	logger.Infof("Verifying signature for TaskRun %s/%s", tr.Namespace, tr.Name)

	// TODO: Hook this up to config.
	enabledSignableTypes := []artifacts.Signable{
		&artifacts.TaskRunArtifact{Logger: logger},
		&artifacts.OCIArtifact{Logger: logger},
	}

	// Storage
	allBackends, err := getBackends(tv.Pipelineclientset, tv.KubeClient, logger, tr, cfg)
	if err != nil {
		return err
	}
	signers := allSigners(tv.SecretPath, cfg, logger)

	for _, signableType := range enabledSignableTypes {
		if !signableType.Enabled(cfg) {
			continue
		}
		// Verify the signature.
		signerType := signableType.Signer(cfg)
		signer, ok := signers[signerType]
		if !ok {
			logger.Warnf("No signer %s configured for %s", signerType, signableType.Type())
			continue
		}

		for _, backend := range signableType.StorageBackend(cfg).List() {
			b := allBackends[backend]
			signatures, err := b.RetrieveSignatures(config.StorageOpts{})
			if err != nil {
				return err
			}
			payload, err := b.RetrievePayloads(config.StorageOpts{})
			if err != nil {
				return err
			}
			for image, sigs := range signatures {
				for _, sig := range sigs {
					if err := signer.VerifySignature(strings.NewReader(sig), strings.NewReader(payload[image])); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
