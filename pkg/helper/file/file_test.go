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

package file

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/test"
)

func TestLoadRemoteFile(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		okResponse := `
apiVersion: tekton.dev/v1alpha1
kind: Task
metadata:
 name: build-docker-image-from-git-source
spec:
 inputs:
   resources:
     - name: docker-source
       type: git
   params:
     - name: pathToDockerFile
       type: string
       description: The path to the dockerfile to build
       default: /workspace/docker-source/Dockerfile
     - name: pathToContext
       type: string
       description:
         The build context used by Kaniko
         (https://github.com/GoogleContainerTools/kaniko#kaniko-build-contexts)
       default: /workspace/docker-source
 outputs:
   resources:
     - name: builtImage
       type: image
 steps:
   - name: build-and-push
     image: gcr.io/kaniko-project/executor:v0.9.0
     # specifying DOCKER_CONFIG is required to allow kaniko to detect docker credential
     env:
       - name: "DOCKER_CONFIG"
         value: "/builder/home/.docker/"
     command:
       - /kaniko/executor
     args:
       - --dockerfile=$(inputs.params.pathToDockerFile)
       - --destination=$(outputs.resources.builtImage.url)
       - --context=$(inputs.params.pathToContext)`
		_, _ = w.Write([]byte(okResponse))
	})
	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	cs := &cli.Clients{
		HTTPClient: httpClient,
	}
	p := &test.Params{
		Cls: cs,
	}

	remoteContent, _ := LoadFileContent(p, "https://foo.com/task.yaml", IsYamlFile())

	localContent, _ := LoadFileContent(p, "./testdata/task.yaml", IsYamlFile())

	if d := cmp.Diff(remoteContent, localContent); d != "" {
		t.Errorf("Unexpected output mismatch: %s", d)
	}
}

func TestGetError(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "my own error message", http.StatusForbidden)

	})
	httpClient, teardown := testingHTTPClient(h)
	defer teardown()
	cs := &cli.Clients{
		HTTPClient: httpClient,
	}
	p := &test.Params{
		Cls: cs,
	}

	_, err := LoadFileContent(p, "httpz://foo.com/task.yaml", IsYamlFile())
	assert.NotNil(t, err)
	test.AssertOutput(t, "Get httpz://foo.com/task.yaml: unsupported protocol scheme \"httpz\"", err.Error())
}

func testingHTTPClient(handler http.Handler) (*http.Client, func()) {
	s := httptest.NewTLSServer(handler)

	cli := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	return cli, s.Close
}
