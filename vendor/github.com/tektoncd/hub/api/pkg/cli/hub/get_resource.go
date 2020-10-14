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
	"encoding/json"
	"fmt"
	"net/http"

	rclient "github.com/tektoncd/hub/api/gen/http/resource/client"
)

// ResourceOption defines option associated with API to fetch a
// particular resource
type ResourceOption struct {
	Name    string
	Catalog string
	Version string
	Kind    string
}

// ResourceResult defines API response
type ResourceResult struct {
	data    []byte
	status  int
	err     error
	version string
}

// GetResource queries the data using Hub Endpoint
func (h *client) GetResource(opt ResourceOption) ResourceResult {
	data, status, err := h.Get(opt.Endpoint())

	return ResourceResult{
		data:    data,
		version: opt.Version,
		status:  status,
		err:     err,
	}
}

// Endpoint computes the endpoint url using input provided
func (opt ResourceOption) Endpoint() string {
	if opt.Version != "" {
		// API: /resource/<catalog>/<kind>/<name>/<version>
		return fmt.Sprintf("/resource/%s/%s/%s/%s", opt.Catalog, opt.Kind, opt.Name, opt.Version)
	}
	// API: /resource/<catalog>/<kind>/<name>
	return fmt.Sprintf("/resource/%s/%s/%s", opt.Catalog, opt.Kind, opt.Name)
}

// RawURL returns the raw url of the resource yaml file
func (gr *ResourceResult) RawURL() (string, error) {
	if gr.err != nil {
		return "", gr.err
	}

	if gr.status == http.StatusNotFound {
		return "", fmt.Errorf("No Resource Found")
	}

	if gr.version != "" {
		resVersion := rclient.VersionResponse{}
		if err := json.Unmarshal(gr.data, &resVersion); err != nil {
			return "", err
		}
		return *resVersion.RawURL, nil
	}

	res := rclient.ResourceResponse{}
	if err := json.Unmarshal(gr.data, &res); err != nil {
		return "", err
	}
	return *res.LatestVersion.RawURL, nil
}

// Manifest gets the resource from catalog
func (gr *ResourceResult) Manifest() ([]byte, error) {
	rawURL, err := gr.RawURL()
	if err != nil {
		return nil, err
	}

	data, status, err := httpGet(rawURL)

	if err != nil {
		return nil, err
	}

	if status != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch resource from catalog")
	}

	return data, nil
}
