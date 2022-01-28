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
	"net/http"
	"net/url"
	"strconv"

	rclient "github.com/tektoncd/hub/api/v1/gen/http/resource/client"
)

// SearchOption defines option associated with query API
type SearchOption struct {
	Name       string
	Kinds      []string
	Tags       []string
	Categories []string
	Platforms  []string
	Match      string
	Limit      uint
	Catalog    string
}

// SearchResponse is the data object which is the search result
type SearchResponse = rclient.ResourceDataCollectionResponseBody

// queryAPIResponse is the response from the API
type queryAPIResponse = rclient.QueryResponseBody

// SearchResult defines API raw response, unmarshalled reponse, and error
type SearchResult struct {
	data      []byte
	status    int
	resources SearchResponse
	err       error
}

// Search queries the data using Hub Endpoint
func (h *client) Search(so SearchOption) SearchResult {
	data, status, err := h.Get(so.Endpoint())
	if status == http.StatusNotFound {
		err = nil
	}
	return SearchResult{data: data, status: status, err: err}
}

// Raw returns API response as byte array
func (sr *SearchResult) Raw() ([]byte, error) {
	return sr.data, sr.err
}

// Typed returns unmarshalled API response as SearchResponse
func (sr *SearchResult) Typed() (SearchResponse, error) {
	if sr.resources != nil || sr.err != nil {
		return sr.resources, sr.err
	}
	res := &queryAPIResponse{}
	if sr.status == http.StatusNotFound {
		return sr.resources, sr.err
	}

	sr.err = json.Unmarshal(sr.data, res)
	sr.resources = res.Data

	return sr.resources, sr.err
}

// Endpoint computes the endpoint url using input provided
func (so SearchOption) Endpoint() string {

	v := url.Values{}

	if so.Name != "" {
		v.Set("name", so.Name)
	}
	if len(so.Kinds) != 0 {
		addArraytoURL("kinds", so.Kinds, v)
	}
	if len(so.Tags) != 0 {
		addArraytoURL("tags", so.Tags, v)
	}
	if len(so.Categories) != 0 {
		addArraytoURL("categories", so.Categories, v)
	}
	if len(so.Platforms) != 0 {
		addArraytoURL("platforms", so.Platforms, v)
	}
	if so.Limit != 0 {
		v.Set("limit", strconv.FormatUint(uint64(so.Limit), 10))
	}
	if so.Catalog != "" {
		v.Set("catalogs", so.Catalog)
	}

	v.Set("match", so.Match)

	return "/v1/query?" + v.Encode()
}

func addArraytoURL(param string, arr []string, v url.Values) {
	for _, a := range arr {
		v.Add(param, a)
	}
}
