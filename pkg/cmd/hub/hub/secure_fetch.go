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
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// maxManifestSize limits the size of downloaded manifests to prevent memory exhaustion
	maxManifestSize = 10 * 1024 * 1024 // 10 MB
)

var (
	// secureHTTPClient is a singleton HTTP client configured with secure defaults
	secureHTTPClient *http.Client
)

func init() {
	secureHTTPClient = createSecureHTTPClient()
}

// createSecureHTTPClient creates an HTTP client with secure TLS configuration
func createSecureHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		// Use strong cipher suites only
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		},
	}
	transport.TLSHandshakeTimeout = 10 * time.Second

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		// Prevent following redirects to arbitrary locations
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			// Validate redirect URL is also HTTPS and resolves to public IP
			if err := validateManifestURL(req.URL.String()); err != nil {
				return fmt.Errorf("redirect to insecure URL blocked: %w", err)
			}
			return nil
		},
	}
}

// validateManifestURL validates that a manifest URL meets security requirements
// It resolves the hostname and checks that all resolved IPs are public to prevent SSRF
func validateManifestURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("manifest URL cannot be empty")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid manifest URL: %w", err)
	}

	// Enforce HTTPS-only for manifest downloads
	if parsedURL.Scheme != "https" {
		return fmt.Errorf("insecure manifest URL scheme '%s' not allowed, only HTTPS is permitted", parsedURL.Scheme)
	}

	// Validate hostname is present
	if parsedURL.Host == "" {
		return fmt.Errorf("manifest URL missing hostname")
	}

	hostname := parsedURL.Hostname()

	// Resolve hostname to IP addresses to prevent DNS rebinding and hostname-based bypasses
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve manifest hostname %s: %w", hostname, err)
	}

	if len(ips) == 0 {
		return fmt.Errorf("manifest hostname %s resolves to no addresses", hostname)
	}

	// Check that all resolved IPs are public (not private, loopback, or link-local)
	for _, ip := range ips {
		if !isPublicIP(ip) {
			return fmt.Errorf("manifest URL hostname %s resolves to non-public address %s", hostname, ip.String())
		}
	}

	return nil
}

// isPublicIP checks if an IP address is publicly routable
// Returns false for loopback, private, link-local, and multicast addresses
func isPublicIP(ip net.IP) bool {
	// Reject loopback addresses (127.0.0.0/8 for IPv4, ::1 for IPv6)
	if ip.IsLoopback() {
		return false
	}

	// Reject private addresses (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, fc00::/7)
	if ip.IsPrivate() {
		return false
	}

	// Reject link-local addresses (169.254.0.0/16 for IPv4, fe80::/10 for IPv6)
	if ip.IsLinkLocalUnicast() {
		return false
	}

	// Reject link-local multicast (224.0.0.0/24 for IPv4, ff02::/16 for IPv6)
	if ip.IsLinkLocalMulticast() {
		return false
	}

	// Reject multicast addresses
	if ip.IsMulticast() {
		return false
	}

	// Reject unspecified addresses (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return false
	}

	return true
}

// verifyDigest verifies the SHA256 digest of data against expected digest
func verifyDigest(data []byte, expectedDigest string) error {
	if expectedDigest == "" {
		// No digest provided - skip verification
		return nil
	}

	// Compute SHA256 hash of the data
	hash := sha256.Sum256(data)
	actualDigest := hex.EncodeToString(hash[:])

	// Compare digests (case-insensitive)
	if !strings.EqualFold(actualDigest, expectedDigest) {
		return fmt.Errorf("manifest digest mismatch: expected %s, got %s", expectedDigest, actualDigest)
	}

	return nil
}

// secureHTTPGet fetches data from a URL with security validations
// This replaces the insecure httpGet function for manifest downloads
func secureHTTPGet(rawURL string) ([]byte, int, error) {
	return secureHTTPGetWithDigest(rawURL, "")
}

// secureHTTPGetWithDigest fetches data from a URL with security validations and optional digest verification
// expectedDigest should be a hex-encoded SHA256 hash, or empty string to skip digest verification
func secureHTTPGetWithDigest(rawURL string, expectedDigest string) ([]byte, int, error) {
	// Validate URL before making request
	if err := validateManifestURL(rawURL); err != nil {
		return nil, 0, fmt.Errorf("manifest URL validation failed: %w", err)
	}

	if err := loadConfigFile(); err != nil {
		return nil, 0, err
	}

	resp, err := secureHTTPClient.Get(rawURL)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP error status codes
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d from manifest URL", resp.StatusCode)
	}

	// Enforce maximum size to prevent memory exhaustion
	limitedReader := io.LimitReader(resp.Body, maxManifestSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read manifest: %w", err)
	}

	if len(data) > maxManifestSize {
		return nil, 0, fmt.Errorf("manifest exceeds maximum size of %d bytes", maxManifestSize)
	}

	// Verify digest if provided
	if err := verifyDigest(data, expectedDigest); err != nil {
		return nil, 0, err
	}

	return data, resp.StatusCode, nil
}
