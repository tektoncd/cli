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
type ResourceResult interface {
	RawURL() (string, error)
	Manifest() ([]byte, error)
	Resource() (interface{}, error)
	ResourceYaml() (string, error)
	ResourceVersion() (string, error)
	MinPipelinesVersion() (string, error)
	Org() (string, error)
	UnmarshalData() error
}

// TektonHubResourceResult defines Tekton Hub API response
type TektonHubResourceResult struct {
	data                    []byte
	yaml                    []byte
	status                  int
	yamlStatus              int
	yamlErr                 error
	err                     error
	version                 string
	set                     bool
	resourceData            *ResourceData
	resourceWithVersionData *ResourceWithVersionData
	ResourceContent         *ResourceContent
}

// ArtifactHubResourceResult defines Artifact Hub API response
type ArtifactHubResourceResult struct {
	data    []byte
	status  int
	err     error
	version string
	set     bool
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

type ResourceContent = rclient.ResourceContentResponseBody

// ResourceWithVersionData is the response of API when finding a resource
// with a specific version
type ResourceWithVersionData = rclient.ResourceVersionDataResponseBody

type resourceYaml = rclient.ByCatalogKindNameVersionYamlResponseBody

type ArtifactHubPkgResponse struct {
	Name              string               `json:"name,omitempty"`
	Data              ArtifactHubPkgData   `json:"data,omitempty"`
	AvailableVersions []ArtifactHubVersion `json:"available_versions,omitempty"`
	Repository        ArtifactHubRepo      `json:"repository,omitempty"`
}

type ArtifactHubRepo struct {
	Org string `json:"organization_name,omitempty"`
}

type ArtifactHubPkgData struct {
	PipelineMinVer string   `json:"pipelines.minVersion"`
	ManifestRaw    string   `json:"manifestRaw"`
	Platforms      []string `json:"platforms"`
}

type ArtifactHubVersion struct {
	Version string `json:"version"`
}

// GetResource queries the data using Artifact Hub Endpoint
func (a *artifactHubClient) GetResource(opt ResourceOption) ResourceResult {
	panic("GetResource() not implemented for artifact type")
}

// GetResource queries the data using Tekton Hub Endpoint
func (t *tektonHubClient) GetResource(opt ResourceOption) ResourceResult {
	data, status, err := t.Get(opt.Endpoint())

	return &TektonHubResourceResult{
		data:    data,
		version: opt.Version,
		status:  status,
		err:     err,
		set:     false,
	}
}

// GetResourceYaml queries the resource yaml using Artifact Hub Endpoint
func (a *artifactHubClient) GetResourceYaml(opt ResourceOption) ResourceResult {
	data, status, err := a.Get(fmt.Sprintf("%s-%s/%s/%s/%s", artifactHubCatInfoEndpoint, opt.Kind, opt.Catalog, opt.Name, opt.Version))

	return &ArtifactHubResourceResult{
		data:    data,
		version: opt.Version,
		status:  status,
		err:     err,
		set:     false,
	}
}

// GetResourceYaml queries the resource yaml using Tekton Hub Endpoint
func (t *tektonHubClient) GetResourceYaml(opt ResourceOption) ResourceResult {

	yaml, yamlStatus, yamlErr := t.Get(fmt.Sprintf("/v1/resource/%s/%s/%s/%s/yaml", opt.Catalog, opt.Kind, opt.Name, opt.Version))
	data, status, err := t.Get(opt.Endpoint())

	return &TektonHubResourceResult{
		data:       data,
		yaml:       yaml,
		version:    opt.Version,
		status:     status,
		err:        err,
		yamlStatus: yamlStatus,
		yamlErr:    yamlErr,
		set:        false,
	}
}

// GetResourcesList queries the resources in the catalog using Artifact Hub Endpoint
func (a *artifactHubClient) GetResourcesList(so SearchOption) ([]string, error) {
	panic("GetResourcesList() not implemented for artifact type")
}

// GetResourcesList queries the resources in the catalog using Tekton Hub Endpoint
func (t *tektonHubClient) GetResourcesList(so SearchOption) ([]string, error) {
	// Get all resources
	result := t.Search(SearchOption{
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

// GetResourceVersionslist queries the versions of a resource using Artifact Hub Endpoint
func (a *artifactHubClient) GetResourceVersionslist(r ResourceOption) ([]string, error) {
	data, _, err := a.Get(fmt.Sprintf("%s-%s/%s/%s", artifactHubCatInfoEndpoint, r.Kind, r.Catalog, r.Name))
	if err != nil {
		return nil, err
	}

	versions, err := findArtifactHubResourceVersions(data)
	if err != nil {
		return nil, err
	}

	return versions, nil
}

// GetResourceVersionslist queries the versions of a resource using Tekton Hub Endpoint
func (t *tektonHubClient) GetResourceVersionslist(r ResourceOption) ([]string, error) {
	opts := &ResourceVersionOptions{}
	// Get the resource versions
	opts.hubResVersionsRes = t.GetResourceVersions(ResourceOption{
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

// UnmarshalData unmarshals the data embedded in TektonHubResourceResult
func (rr *TektonHubResourceResult) UnmarshalData() error {
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

// UnmarshalData unmarshals the data embedded in ArtifactHubResourceResult
func (rr *ArtifactHubResourceResult) UnmarshalData() error {
	panic("UnmarshalData() not implemented for artifact type")
}

// RawURL returns the raw url of the resource yaml file from TektonHubResourceResult
func (rr *TektonHubResourceResult) RawURL() (string, error) {
	if err := rr.UnmarshalData(); err != nil {
		return "", err
	}

	if rr.version != "" {
		return *rr.resourceWithVersionData.RawURL, nil
	}
	return *rr.resourceData.LatestVersion.RawURL, nil
}

// RawURL returns the raw url of the resource yaml file from ArtifactHubResourceResult
func (rr *ArtifactHubResourceResult) RawURL() (string, error) {
	panic("RawURL() not implemented for artifact type")
}

// Manifest gets the raw resource of the catalog from TektonHubResourceResult
func (rr *TektonHubResourceResult) Manifest() ([]byte, error) {
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

// Manifest gets the raw resource of the catalog from ArtifactHubResourceResult
func (rr *ArtifactHubResourceResult) Manifest() ([]byte, error) {
	panic("Manifest() not implemented for artifact type")
}

// Resource returns the resource found from TektonHubResourceResult
func (rr *TektonHubResourceResult) Resource() (interface{}, error) {
	if err := rr.UnmarshalData(); err != nil {
		return "", err
	}

	if rr.version != "" {
		return *rr.resourceWithVersionData, nil
	}
	return *rr.resourceData, nil
}

// Resource returns the resource found from ArtifactHubResourceResult
func (rr *ArtifactHubResourceResult) Resource() (interface{}, error) {
	panic("Resource() not implemented for artifact type")
}

// ResourceYaml returns the resource yaml file found from TektonHubResourceResult
func (rr *TektonHubResourceResult) ResourceYaml() (string, error) {

	if rr.yamlErr != nil {
		return "", rr.err
	}
	if rr.set {
		return "", nil
	}

	if rr.yamlStatus == http.StatusNotFound {
		return "", fmt.Errorf("No Resource Found")
	}

	res := resourceYaml{}
	if err := json.Unmarshal(rr.yaml, &res); err != nil {
		return "", err
	}
	rr.ResourceContent = res.Data

	return *rr.ResourceContent.Yaml, nil
}

// ResourceYaml returns the resource yaml file found from ArtifactHubResourceResult
func (rr *ArtifactHubResourceResult) ResourceYaml() (string, error) {
	if err := rr.validateData(); err != nil {
		return "", err
	}
	res := ArtifactHubPkgResponse{}
	if err := json.Unmarshal(rr.data, &res); err != nil {
		return "", err
	}
	return res.Data.ManifestRaw, nil
}

// ResourceVersion returns the resource version found from TektonHubResourceResult
func (rr *TektonHubResourceResult) ResourceVersion() (string, error) {
	if err := rr.UnmarshalData(); err != nil {
		return "", err
	}

	if rr.version != "" {
		return *rr.resourceWithVersionData.Version, nil
	}
	return *rr.resourceData.LatestVersion.Version, nil
}

// ResourceVersion returns the resource version found from ArtifactHubResourceResult
func (rr *ArtifactHubResourceResult) ResourceVersion() (string, error) {
	if err := rr.validateData(); err != nil {
		return "", err
	}
	if rr.version == "" {
		versions, err := findArtifactHubResourceVersions(rr.data)
		if err != nil {
			return "", err
		}
		return versions[0], nil
	}
	return rr.version, nil
}

// MinPipelinesVersion returns the minimum pipeline version the resource is compatible from TektonHubResourceResult
func (rr *TektonHubResourceResult) MinPipelinesVersion() (string, error) {
	if err := rr.UnmarshalData(); err != nil {
		return "", err
	}

	if rr.version != "" {
		return *rr.resourceWithVersionData.MinPipelinesVersion, nil
	}
	return *rr.resourceData.LatestVersion.MinPipelinesVersion, nil
}

// MinPipelinesVersion returns the minimum pipeline version the resource is compatible from ArtifactHubResourceResult
func (rr *ArtifactHubResourceResult) MinPipelinesVersion() (string, error) {
	if err := rr.validateData(); err != nil {
		return "", err
	}
	resp := ArtifactHubPkgResponse{}
	if err := json.Unmarshal(rr.data, &resp); err != nil {
		return "", err
	}
	if resp.Data.PipelineMinVer == "" {
		return "", fmt.Errorf("Minimum pipeline version is not specified in the resource")
	}
	return resp.Data.PipelineMinVer, nil
}

// Org returns the organization of the catalog from Artifact Hub
func (rr *ArtifactHubResourceResult) Org() (string, error) {
	if err := rr.validateData(); err != nil {
		return "", err
	}
	res := ArtifactHubPkgResponse{}
	if err := json.Unmarshal(rr.data, &res); err != nil {
		return "", err
	}
	return res.Repository.Org, nil
}

// Org returns the organization of the catalog from Tekton Hub
func (rr *TektonHubResourceResult) Org() (string, error) {
	// TODO: implement org() for Tekton Hub
	return "", nil
}

func (rr *ArtifactHubResourceResult) validateData() error {
	if rr.err != nil {
		return rr.err
	}
	if rr.set {
		return nil
	}
	if rr.status == http.StatusNotFound {
		return fmt.Errorf("No Resource Found")
	}

	return nil
}

func findArtifactHubResourceVersions(data []byte) ([]string, error) {
	resp := ArtifactHubPkgResponse{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("Error unmarshalling json response: %w", err)
	}
	if len(resp.AvailableVersions) == 0 {
		return nil, fmt.Errorf("No available versions found in Artifact Hub")
	}

	var versions []string
	for _, r := range resp.AvailableVersions {
		versions = append(versions, r.Version)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))

	return versions, nil
}
