// Copyright © 2020 The Tekton Authors.
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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/user"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	ArtifactHubType = "artifact"
	TektonHubType   = "tekton"

	// hubURL - Hub API Server URL
	tektonHubURL   = "https://api.hub.tekton.dev"
	artifactHubURL = "https://artifacthub.io"
	hubConfigPath  = ".tekton/hub-config"

	tektonHubCatEndpoint         = "/v1/catalogs"
	artifactHubCatSearchEndpoint = "/api/v1/repositories/search"
	artifactHubCatInfoEndpoint   = "/api/v1/packages/tekton"
	artifactHubTaskType          = 7
	artifactHubPipelineType      = 11
)

type Client interface {
	GetType() string
	SetURL(u string) error
	Get(endpoint string) ([]byte, int, error)
	GetCatalogsList() ([]string, error)
	Search(opt SearchOption) SearchResult
	GetResource(opt ResourceOption) ResourceResult
	GetResourceYaml(opt ResourceOption) ResourceResult
	GetResourcesList(opt SearchOption) ([]string, error)
	GetResourceVersions(opt ResourceOption) ResourceVersionResult
	GetResourceVersionslist(opt ResourceOption) ([]string, error)
}

type tektonHubClient struct {
	apiURL string
}

type artifactHubClient struct {
	apiURL string
}

var _ Client = (*tektonHubClient)(nil)
var _ Client = (*artifactHubClient)(nil)

func NewTektonHubClient() *tektonHubClient {
	return &tektonHubClient{apiURL: tektonHubURL}
}

func NewArtifactHubClient() *artifactHubClient {
	return &artifactHubClient{apiURL: artifactHubURL}
}

// URL returns the Hub API Server URL
func URL() string {
	return tektonHubURL
}

// GetType returns the type of the Hub Client
func (a *artifactHubClient) GetType() string {
	return ArtifactHubType
}

// GetType returns the type of the Hub Client
func (t *tektonHubClient) GetType() string {
	return TektonHubType
}

// SetURL validates and sets the Artifact Hub apiURL server URL
// URL passed through flag will take precedence over the Artifact Hub API URL
// in config file and default URL
func (a *artifactHubClient) SetURL(apiURL string) error {
	resUrl, err := resolveUrl(apiURL, "ARTIFACT_HUB_API_SERVER", artifactHubURL)
	if err != nil {
		return err
	}

	a.apiURL = resUrl
	return nil
}

// SetURL validates and sets the Tekton Hub apiURL server URL
// URL passed through flag will take precedence over the Tekton Hub API URL
// in config file and default URL
func (t *tektonHubClient) SetURL(apiURL string) error {
	resUrl, err := resolveUrl(apiURL, "TEKTON_HUB_API_SERVER", tektonHubURL)
	if err != nil {
		return err
	}

	t.apiURL = resUrl
	return nil
}

// Get gets data from Artifact Hub
func (a *artifactHubClient) Get(endpoint string) ([]byte, int, error) {
	return get(a.apiURL + endpoint)
}

// Get gets data from Tekton Hub
func (t *tektonHubClient) Get(endpoint string) ([]byte, int, error) {
	return get(t.apiURL + endpoint)
}

func resolveUrl(apiURL, envVariable, defaultUrl string) (string, error) {
	if apiURL != "" {
		_, err := url.ParseRequestURI(apiURL)
		if err != nil {
			return "", err
		}

		return apiURL, nil
	}

	if err := loadConfigFile(); err != nil {
		return "", err
	}

	viper.AutomaticEnv()
	if apiURL := viper.GetString(envVariable); apiURL != "" {
		_, err := url.ParseRequestURI(apiURL)
		if err != nil {
			return "", fmt.Errorf("invalid url set for %s: %s : %v", envVariable, apiURL, err)
		}
		return apiURL, nil
	}

	return defaultUrl, nil
}

func get(url string) ([]byte, int, error) {
	data, status, err := httpGet(url)
	if err != nil {
		return nil, 0, err
	}

	switch status {
	case http.StatusOK:
		err = nil
	case http.StatusNotFound:
		err = fmt.Errorf("No Resource Found")
	case http.StatusInternalServerError:
		err = fmt.Errorf("Internal server Error: consider filing a bug report")
	default:
		err = fmt.Errorf("Invalid Response from server")
	}

	return data, status, err
}

// httpGet gets raw data given the url
func httpGet(url string) ([]byte, int, error) {

	err := loadConfigFile()
	if err != nil {
		return nil, 0, err
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	return data, resp.StatusCode, err
}

// Looks for config file at $HOME/.tekton/hub-config and loads into
// in the environment
func loadConfigFile() error {

	user, err := user.Current()
	if err != nil {
		return err
	}

	// if hub-config file not found, then returns
	path := fmt.Sprintf("%s/%s", user.HomeDir, hubConfigPath)
	if err := godotenv.Load(path); err != nil {
		return nil
	}

	return nil
}
