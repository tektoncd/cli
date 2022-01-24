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
	"io/ioutil"
	"net/http"
	"net/url"
	"os/user"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	// hubURL - Hub API Server URL
	hubURL        = "https://api.hub.tekton.dev"
	hubConfigPath = ".tekton/hub-config"
)

type Client interface {
	SetURL(u string) error
	Get(endpoint string) ([]byte, int, error)
	GetCatalogsList() ([]string, error)
	Search(opt SearchOption) SearchResult
	GetResource(opt ResourceOption) ResourceResult
	GetResourcesList(opt SearchOption) ([]string, error)
	GetResourceVersions(opt ResourceOption) ResourceVersionResult
	GetResourceVersionslist(opt ResourceOption) ([]string, error)
}

type client struct {
	apiURL string
}

var _ Client = (*client)(nil)

func NewClient() *client {
	return &client{apiURL: hubURL}
}

// URL returns the Hub API Server URL
func URL() string {
	return hubURL
}

// SetURL validates and sets the hub apiURL server URL
// URL passed through flag will take precedence over the hub API URL
// in config file and default URL
func (h *client) SetURL(apiURL string) error {

	if apiURL != "" {
		_, err := url.ParseRequestURI(apiURL)
		if err != nil {
			return err
		}
		h.apiURL = apiURL
		return nil
	}

	if err := loadConfigFile(); err != nil {
		return err
	}

	viper.AutomaticEnv()
	if apiURL := viper.GetString("HUB_API_SERVER"); apiURL != "" {
		_, err := url.ParseRequestURI(apiURL)
		if err != nil {
			return fmt.Errorf("invalid url set for HUB_API_SERVER: %s : %v", apiURL, err)
		}
		h.apiURL = apiURL
	}

	return nil
}

// Get gets data from Hub
func (h *client) Get(endpoint string) ([]byte, int, error) {
	data, status, err := httpGet(h.apiURL + endpoint)
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

	data, err := ioutil.ReadAll(resp.Body)
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
