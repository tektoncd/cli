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

package actions

import (
	"context"
	"io"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/printer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func PrintObject(groupResource schema.GroupVersionResource, obj string, w io.Writer, p cli.Params, f *cliopts.PrintFlags, ns string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	res, err := Get(groupResource, cs, obj, ns, metav1.GetOptions{})
	if err != nil {
		return err
	}

	return printer.PrintObject(w, res, f)
}

func Get(gr schema.GroupVersionResource, clients *cli.Clients, objname, ns string, op metav1.GetOptions) (*unstructured.Unstructured, error) {
	gvr, err := GetGroupVersionResource(gr, clients.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	obj, err := clients.Dynamic.Resource(*gvr).Namespace(ns).Get(context.Background(), objname, op)
	if err != nil {
		return nil, err
	}

	return obj, nil
}
