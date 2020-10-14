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

	rclient "github.com/tektoncd/hub/api/gen/http/resource/client"
)

// SearchOption defines option associated with query API
type SearchOption struct {
	Name  string
	Kinds []string
	Tags  []string
	Match string
	Limit uint
}

// SearchResponse is the array of ResourceResponse
type SearchResponse = []rclient.ResourceResponse

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
	sr.resources = SearchResponse{}
	if sr.status == http.StatusNotFound {
		return sr.resources, sr.err
	}

	sr.err = json.Unmarshal(sr.data, &sr.resources)
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
	if so.Limit != 0 {
		v.Set("limit", strconv.FormatUint(uint64(so.Limit), 10))
	}

	v.Set("match", so.Match)

	return "/query?" + v.Encode()
}

func addArraytoURL(param string, arr []string, v url.Values) {
	for _, a := range arr {
		v.Add(param, a)
	}
}
