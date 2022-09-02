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

package clustertask

import (
	"fmt"
	"os"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var clustertaskGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "clustertasks"}

func GetAllClusterTaskNames(p cli.Params) ([]string, error) {
	cs, err := p.Clients()
	if err != nil {
		return nil, err
	}

	clustertasks, err := List(cs, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := []string{}
	for _, item := range clustertasks.Items {
		ret = append(ret, item.ObjectMeta.Name)
	}
	return ret, nil
}

func List(c *cli.Clients, opts metav1.ListOptions) (*v1beta1.ClusterTaskList, error) {
	unstructuredCT, err := actions.List(clustertaskGroupResource, c.Dynamic, c.Tekton.Discovery(), "", opts)
	if err != nil {
		return nil, err
	}

	var clustertasks *v1beta1.ClusterTaskList
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredCT.UnstructuredContent(), &clustertasks); err != nil {
		return nil, err
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list clustertasks\n")
		return nil, err
	}

	return clustertasks, nil
}

// It will fetch the ClusterTask based on ClusterTask name
func Get(c *cli.Clients, clustertaskname string, opts metav1.GetOptions) (*v1beta1.ClusterTask, error) {
	unstructuredCT, err := actions.Get(clustertaskGroupResource, c.Dynamic, c.Tekton.Discovery(), clustertaskname, "", opts)
	if err != nil {
		return nil, err
	}

	var clustertask *v1beta1.ClusterTask
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredCT.UnstructuredContent(), &clustertask); err != nil {
		fmt.Fprintf(os.Stderr, "failed to get clustertask\n")
		return nil, err
	}
	return clustertask, nil
}

func Create(c *cli.Clients, ct *v1beta1.ClusterTask, opts metav1.CreateOptions) (*v1beta1.ClusterTask, error) {
	object, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ct)
	if err != nil {
		return nil, err
	}
	unstructuredCT := &unstructured.Unstructured{
		Object: object,
	}
	newUnstructuredCT, err := actions.Create(clustertaskGroupResource, c, unstructuredCT, "", opts)
	if err != nil {
		return nil, err
	}
	var clusterTask *v1beta1.ClusterTask
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(newUnstructuredCT.UnstructuredContent(), &clusterTask); err != nil {
		return nil, err
	}
	return clusterTask, nil
}
