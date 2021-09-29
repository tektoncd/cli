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

package clustertriggerbinding

import (
	"fmt"
	"os"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var clustertriggerbindingGroupResource = schema.GroupVersionResource{Group: "triggers.tekton.dev", Resource: "clustertriggerbindings"}

func GetAllClusterTriggerBindingNames(client *cli.Clients) ([]string, error) {
	ps, err := List(client, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := []string{}
	for _, item := range ps.Items {
		ret = append(ret, item.ObjectMeta.Name)
	}
	return ret, nil
}

func List(c *cli.Clients, opts metav1.ListOptions) (*v1beta1.ClusterTriggerBindingList, error) {
	unstructuredCTB, err := actions.List(clustertriggerbindingGroupResource, c.Dynamic, c.Triggers.Discovery(), "", opts)
	if err != nil {
		return nil, err
	}

	var clustertriggerbindings *v1beta1.ClusterTriggerBindingList
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredCTB.UnstructuredContent(), &clustertriggerbindings); err != nil {
		return nil, err
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list clustertriggerbindings\n")
		return nil, err
	}

	return clustertriggerbindings, nil
}

func Get(c *cli.Clients, ctbname string, opts metav1.GetOptions) (*v1beta1.ClusterTriggerBinding, error) {
	unstructuredCTB, err := actions.Get(clustertriggerbindingGroupResource, c.Dynamic, c.Triggers.Discovery(), ctbname, "", opts)
	if err != nil {
		return nil, err
	}

	var ctb *v1beta1.ClusterTriggerBinding
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredCTB.UnstructuredContent(), &ctb); err != nil {
		fmt.Fprintf(os.Stderr, "failed to get clustertriggerbinding\n")
		return nil, err
	}
	return ctb, nil
}
