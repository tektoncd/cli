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

package actions

import (
	"sync"

	"github.com/tektoncd/cli/pkg/exitcode"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/restmapper"
)

var (
	doOnce      sync.Once
	apiGroupRes []*restmapper.APIGroupResources
)

// GetGroupVersionResource takes a partial resource, and returns the full resource matching the partial resource, if there's only one match.
func GetGroupVersionResource(gr schema.GroupVersionResource, discovery discovery.DiscoveryInterface) (*schema.GroupVersionResource, error) {
	var err error
	doOnce.Do(func() {
		err = InitializeAPIGroupRes(discovery)
	})
	if err != nil {
		// Discovery contacts the API server, so classify auth/timeout
		// failures here rather than letting them fall through as a generic
		// error; restmapper "no match" errors are passed through unchanged.
		return nil, exitcode.FromAPIError(err)
	}

	rm := restmapper.NewDiscoveryRESTMapper(apiGroupRes)
	gvr, err := rm.ResourceFor(gr)
	if err != nil {
		return nil, exitcode.FromAPIError(err)
	}

	return &gvr, nil
}

// InitializeAPIGroupRes initializes and populates the discovery client.
func InitializeAPIGroupRes(discovery discovery.DiscoveryInterface) error {
	var err error
	apiGroupRes, err = restmapper.GetAPIGroupResources(discovery)
	if err != nil {
		return err
	}
	return nil
}
