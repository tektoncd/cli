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
	"strconv"

	rclient "github.com/tektoncd/hub/api/v1/gen/http/resource/client"
)

// resVersionsResponse is the response of API when finding resource versions
type resVersionsResponse = rclient.VersionsByIDResponseBody

// ResVersions is the data in API response consisting of list of versions
type ResVersions = rclient.VersionsResponseBody

// ResourceVersionResult defines API response
type ResourceVersionResult struct {
	rr       ResourceResult
	data     []byte
	status   int
	err      error
	set      bool
	versions *ResVersions
}

// GetResourceVersion queries the data using Hub Endpoint
func (c *client) GetResourceVersions(opt ResourceOption) ResourceVersionResult {

	rvr := ResourceVersionResult{set: false}

	rvr.rr = c.GetResource(opt)
	if rvr.err = rvr.rr.unmarshalData(); rvr.err != nil {
		return rvr
	}

	var resID uint
	if rvr.rr.version != "" {
		resID = *rvr.rr.resourceWithVersionData.Resource.ID
	} else {
		resID = *rvr.rr.resourceData.ID
	}

	rvr.data, rvr.status, rvr.err = c.Get(resVersionsEndpoint(resID))

	return rvr
}

// Endpoint computes the endpoint url using input provided
func resVersionsEndpoint(rID uint) string {
	return fmt.Sprintf("/v1/resource/%s/versions", strconv.FormatUint(uint64(rID), 10))
}

func (rvr *ResourceVersionResult) unmarshalData() error {
	if rvr.err != nil {
		return rvr.err
	}
	if rvr.set {
		return nil
	}

	res := resVersionsResponse{}
	if err := json.Unmarshal(rvr.data, &res); err != nil {
		return err
	}
	rvr.versions = res.Data
	rvr.set = true
	return nil
}

// ResourceVersions returns list of all versions of the resource
func (rvr *ResourceVersionResult) ResourceVersions() (*ResVersions, error) {

	if err := rvr.unmarshalData(); err != nil {
		return nil, err
	}

	return rvr.versions, nil
}

// VersionManifest gets the resource from catalog for the resource's version passed
func (rvr *ResourceVersionResult) VersionManifest(version string) ([]byte, error) {

	if err := rvr.unmarshalData(); err != nil {
		return nil, err
	}

	var rawURL string
	for _, v := range rvr.versions.Versions {
		if version == *v.Version {
			rawURL = *v.RawURL
			break
		}
	}
	if rawURL == "" {
		return nil, fmt.Errorf("resource version (v%s) not found", version)
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
