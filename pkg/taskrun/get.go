// Copyright © 2019-2020 The Tekton Authors.
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

package taskrun

import (
	"context"
	"fmt"
	"os"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// It will fetch the resource based on the api available and return v1beta1 form
func Get(c *cli.Clients, trname string, opts metav1.GetOptions, ns string) (*v1beta1.TaskRun, error) {
	gvr, err := actions.GetGroupVersionResource(trGroupResource, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if gvr.Version == "v1alpha1" {
		taskrun, err := getV1alpha1(c, trname, opts, ns)
		if err != nil {
			return nil, err
		}
		var taskrunConverted v1beta1.TaskRun
		err = taskrun.ConvertUp(context.Background(), &taskrunConverted)
		if err != nil {
			return nil, err
		}
		return &taskrunConverted, nil
	}
	return GetV1beta1(c, trname, opts, ns)
}

// It will fetch the resource in v1beta1 struct format
func GetV1beta1(c *cli.Clients, trname string, opts metav1.GetOptions, ns string) (*v1beta1.TaskRun, error) {
	unstructuredTR, err := actions.Get(trGroupResource, c, trname, ns, opts)
	if err != nil {
		return nil, err
	}

	var taskrun *v1beta1.TaskRun
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredTR.UnstructuredContent(), &taskrun); err != nil {
		fmt.Fprintf(os.Stderr, "failed to get taskrun from %s namespace \n", ns)
		return nil, err
	}
	return taskrun, nil
}

// It will fetch the resource in v1alpha1 struct format
func getV1alpha1(c *cli.Clients, trname string, opts metav1.GetOptions, ns string) (*v1alpha1.TaskRun, error) {
	unstructuredTR, err := actions.Get(trGroupResource, c, trname, ns, opts)
	if err != nil {
		return nil, err
	}

	var taskrun *v1alpha1.TaskRun
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredTR.UnstructuredContent(), &taskrun); err != nil {
		fmt.Fprintf(os.Stderr, "failed to get taskrun from %s namespace \n", ns)
		return nil, err
	}
	return taskrun, nil
}
