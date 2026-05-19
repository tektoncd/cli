package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetURL_TektonHub(t *testing.T) {
	tHub := &tektonHubClient{}
	err := tHub.SetURL("http://localhost:80000")
	assert.NoError(t, err)

	err = tHub.SetURL("localhost:8000")
	assert.NoError(t, err)

	err = tHub.SetURL("http://80.80.79.9:80")
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

	err = aHub.SetURL("localhost:8000")
	assert.NoError(t, err)

	err = aHub.SetURL("http://80.80.79.9:80")
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
}
