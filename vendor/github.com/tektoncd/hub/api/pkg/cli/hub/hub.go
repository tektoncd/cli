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
)

const (
	// hubURL - Hub API Server URL
	hubURL = "https://api.hub.tekton.dev"
)

type Client interface {
	SetURL(u string) error
	Get(endpoint string) ([]byte, int, error)
	Search(opt SearchOption) SearchResult
	GetResource(opt ResourceOption) ResourceResult
	GetResourceVersions(opt ResourceOption) ResourceVersionResult
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
func (h *client) SetURL(apiURL string) error {

	_, err := url.ParseRequestURI(apiURL)
	if err != nil {
		return err
	}

	h.apiURL = apiURL
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
