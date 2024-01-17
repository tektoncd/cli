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

package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	cosignsignature "github.com/sigstore/cosign/v2/pkg/signature"
	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/cli/pkg/trustedresources"
)

func TestSign(t *testing.T) {
	ctx := context.Background()
	p := &test.Params{}
	pipeline := Command(p)
	os.Setenv("PRIVATE_PASSWORD", "1234")

	testcases := []struct {
		name       string
		taskFile   string
		apiVersion string
	}{{
		name:       "sign and verify v1beta1 Pipeline",
		taskFile:   "testdata/pipeline.yaml",
		apiVersion: "v1beta1",
	}, {
		name:       "sign and verify v1 Pipeline",
		taskFile:   "testdata/pipeline-v1.yaml",
		apiVersion: "v1",
	}}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			targetFile := filepath.Join(tmpDir, "signed.yaml")
			out, err := test.ExecuteCommand(pipeline, "sign", tc.taskFile, "-K", "testdata/cosign.key", "-f", targetFile, "--api-version", tc.apiVersion)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			expected := fmt.Sprintf("*Warning*: This is an experimental command, it's usage and behavior can change in the next release(s)\nPipeline %s is signed successfully \n", tc.taskFile)
			test.AssertOutput(t, expected, out)

			// verify the signed task
			verifier, err := cosignsignature.LoadPublicKey(ctx, "testdata/cosign.pub")
			if err != nil {
				t.Errorf("error getting verifier from key file: %v", err)
			}

			signed, err := os.ReadFile(targetFile)
			if err != nil {
				t.Fatalf("error reading file: %v", err)
			}

			target, signature, err := trustedresources.UnmarshalCRD(signed, "Pipeline", tc.apiVersion)
			if err != nil {
				t.Fatalf("error unmarshalling crd: %v", err)
			}

			if err := trustedresources.VerifyInterface(target, verifier, signature); err != nil {
				t.Fatalf("VerifyInterface get error: %v", err)
			}
		})
	}
}
