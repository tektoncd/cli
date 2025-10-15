// Copyright 2021 The Witness Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cryptoutil

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"

	"go.step.sm/crypto/pemutil"
)

// PEMType is a specific type for string constants used during PEM encoding and decoding
type PEMType string

const (
	// PublicKeyPEMType is the string "PUBLIC KEY" to be used during PEM encoding and decoding
	PublicKeyPEMType PEMType = "PUBLIC KEY"
	// PKCS1PublicKeyPEMType is the string "RSA PUBLIC KEY" used to parse PKCS#1-encoded public keys
	PKCS1PublicKeyPEMType PEMType = "RSA PUBLIC KEY"
)

type ErrUnsupportedPEM struct {
	t string
}

func (e ErrUnsupportedPEM) Error() string {
	return fmt.Sprintf("unsupported pem type: %v", e.t)
}

type ErrInvalidPemBlock struct{}

func (e ErrInvalidPemBlock) Error() string {
	return "invalid pem block"
}

func DigestBytes(data []byte, hash crypto.Hash) ([]byte, error) {
	return Digest(bytes.NewReader(data), hash)
}

func Digest(r io.Reader, hash crypto.Hash) ([]byte, error) {
	hashFunc := hash.New()
	if _, err := io.Copy(hashFunc, r); err != nil {
		return nil, err
	}

	return hashFunc.Sum(nil), nil
}

func HexEncode(src []byte) []byte {
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)
	return dst
}

func GeneratePublicKeyID(pub interface{}, hash crypto.Hash) (string, error) {
	pemBytes, err := PublicPemBytes(pub)
	if err != nil {
		return "", err
	}

	digest, err := DigestBytes(pemBytes, hash)
	if err != nil {
		return "", err
	}

	return string(HexEncode(digest)), nil
}

func PublicPemBytes(pub interface{}) ([]byte, error) {
	keyBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: keyBytes})
	if err != nil {
		return nil, err
	}

	return pemBytes, err
}

// UnmarshalPEMToPublicKey converts a PEM-encoded byte slice into a crypto.PublicKey
func UnmarshalPEMToPublicKey(pemBytes []byte) (crypto.PublicKey, error) {
	derBytes, _ := pem.Decode(pemBytes)
	if derBytes == nil {
		return nil, errors.New("PEM decoding failed")
	}
	switch derBytes.Type {
	case string(PublicKeyPEMType):
		return x509.ParsePKIXPublicKey(derBytes.Bytes)
	case string(PKCS1PublicKeyPEMType):
		return x509.ParsePKCS1PublicKey(derBytes.Bytes)
	default:
		return nil, fmt.Errorf("unknown Public key PEM file type: %v. Are you passing the correct public key?",
			derBytes.Type)
	}
}

// TryParsePEMBlock attempts to parse a PEM block without a passphrase.
// For encrypted keys, prefer TryParsePEMBlockWithPassword.
func TryParsePEMBlock(block *pem.Block) (interface{}, error) {
	return TryParsePEMBlockWithPassword(block, nil)
}

// TryParsePEMBlockWithPassword attempts to parse a PEM block, optionally
// decrypting it using the provided passphrase if the block is encrypted.
func TryParsePEMBlockWithPassword(block *pem.Block, password []byte) (interface{}, error) {
	if block == nil {
		return nil, ErrInvalidPemBlock{}
	}

	// If no password, only attempt unencrypted formats and return.
	if len(password) == 0 {
		// Unencrypted formats.
		if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
			return key, nil
		}
		if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
			return key, nil
		}
		if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
			return key, nil
		}
		if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
			return key, nil
		}
		if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
			return key, nil
		}
		if key, err := x509.ParseCertificate(block.Bytes); err == nil {
			return key, nil
		}

		return nil, ErrUnsupportedPEM{block.Type}
	}

	// Password provided: handle legacy-encrypted PEM, otherwise fall back to unencrypted parse.
	if _, ok := block.Headers["DEK-Info"]; ok {
		decryptedDER, err := x509.DecryptPEMBlock(block, password) //nolint:staticcheck // legacy support
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt legacy encrypted PEM: %w", err)
		}

		// Parse based on likely key formats
		if k, err := x509.ParsePKCS1PrivateKey(decryptedDER); err == nil {
			return k, nil
		}
		if k, err := x509.ParseECPrivateKey(decryptedDER); err == nil {
			return k, nil
		}
		if k, err := x509.ParsePKCS8PrivateKey(decryptedDER); err == nil {
			return k, nil
		}

		// Decrypted but unsupported key format.
		return nil, fmt.Errorf("decrypted legacy encrypted PEM does not contain a supported private key format")
	}

	// Not legacy-encrypted despite a provided password; try unencrypted parses.
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParseCertificate(block.Bytes); err == nil {
		return key, nil
	}

	return nil, ErrUnsupportedPEM{block.Type}
}

// TryParseKeyFromReader parses a key/cert from reader without password
// support. For encrypted keys, use TryParseKeyFromReaderWithPassword.
func TryParseKeyFromReader(r io.Reader) (interface{}, error) {
	return TryParseKeyFromReaderWithPassword(r, nil)
}

// TryParseKeyFromReaderWithPassword parses a key/cert from reader, optionally
// decrypting if a passphrase is provided.
func TryParseKeyFromReaderWithPassword(r io.Reader, password []byte) (interface{}, error) {
	bytes, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if len(password) == 0 {
		// We may want to handle files with multiple PEM blocks in them, but for now...
		pemBlock, _ := pem.Decode(bytes)
		return TryParsePEMBlockWithPassword(pemBlock, password)
	}

	// Use smallstep's pemutil to parse and decrypt various PEM/DER key formats,
	// including PKCS#8 EncryptedPrivateKeyInfo.
	obj, err := pemutil.Parse(bytes, pemutil.WithPassword(password))
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func TryParseCertificate(data []byte) (*x509.Certificate, error) {
	possibleCert, err := TryParseKeyFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	cert, ok := possibleCert.(*x509.Certificate)
	if !ok {
		return nil, fmt.Errorf("data was a valid verifier but not a certificate")
	}

	return cert, nil
}

// ComputeDigest calculates the digest value for the specified message using the supplied hash function
func ComputeDigest(rawMessage io.Reader, hashFunc crypto.Hash, supportedHashFuncs []crypto.Hash) ([]byte, crypto.Hash, error) {
	var cryptoSignerOpts crypto.SignerOpts = hashFunc
	hashedWith := cryptoSignerOpts.HashFunc()
	if !isSupportedAlg(hashedWith, supportedHashFuncs) {
		return nil, crypto.Hash(0), fmt.Errorf("unsupported hash algorithm: %q not in %v", hashedWith.String(), supportedHashFuncs)
	}

	digest, err := Digest(rawMessage, hashedWith)
	return digest, hashedWith, err
}

func isSupportedAlg(alg crypto.Hash, supportedAlgs []crypto.Hash) bool {
	if supportedAlgs == nil {
		return true
	}
	for _, supportedAlg := range supportedAlgs {
		if alg == supportedAlg {
			return true
		}
	}
	return false
}
