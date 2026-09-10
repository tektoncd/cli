// Copyright © 2026 The Tekton Authors.
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

package hub

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	rclient "github.com/tektoncd/cli/pkg/cmd/hub/gen/http/resource/client"
)

func TestValidateManifestURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid HTTPS URL with public IP",
			url:     "https://8.8.8.8/manifest.yaml",
			wantErr: false,
		},
		{
			name:    "valid HTTPS URL with public IPv6",
			url:     "https://[2001:4860:4860::8888]/manifest.yaml",
			wantErr: false,
		},
		{
			name:    "HTTP URL rejected",
			url:     "http://example.com/manifest.yaml",
			wantErr: true,
			errMsg:  "insecure manifest URL scheme",
		},
		{
			name:    "FTP URL rejected",
			url:     "ftp://example.com/manifest.yaml",
			wantErr: true,
			errMsg:  "insecure manifest URL scheme",
		},
		{
			name:    "empty URL rejected",
			url:     "",
			wantErr: true,
			errMsg:  "manifest URL cannot be empty",
		},
		{
			name:    "malformed URL rejected",
			url:     "ht!tp://invalid",
			wantErr: true,
			errMsg:  "invalid manifest URL",
		},
		{
			name:    "localhost rejected",
			url:     "https://localhost/manifest.yaml",
			wantErr: true,
			errMsg:  "non-public address",
		},
		{
			name:    "127.0.0.1 rejected",
			url:     "https://127.0.0.1/manifest.yaml",
			wantErr: true,
			errMsg:  "non-public address",
		},
		{
			name:    "IPv6 localhost rejected",
			url:     "https://[::1]/manifest.yaml",
			wantErr: true,
			errMsg:  "non-public address",
		},
		{
			name:    "URL without hostname rejected",
			url:     "https:///manifest.yaml",
			wantErr: true,
			errMsg:  "missing hostname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateManifestURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateManifestURL() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateManifestURL() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("validateManifestURL() unexpected error = %v", err)
			}
		})
	}
}

func TestIsPublicIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		// Loopback addresses - should be rejected
		{"IPv4 loopback", "127.0.0.1", false},
		{"IPv6 loopback", "::1", false},

		// Private IPv4 ranges - should be rejected
		{"private 10.x", "10.0.0.1", false},
		{"private 192.168.x", "192.168.1.1", false},
		{"private 172.16.x", "172.16.0.1", false},
		{"private 172.31.x", "172.31.255.255", false},

		// Link-local addresses - should be rejected
		{"IPv4 link-local", "169.254.1.1", false},
		{"IPv6 link-local", "fe80::1", false},

		// IPv6 private ranges - should be rejected
		{"IPv6 ULA fc00", "fc00::1", false},
		{"IPv6 ULA fd00", "fd00::1", false},

		// Multicast - should be rejected
		{"IPv4 multicast", "224.0.0.1", false},
		{"IPv6 multicast", "ff02::1", false},

		// Unspecified - should be rejected
		{"IPv4 unspecified", "0.0.0.0", false},
		{"IPv6 unspecified", "::", false},

		// Public addresses - should be accepted
		{"public IPv4", "8.8.8.8", true},
		{"public IPv4 alt", "1.1.1.1", true},
		{"public IPv6", "2001:4860:4860::8888", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP %s", tt.ip)
			}
			if got := isPublicIP(ip); got != tt.want {
				t.Errorf("isPublicIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestVerifyDigest(t *testing.T) {
	testData := []byte("test manifest content")
	hash := sha256.Sum256(testData)
	validDigest := hex.EncodeToString(hash[:])
	invalidDigest := "0000000000000000000000000000000000000000000000000000000000000000"

	tests := []struct {
		name           string
		data           []byte
		expectedDigest string
		wantErr        bool
	}{
		{
			name:           "valid digest",
			data:           testData,
			expectedDigest: validDigest,
			wantErr:        false,
		},
		{
			name:           "valid digest uppercase",
			data:           testData,
			expectedDigest: strings.ToUpper(validDigest),
			wantErr:        false,
		},
		{
			name:           "invalid digest",
			data:           testData,
			expectedDigest: invalidDigest,
			wantErr:        true,
		},
		{
			name:           "empty digest skips verification",
			data:           testData,
			expectedDigest: "",
			wantErr:        false,
		},
		{
			name:           "different data fails verification",
			data:           []byte("different content"),
			expectedDigest: validDigest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyDigest(tt.data, tt.expectedDigest)
			if tt.wantErr {
				if err == nil {
					t.Errorf("verifyDigest() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("verifyDigest() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestSecureHTTPGet(t *testing.T) {
	t.Run("rejects HTTP server", func(t *testing.T) {
		// Create HTTP server (not HTTPS)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("test"))
		}))
		defer server.Close()

		// Attempt to fetch - should fail due to HTTP
		_, _, err := secureHTTPGet(server.URL)
		if err == nil {
			t.Error("secureHTTPGet() should reject HTTP URLs")
		}
		if !strings.Contains(err.Error(), "insecure manifest URL scheme") {
			t.Errorf("secureHTTPGet() error = %v, want error about insecure scheme", err)
		}
	})

	t.Run("rejects localhost URLs", func(t *testing.T) {
		_, _, err := secureHTTPGet("https://localhost/manifest.yaml")
		if err == nil {
			t.Error("secureHTTPGet() should reject localhost URLs")
		}
		if !strings.Contains(err.Error(), "non-public address") {
			t.Errorf("secureHTTPGet() error = %v, want error about non-public address", err)
		}
	})
}

func TestSecureHTTPGetWithDigest(t *testing.T) {
	t.Run("validates URL before digest", func(t *testing.T) {
		// Should reject HTTP URLs even with valid digest
		_, _, err := secureHTTPGetWithDigest("http://example.com/manifest.yaml", "abc123")
		if err == nil {
			t.Error("secureHTTPGetWithDigest() should reject HTTP URLs")
		}
		if !strings.Contains(err.Error(), "insecure manifest URL scheme") {
			t.Errorf("secureHTTPGetWithDigest() error = %v, want error about insecure scheme", err)
		}
	})

}

func TestSecureHTTPClient(t *testing.T) {
	t.Run("has secure TLS configuration", func(t *testing.T) {
		transport, ok := secureHTTPClient.Transport.(*http.Transport)
		if !ok {
			t.Fatal("secureHTTPClient.Transport is not *http.Transport")
		}

		if transport.TLSClientConfig == nil {
			t.Fatal("TLSClientConfig is nil")
		}

		if transport.TLSClientConfig.MinVersion < tls.VersionTLS12 {
			t.Errorf("MinVersion = %d, want >= %d (TLS 1.2)", transport.TLSClientConfig.MinVersion, tls.VersionTLS12)
		}

		if len(transport.TLSClientConfig.CipherSuites) == 0 {
			t.Error("CipherSuites is empty, should have secure ciphers configured")
		}
	})

	t.Run("has timeout configured", func(t *testing.T) {
		if secureHTTPClient.Timeout == 0 {
			t.Error("secureHTTPClient.Timeout is 0, should have timeout configured")
		}
	})

	t.Run("has redirect validation", func(t *testing.T) {
		if secureHTTPClient.CheckRedirect == nil {
			t.Error("CheckRedirect is nil, should validate redirects")
		}
	})

	t.Run("validates redirect URLs", func(t *testing.T) {
		// Test that CheckRedirect validates HTTPS
		req := &http.Request{
			URL: mustParseURL("http://insecure.example.com/redirect"),
		}

		err := secureHTTPClient.CheckRedirect(req, nil)
		if err == nil {
			t.Error("CheckRedirect should reject HTTP redirects")
		}
	})
}

func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(fmt.Sprintf("mustParseURL: %v", err))
	}
	return u
}

// Integration tests for Manifest() security with TektonHubResourceResult

func TestManifestSecurityHTTPRejection(t *testing.T) {
	httpURL := "http://evil.example.com/malicious.yaml"

	resourceData := &ResourceData{
		LatestVersion: &rclient.ResourceVersionDataResponseBody{
			RawURL: &httpURL,
		},
	}

	result := &TektonHubResourceResult{
		data:         []byte(`{}`),
		status:       http.StatusOK,
		err:          nil,
		set:          true,
		resourceData: resourceData,
	}

	_, err := result.Manifest()
	if err == nil {
		t.Fatal("Manifest() should reject HTTP URLs")
	}

	if !strings.Contains(err.Error(), "insecure manifest URL scheme") {
		t.Errorf("Expected error about insecure scheme, got: %v", err)
	}
}

func TestManifestSecurityNonPublicRejection(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"localhost", "https://localhost/manifest.yaml"},
		{"127.0.0.1", "https://127.0.0.1/manifest.yaml"},
		{"::1", "https://[::1]/manifest.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceData := &ResourceData{
				LatestVersion: &rclient.ResourceVersionDataResponseBody{
					RawURL: &tt.url,
				},
			}

			result := &TektonHubResourceResult{
				data:         []byte(`{}`),
				status:       http.StatusOK,
				err:          nil,
				set:          true,
				resourceData: resourceData,
			}

			_, err := result.Manifest()
			if err == nil {
				t.Fatalf("Manifest() should reject non-public address %s", tt.url)
			}

			if !strings.Contains(err.Error(), "non-public address") {
				t.Errorf("Expected error about non-public address, got: %v", err)
			}
		})
	}
}
