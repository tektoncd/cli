// Copyright © 2021 The Tekton Authors.
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

	cclient "github.com/tektoncd/hub/api/v1/gen/http/catalog/client"
)

const (
	tektonHubCatEndpoint    = "/v1/catalogs"
	artifactHubCatEndpoint  = "/api/v1/repositories/search"
	artifactHubTaskType     = 7
	artifactHubPipelineType = 11
)

type CatalogResult struct {
	data    []byte
	status  int
	err     error
	Catalog CatalogData
}
type CatalogData = cclient.ListResponseBody

type artifactHubCatalogResponse struct {
	Name string `json:"name,omitempty"`
}

func (c *tektonHubclient) GetAllCatalogs() CatalogResult {
	data, status, err := c.Get(tektonHubCatEndpoint)
	if status == http.StatusNotFound {
		err = nil
	}

	return CatalogResult{data: data, status: status, err: err}
}

// Typed returns unmarshalled API response as CatalogResponse
func (cr *CatalogResult) Type() (CatalogData, error) {
	if cr.Catalog.Data != nil || cr.err != nil {
		return cr.Catalog, cr.err
	}
	res := &CatalogData{}

	cr.err = json.Unmarshal(cr.data, res)
	cr.Catalog.Data = res.Data

	return cr.Catalog, cr.err
}

func (a *artifactHubClient) GetCatalogsList() ([]string, error) {
	data, _, err := a.Get(fmt.Sprintf("%s?kind=%v&kind=%v", artifactHubCatEndpoint, artifactHubTaskType, artifactHubPipelineType))
	if err != nil {
		return nil, err
	}

	resp := []artifactHubCatalogResponse{}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling json response: %w", err)
	}

	var cat []string
	for _, r := range resp {
		cat = append(cat, r.Name)
	}

	return cat, nil
}

func (t *tektonHubclient) GetCatalogsList() ([]string, error) {
	// Get all catalogs
	c := t.GetAllCatalogs()

	// Unmarshal the data
	var err error
	typed, err := c.Type()
	if err != nil {
		return nil, err
	}

	var data = struct {
		Catalogs CatalogData
	}{
		Catalogs: typed,
	}

	Catalog := data.Catalogs
	// Get all catalog names
	var cat []string
	for i := range Catalog.Data {
		cat = append(cat, *Catalog.Data[i].Name)
	}

	return cat, nil
}
