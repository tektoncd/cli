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

package trustedresources

import (
	"bytes"
	"context"
	"crypto"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/kms"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"golang.org/x/term"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	// SignatureAnnotation is the key of signature in annotation map
	SignatureAnnotation = "tekton.dev/signature"
)

var (
	// Read is for fuzzing
	read = readPasswordFn
)

// Sign the crd and output signed bytes to writer
func Sign(o metav1.Object, keyfile, kmsKey, targetFile string) error {
	// Load signer
	var signer signature.Signer
	var err error
	if keyfile != "" {
		signer, err = signature.LoadSignerFromPEMFile(keyfile, crypto.SHA256, getPass)
		if err != nil {
			return fmt.Errorf("error getting signer from key file: %v", err)
		}
	}
	if kmsKey != "" {
		ctx := context.Background()
		signer, err = kms.Get(ctx, kmsKey, crypto.SHA256)
		if err != nil {
			return fmt.Errorf("error getting kms signer: %v", err)
		}
	}

	// Get annotation
	a := o.GetAnnotations()
	if a == nil {
		a = map[string]string{}
	}

	// Sign object
	sig, err := signInterface(signer, o)
	if err != nil {
		return err
	}
	a[SignatureAnnotation] = base64.StdEncoding.EncodeToString(sig)
	o.SetAnnotations(a)
	signedBuf, err := yaml.Marshal(o)
	if err != nil {
		return err
	}

	// save signed file
	f, err := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("error opening output file: %v", err)
	}
	defer f.Close()
	_, err = f.Write(signedBuf)

	return err
}

// signInterface returns the encoded signature for the given object.
func signInterface(signer signature.Signer, i interface{}) ([]byte, error) {
	if signer == nil {
		return nil, fmt.Errorf("signer is nil")
	}
	b, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	h.Write(b)

	sig, err := signer.SignMessage(bytes.NewReader(h.Sum(nil)))
	if err != nil {
		return nil, err
	}

	return sig, nil
}

// UnmarshalCRD will get the task/pipeline from buffer and extract the signature.
func UnmarshalCRD(buf []byte, kind string) (metav1.Object, []byte, error) {
	var resource metav1.Object
	var signature []byte

	switch kind {
	case "Task":
		resource = &v1beta1.Task{}
		if err := yaml.Unmarshal(buf, &resource); err != nil {
			return nil, nil, err
		}
	case "Pipeline":
		resource = &v1beta1.Pipeline{}
		if err := yaml.Unmarshal(buf, &resource); err != nil {
			return nil, nil, err
		}
	}
	annotations := resource.GetAnnotations()
	signature, err := base64.StdEncoding.DecodeString(annotations[SignatureAnnotation])
	if err != nil {
		return nil, nil, err
	}
	delete(annotations, SignatureAnnotation)

	return resource, signature, nil
}

func getPass(confirm bool) ([]byte, error) {
	read := read(confirm)
	return read()
}

func readPasswordFn(confirm bool) func() ([]byte, error) {
	pw, ok := os.LookupEnv("PRIVATE_PASSWORD")
	switch {
	case ok:
		return func() ([]byte, error) {
			return []byte(pw), nil
		}
	case isTerminal():
		return func() ([]byte, error) {
			return getPassFromTerm(confirm)
		}
	// Handle piped in passwords.
	default:
		return func() ([]byte, error) {
			return io.ReadAll(os.Stdin)
		}
	}
}

func getPassFromTerm(confirm bool) ([]byte, error) {
	fmt.Fprint(os.Stderr, "Enter password for private key: ")
	// Unnecessary convert of syscall.Stdin on *nix, but Windows is a uintptr
	// nolint:unconvert
	pw1, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, err
	}
	fmt.Fprintln(os.Stderr)
	if !confirm {
		return pw1, nil
	}
	fmt.Fprint(os.Stderr, "Enter password for private key again: ")
	// Unnecessary convert of syscall.Stdin on *nix, but Windows is a uintptr
	// nolint:unconvert
	confirmpw, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, err
	}

	if string(pw1) != string(confirmpw) {
		return nil, errors.New("passwords do not match")
	}
	return pw1, nil
}

func isTerminal() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}
