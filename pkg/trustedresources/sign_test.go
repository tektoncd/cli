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

package trustedresources

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/theupdateframework/go-tuf/encrypted"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	password  = "hello"
	namespace = "test"
)

func init() {
	os.Setenv("PRIVATE_PASSWORD", password)
}

var (
	// tasks for testing
	taskSpec = &v1beta1.TaskSpec{
		Steps: []v1beta1.Step{{
			Image: "ubuntu",
			Name:  "echo",
		}},
	}

	pipelineSpec = &v1beta1.PipelineSpec{
		Tasks: []v1beta1.PipelineTask{
			{
				Name: "pipelinetask",
			},
		},
	}
)

func TestSign(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyFile := "cosign.key"
	publicKeyFile := "cosign.pub"

	signer, err := GenerateKeyFile(tmpDir, privateKeyFile, publicKeyFile)
	if err != nil {
		t.Fatal(err)
	}

	tcs := []struct {
		name       string
		resource   metav1.Object
		kind       string
		targetFile string
	}{{
		name:       "Task Sign and pass verification",
		resource:   getTask(),
		kind:       "Task",
		targetFile: "signed-task.yaml",
	}, {
		name:       "Pipeline Sign and pass verification",
		resource:   getPipeline(),
		kind:       "Pipeline",
		targetFile: "signed-pipeline.yaml",
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if err := Sign(tc.resource, filepath.Join(tmpDir, privateKeyFile), "", filepath.Join(tmpDir, tc.targetFile)); err != nil {
				t.Fatalf("Sign() get err %v", err)
			}
			signed, err := os.ReadFile(filepath.Join(tmpDir, tc.targetFile))
			if err != nil {
				t.Fatalf("error reading file: %v", err)
			}

			target, signature, err := UnmarshalCRD(signed, tc.kind)
			if err != nil {
				t.Fatalf("error unmarshalling crd: %v", err)
			}
			if err := VerifyInterface(target, signer, signature); err != nil {
				t.Fatalf("VerifyTaskOCIBundle get error: %v", err)
			}
		})
	}
}

func getTask() *v1beta1.Task {
	return &v1beta1.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Task",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-task",
			Namespace: namespace,
		},
		Spec: *taskSpec,
	}
}

func getPipeline() *v1beta1.Pipeline {
	return &v1beta1.Pipeline{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Pipeline",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-Pipeline",
			Namespace: namespace,
		},
		Spec: *pipelineSpec,
	}
}

// GenerateKeyFile creates public key files, return the SignerVerifier
func GenerateKeyFile(dir string, privateKeyFile, pubkeyfile string) (signature.SignerVerifier, error) {
	keys, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	p := []byte(password)

	x509Encoded, err := x509.MarshalPKCS8PrivateKey(keys)
	if err != nil {
		return nil, errors.Wrap(err, "x509 encoding private key")
	}
	encBytes, err := encrypted.Encrypt(x509Encoded, p)
	if err != nil {
		return nil, err
	}

	privBytes := pem.EncodeToMemory(&pem.Block{
		Bytes: encBytes,
		Type:  cosign.CosignPrivateKeyPemType,
	})

	if err := os.WriteFile(filepath.Join(dir, privateKeyFile), privBytes, 0600); err != nil {
		return nil, err
	}

	// Now do the public key
	pubBytes, err := cryptoutils.MarshalPublicKeyToPEM(keys.Public())
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(filepath.Join(dir, pubkeyfile), pubBytes, 0600); err != nil {
		return nil, err
	}

	sv, err := signature.LoadSignerVerifier(keys, crypto.SHA256)
	if err != nil {
		return nil, err
	}

	return sv, nil
}
