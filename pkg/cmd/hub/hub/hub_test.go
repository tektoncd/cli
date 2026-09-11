package hub

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetURL_TektonHub(t *testing.T) {
	tHub := &tektonHubClient{}
	err := tHub.SetURL("http://localhost:80000")
	assert.NoError(t, err)

	err = tHub.SetURL("https://api.hub.tekton.dev")
	assert.NoError(t, err)

	err = tHub.SetURL("http://127.0.0.1:8080")
	assert.NoError(t, err)

	// default url
	err = tHub.SetURL("")
	assert.NoError(t, err)
	assert.Equal(t, tHub.apiURL, tektonHubURL)
}

func TestSetURL_ArtifactHub(t *testing.T) {
	aHub := &artifactHubClient{}
	err := aHub.SetURL("http://localhost:80000")
	assert.NoError(t, err)

	err = aHub.SetURL("https://artifacthub.io")
	assert.NoError(t, err)

	err = aHub.SetURL("http://127.0.0.1:8080")
	assert.NoError(t, err)

	// default url
	err = aHub.SetURL("")
	assert.NoError(t, err)
	assert.Equal(t, aHub.apiURL, artifactHubURL)
}

func TestSetURL_InvalidCase(t *testing.T) {

	hub := &tektonHubClient{}
	err := hub.SetURL("abc")
	assert.Error(t, err)
	assert.EqualError(t, err, "parse \"abc\": invalid URI for request")

	err = hub.SetURL("localhost:8000")
	assert.EqualError(t, err, "unsupported URL scheme \"localhost\"; only https is allowed")

	err = hub.SetURL("http://80.80.79.9:80")
	assert.EqualError(t, err, "refusing insecure HTTP URL \"http://80.80.79.9:80\"; use HTTPS")
}

func TestValidateHubURL(t *testing.T) {
	tests := []struct {
		raw     string
		wantErr string
	}{
		{raw: "https://artifacthub.io"},
		{raw: "https://api.hub.tekton.dev/v1/resource/tekton/task/git-clone/0.9/yaml"},
		{raw: "http://localhost:8080"},
		{raw: "http://127.0.0.1:8080"},
		{raw: "http://[::1]:8080"},
		{raw: "http://evil.example", wantErr: "refusing insecure HTTP URL"},
		{raw: "file:///etc/passwd", wantErr: "unsupported URL scheme"},
		{raw: "ftp://artifacthub.io", wantErr: "unsupported URL scheme"},
	}

	for _, tc := range tests {
		t.Run(tc.raw, func(t *testing.T) {
			u, err := url.ParseRequestURI(tc.raw)
			assert.NoError(t, err)
			err = validateHubURL(u)
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
