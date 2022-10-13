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
	"encoding/base64"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVerify(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyFile := "cosign.key"
	publicKeyFile := "cosign.pub"

	sv, err := GenerateKeyFile(tmpDir, privateKeyFile, publicKeyFile)
	if err != nil {
		t.Fatal(err)
	}

	unsignedTask := getTask()

	signedTask, err := getSignedTask(unsignedTask, sv)
	if err != nil {
		t.Fatal("fail to sign task", err)
	}

	tamperedTask := signedTask.DeepCopy()
	tamperedTask.Annotations["random"] = "attack"

	unsignedPipeline := getPipeline()

	signedPipeline, err := getSignedPipeline(unsignedPipeline, sv)
	if err != nil {
		t.Fatal("fail to sign task", err)
	}

	tamperedPipeline := signedPipeline.DeepCopy()
	tamperedPipeline.Annotations["random"] = "attack"

	tcs := []struct {
		name        string
		resource    metav1.Object
		expectedErr error
	}{{
		name:        "Unsigned Task fails verification",
		resource:    unsignedTask,
		expectedErr: fmt.Errorf("invalid signature when validating ASN.1 encoded signature"),
	}, {
		name:        "Tampered Task fails verification",
		resource:    tamperedTask,
		expectedErr: fmt.Errorf("invalid signature when validating ASN.1 encoded signature"),
	}, {
		name:        "Signed Task passes verification",
		resource:    signedTask,
		expectedErr: nil,
	}, {
		name:        "Unsigned Pipeline fails verification",
		resource:    unsignedPipeline,
		expectedErr: fmt.Errorf("invalid signature when validating ASN.1 encoded signature"),
	}, {
		name:        "Tampered Pipeline fails verification",
		resource:    tamperedPipeline,
		expectedErr: fmt.Errorf("invalid signature when validating ASN.1 encoded signature"),
	}, {
		name:        "Signed Pipeline passes verification",
		resource:    signedPipeline,
		expectedErr: nil,
	},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := Verify(tc.resource, filepath.Join(tmpDir, publicKeyFile), "")
			if tc.expectedErr != nil && (err == nil || err.Error() != tc.expectedErr.Error()) {
				t.Fatalf("Expected error %v but found %v instead", tc.expectedErr, err)
			} else if tc.expectedErr == nil && err != nil {
				t.Fatalf("Received unexpected error ( %#v )", err)
			}
		})
	}
}

func getSignedTask(unsigned *v1beta1.Task, signer signature.Signer) (*v1beta1.Task, error) {
	signedTask := unsigned.DeepCopy()
	signedTask.Name = "signed"
	if signedTask.Annotations == nil {
		signedTask.Annotations = map[string]string{}
	}
	signature, err := signInterface(signer, signedTask)
	if err != nil {
		return nil, err
	}
	signedTask.Annotations[SignatureAnnotation] = base64.StdEncoding.EncodeToString(signature)
	return signedTask, nil
}

func getSignedPipeline(unsigned *v1beta1.Pipeline, signer signature.Signer) (*v1beta1.Pipeline, error) {
	signedPipeline := unsigned.DeepCopy()
	signedPipeline.Name = "signed"
	if signedPipeline.Annotations == nil {
		signedPipeline.Annotations = map[string]string{}
	}
	signature, err := signInterface(signer, signedPipeline)
	if err != nil {
		return nil, err
	}
	signedPipeline.Annotations[SignatureAnnotation] = base64.StdEncoding.EncodeToString(signature)
	return signedPipeline, nil
}
