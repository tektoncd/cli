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
	"sort"
	"strings"

	rclient "github.com/tektoncd/hub/api/v1/gen/http/resource/client"
)

// ResourceOption defines option associated with API to fetch a
// particular resource
type ResourceOption struct {
	Name            string
	Catalog         string
	Version         string
	Kind            string
	PipelineVersion string
}

// ResourceResult defines API response
type ResourceResult struct {
	data                    []byte
	status                  int
	err                     error
	version                 string
	set                     bool
	resourceData            *ResourceData
	resourceWithVersionData *ResourceWithVersionData
}

type ResourceVersionOptions struct {
	hubResVersionsRes ResourceVersionResult
	hubResVersions    *ResVersions
}

// resResponse is the response of API when finding a resource
type resResponse = rclient.ByCatalogKindNameResponseBody

// resVersionResponse is the response of API when finding a resource
// with a specific version
type resVersionResponse = rclient.ByCatalogKindNameVersionResponseBody

// ResourceData is the response of API when finding a resource
type ResourceData = rclient.ResourceDataResponseBody

// ResourceWithVersionData is the response of API when finding a resource
// with a specific version
type ResourceWithVersionData = rclient.ResourceVersionDataResponseBody

// GetResource queries the data using Hub Endpoint
func (c *client) GetResource(opt ResourceOption) ResourceResult {
	data, status, err := c.Get(opt.Endpoint())

	return ResourceResult{
		data:    data,
		version: opt.Version,
		status:  status,
		err:     err,
		set:     false,
	}
}

// Endpoint computes the endpoint url using input provided
func (opt ResourceOption) Endpoint() string {
	if opt.Version != "" {
		// API: /resource/<catalog>/<kind>/<name>/<version>
		return fmt.Sprintf("/v1/resource/%s/%s/%s/%s", opt.Catalog, opt.Kind, opt.Name, opt.Version)
	}
	if opt.PipelineVersion != "" {
		opt.PipelineVersion = strings.TrimLeft(opt.PipelineVersion, "v")
		// API: /resource/<catalog>/<kind>/<name>?pipelinesversion=<version>
		return fmt.Sprintf("/v1/resource/%s/%s/%s?pipelinesversion=%s", opt.Catalog, opt.Kind, opt.Name, opt.PipelineVersion)
	}
	// API: /resource/<catalog>/<kind>/<name>
	return fmt.Sprintf("/v1/resource/%s/%s/%s", opt.Catalog, opt.Kind, opt.Name)
}

func (rr *ResourceResult) unmarshalData() error {
	if rr.err != nil {
		return rr.err
	}
	if rr.set {
		return nil
	}

	if rr.status == http.StatusNotFound {
		return fmt.Errorf("No Resource Found")
	}

	// API Response when version is not mentioned, will fetch latest by default
	if rr.version == "" {
		res := resResponse{}
		if err := json.Unmarshal(rr.data, &res); err != nil {
			return err
		}
		rr.resourceData = res.Data
		rr.set = true
		return nil
	}

	// API Response when a specific version is mentioned
	res := resVersionResponse{}
	if err := json.Unmarshal(rr.data, &res); err != nil {
		return err
	}
	rr.resourceWithVersionData = res.Data
	rr.set = true
	return nil
}

// RawURL returns the raw url of the resource yaml file
func (rr *ResourceResult) RawURL() (string, error) {
	if err := rr.unmarshalData(); err != nil {
		return "", err
	}

	if rr.version != "" {
		return *rr.resourceWithVersionData.RawURL, nil
	}
	return *rr.resourceData.LatestVersion.RawURL, nil
}

// Manifest gets the resource from catalog
func (rr *ResourceResult) Manifest() ([]byte, error) {
	rawURL, err := rr.RawURL()
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

// Resource returns the resource found
func (rr *ResourceResult) Resource() (interface{}, error) {
	if err := rr.unmarshalData(); err != nil {
		return "", err
	}

	if rr.version != "" {
		return *rr.resourceWithVersionData, nil
	}
	return *rr.resourceData, nil
}

// ResourceVersion returns the resource version found
func (rr *ResourceResult) ResourceVersion() (string, error) {
	if err := rr.unmarshalData(); err != nil {
		return "", err
	}

	if rr.version != "" {
		return *rr.resourceWithVersionData.Version, nil
	}
	return *rr.resourceData.LatestVersion.Version, nil
}

// MinPipelinesVersion returns the minimum pipeline version the resource is compatible
func (rr *ResourceResult) MinPipelinesVersion() (string, error) {
	if err := rr.unmarshalData(); err != nil {
		return "", err
	}

	if rr.version != "" {
		return *rr.resourceWithVersionData.MinPipelinesVersion, nil
	}
	return *rr.resourceData.LatestVersion.MinPipelinesVersion, nil
}

func (hubClient *client) GetResourcesList(so SearchOption) ([]string, error) {
	// Get all resources
	result := hubClient.Search(SearchOption{
		Kinds:   so.Kinds,
		Catalog: so.Catalog,
	})

	typed, err := result.Typed()
	if err != nil {
		return nil, err
	}

	var data = struct {
		Resources SearchResponse
	}{
		Resources: typed,
	}

	// Get all resource names
	var resources []string
	for i := range data.Resources {
		resources = append(resources, *data.Resources[i].Name)
	}

	return resources, nil
}

func (hubClient *client) GetResourceVersionslist(r ResourceOption) ([]string, error) {
	opts := &ResourceVersionOptions{}
	// Get the resource versions
	opts.hubResVersionsRes = hubClient.GetResourceVersions(ResourceOption{
		Name:    r.Name,
		Catalog: r.Catalog,
		Kind:    r.Kind,
	})

	var err error
	opts.hubResVersions, err = opts.hubResVersionsRes.ResourceVersions()
	if err != nil {
		return nil, err
	}

	var ver []string
	for i := range opts.hubResVersions.Versions {
		ver = append(ver, *opts.hubResVersions.Versions[i].Version)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ver)))

	return ver, nil
}
