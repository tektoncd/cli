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
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/theupdateframework/go-tuf/encrypted"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	password  = "hello"
	namespace = "test"
	// PEM type used by cosign for encrypted private keys. Kept as a local string so tests can generate compatible key files without importing cosign.
	encryptedCosignPrivateKeyPEMType = "ENCRYPTED COSIGN PRIVATE KEY"
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

const (
	v1TaskYAML = `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: test-task
  namespace: test
spec:
  steps:
  - name: build
    image: golang
    script: go build ./...
    computeResources:
      requests:
        memory: 1Gi
`
	v1PipelineYAML = `apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: test-pipeline
  namespace: test
spec:
  tasks:
  - name: build
    taskSpec:
      steps:
      - name: build
        image: golang
        script: go build ./...
`
)

func TestDecode(t *testing.T) {
	tcs := []struct {
		name string
		yaml string
		kind string
		want metav1.Object
	}{{
		name: "v1 Task",
		yaml: v1TaskYAML,
		kind: "Task",
		want: &v1.Task{},
	}, {
		name: "v1beta1 Task",
		yaml: "apiVersion: tekton.dev/v1beta1\nkind: Task\nmetadata:\n  name: test-task\n",
		kind: "Task",
		want: &v1beta1.Task{},
	}, {
		name: "Task without apiVersion",
		yaml: "kind: Task\nmetadata:\n  name: test-task\n",
		kind: "Task",
		want: &v1beta1.Task{},
	}, {
		name: "v1 Pipeline",
		yaml: v1PipelineYAML,
		kind: "Pipeline",
		want: &v1.Pipeline{},
	}, {
		name: "v1beta1 Pipeline",
		yaml: "apiVersion: tekton.dev/v1beta1\nkind: Pipeline\nmetadata:\n  name: test-pipeline\n",
		kind: "Pipeline",
		want: &v1beta1.Pipeline{},
	}}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Decode([]byte(tc.yaml), tc.kind)
			if err != nil {
				t.Fatalf("Decode() get err %v", err)
			}
			if gotType, wantType := reflectTypeName(got), reflectTypeName(tc.want); gotType != wantType {
				t.Fatalf("Decode() returned %s, want %s", gotType, wantType)
			}
		})
	}

	if _, err := Decode([]byte(v1TaskYAML), "TaskRun"); err == nil {
		t.Fatal("Decode() of an unsupported kind should fail")
	}
}

// TestSignV1 checks that a signed tekton.dev/v1 resource is still a valid v1
// resource and that its signature matches the checksum Tekton Pipelines verifies.
func TestSignV1(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyFile := "cosign.key"
	publicKeyFile := "cosign.pub"

	signer, err := GenerateKeyFile(tmpDir, privateKeyFile, publicKeyFile)
	if err != nil {
		t.Fatal(err)
	}

	tcs := []struct {
		name    string
		yaml    string
		kind    string
		decoded hashable
		checkFn func(t *testing.T, o hashable)
	}{{
		name:    "Task",
		yaml:    v1TaskYAML,
		kind:    "Task",
		decoded: &v1.Task{},
		checkFn: func(t *testing.T, o hashable) {
			task := o.(*v1.Task)
			if got := task.Spec.Steps[0].ComputeResources.Requests.Memory().String(); got != "1Gi" {
				t.Errorf("signed Task lost computeResources: memory request is %q", got)
			}
		},
	}, {
		name:    "Pipeline",
		yaml:    v1PipelineYAML,
		kind:    "Pipeline",
		decoded: &v1.Pipeline{},
		checkFn: func(*testing.T, hashable) {},
	}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			resource, err := Decode([]byte(tc.yaml), tc.kind)
			if err != nil {
				t.Fatalf("Decode() get err %v", err)
			}
			targetFile := filepath.Join(tmpDir, "signed-"+tc.kind+".yaml")
			if err := Sign(resource, filepath.Join(tmpDir, privateKeyFile), "", targetFile); err != nil {
				t.Fatalf("Sign() get err %v", err)
			}
			signed, err := os.ReadFile(targetFile)
			if err != nil {
				t.Fatalf("error reading file: %v", err)
			}

			// Tekton Pipelines decodes strictly, so a v1beta1-only field such as
			// steps[].resources would be rejected.
			if err := yaml.UnmarshalStrict(signed, tc.decoded); err != nil {
				t.Fatalf("signed %s is not a valid v1 %s: %v", tc.kind, tc.kind, err)
			}
			tc.checkFn(t, tc.decoded)

			annotations := tc.decoded.GetAnnotations()
			sig, err := base64.StdEncoding.DecodeString(annotations[SignatureAnnotation])
			if err != nil {
				t.Fatalf("error decoding signature: %v", err)
			}
			delete(annotations, SignatureAnnotation)
			tc.decoded.SetAnnotations(annotations)
			checksum, err := tc.decoded.Checksum()
			if err != nil {
				t.Fatalf("Checksum() get err %v", err)
			}
			if err := signer.VerifySignature(bytes.NewReader(sig), bytes.NewReader(checksum)); err != nil {
				t.Fatalf("signature does not match the Tekton Pipelines checksum: %v", err)
			}
		})
	}
}

// hashable is a resource that Tekton Pipelines can compute a checksum for.
type hashable interface {
	metav1.Object
	Checksum() ([]byte, error)
}

func reflectTypeName(o metav1.Object) string {
	return fmt.Sprintf("%T", o)
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
		Type:  encryptedCosignPrivateKeyPEMType,
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
