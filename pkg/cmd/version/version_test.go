// Copyright © 2019 The Tekton Authors.
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

package version

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestVersionGood(t *testing.T) {
	v := clientVersion
	defer func() { clientVersion = v }()

	scenarios := []struct {
		name          string
		clientVersion string
		serverVersion string
		expected      string
	}{
		{
			name:          "test-available-new-version",
			clientVersion: "v0.0.1",
			serverVersion: "v0.0.2",
			expected:      "A newer version (v0.0.2) of Tekton CLI is available, please check https://github.com/tektoncd/cli/releases/tag/v0.0.2\n",
		},
		{
			name:          "test-same-version",
			clientVersion: "v0.0.10",
			serverVersion: "v0.0.10",
			expected:      "You are running the latest version (v0.0.10) of Tekton CLI\n",
		},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			clientVersion = s.clientVersion
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				okResponse, _ := json.Marshal(GHVersion{
					TagName: s.serverVersion,
					HTMLURL: "https://github.com/tektoncd/cli/releases/tag/" + s.serverVersion,
				})
				_, _ = w.Write([]byte(okResponse))
			})
			httpClient, teardown := testingHTTPClient(h)
			defer teardown()

			cli := NewClient(time.Duration(0))
			cli.httpClient = httpClient
			output, err := checkRelease(cli)
			assert.NilError(t, err)
			golden.Assert(t, output, fmt.Sprintf("%s.golden", t.Name()))
		})
	}

	clientVersion = "v1.2.3"

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{})

	cs := pipelinetest.Clients{Kube: seedData.Kube}
	p := &test.Params{Kube: cs.Kube}
	version := Command(p)
	got, err := test.ExecuteCommand(version, "version", "")
	assert.NilError(t, err)
	golden.Assert(t, got, fmt.Sprintf("%s.golden", t.Name()))
}

func TestVersionBad(t *testing.T) {
	v := clientVersion
	defer func() { clientVersion = v }()

	scenarios := []struct {
		name          string
		clientVersion string
		serverVersion string
		expectederr   string
	}{
		{
			name:          "bad-server-version",
			clientVersion: "v0.0.1",
			serverVersion: "BAD",
			expectederr:   "failed to parse version: No Major.Minor.Patch elements found",
		},
		{
			name:          "bad-client-version",
			clientVersion: "BAD",
			serverVersion: "v0.0.1",
			expectederr:   "failed to parse version: No Major.Minor.Patch elements found",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			clientVersion = s.clientVersion
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				okResponse, _ := json.Marshal(GHVersion{
					TagName: s.serverVersion,
					HTMLURL: "https://github.com/tektoncd/cli/releases/tag/" + s.serverVersion,
				})
				_, _ = w.Write([]byte(okResponse))
			})
			httpClient, teardown := testingHTTPClient(h)
			defer teardown()

			cli := NewClient(time.Duration(0))
			cli.httpClient = httpClient
			output, err := checkRelease(cli)
			assert.Error(t, err, s.expectederr)
			assert.Assert(t, output == "")
		})
	}
}

func testingHTTPClient(handler http.Handler) (*http.Client, func()) {
	s := httptest.NewTLSServer(handler)

	cli := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
			TLSClientConfig: &tls.Config{
				// nolint: gosec
				InsecureSkipVerify: true,
			},
		},
	}

	return cli, s.Close
}
