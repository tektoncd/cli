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
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/storage"
	"github.com/tektoncd/chains/pkg/config"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	versioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/logging"
)

type Verifier interface {
	VerifyTaskRun(ctx context.Context, tr *v1.TaskRun) error
}

type TaskRunVerifier struct {
	KubeClient        kubernetes.Interface
	Pipelineclientset versioned.Interface
	SecretPath        string
}

func (tv *TaskRunVerifier) VerifyTaskRun(ctx context.Context, tr *v1.TaskRun) error {
	// Get all the things we might need (storage backends, signers and formatters)
	cfg := *config.FromContext(ctx)
	logger := logging.FromContext(ctx)
	logger.Infof("Verifying signature for TaskRun %s/%s", tr.Namespace, tr.Name)

	// TODO: Hook this up to config.
	enabledSignableTypes := []artifacts.Signable{
		&artifacts.TaskRunArtifact{},
		&artifacts.OCIArtifact{},
	}

	trObj := objects.NewTaskRunObjectV1(tr)

	// Storage
	allBackends, err := storage.InitializeBackends(ctx, tv.Pipelineclientset, tv.KubeClient, cfg)
	if err != nil {
		return err
	}
	signers := allSigners(ctx, tv.SecretPath, cfg)

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

		for _, backend := range sets.List[string](signableType.StorageBackend(cfg)) {
			b := allBackends[backend]
			signatures, err := b.RetrieveSignatures(ctx, trObj, config.StorageOpts{})
			if err != nil {
				return err
			}
			payload, err := b.RetrievePayloads(ctx, trObj, config.StorageOpts{})
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
