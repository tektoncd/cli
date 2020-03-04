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

package list

import (
	"io"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/printer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/restmapper"
)

func PrintObject(groupResource schema.GroupVersionResource, w io.Writer, p cli.Params, f *cliopts.PrintFlags) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	allres, err := AllObjecs(groupResource, cs, p.Namespace())
	if err != nil {
		return err
	}

	return printer.PrintObject(w, allres, f)
}

func AllObjecs(gr schema.GroupVersionResource, clients *cli.Clients, n string) (*unstructured.UnstructuredList, error) {
	gvr, err := getGVR(gr, clients.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	allRes, err := clients.Dynamic.Resource(*gvr).Namespace(n).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return allRes, nil
}

func getGVR(gr schema.GroupVersionResource, discovery discovery.DiscoveryInterface) (*schema.GroupVersionResource, error) {
	apiGroupRes, err := restmapper.GetAPIGroupResources(discovery)
	if err != nil {
		return nil, err
	}

	rm := restmapper.NewDiscoveryRESTMapper(apiGroupRes)
	gvr, err := rm.ResourceFor(gr)
	if err != nil {
		return nil, err
	}

	return &gvr, nil
}
