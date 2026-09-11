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

package test

import (
	"github.com/tektoncd/cli/pkg/cmd/hub/gvr"
	"github.com/tektoncd/cli/pkg/cmd/hub/kube"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/dynamic"
)

type fakeClients struct {
	tekton    versioned.Interface
	dynamic   dynamic.Interface
	namespace string
}

var _ kube.ClientSet = (*fakeClients)(nil)

func (p *fakeClients) Dynamic() dynamic.Interface {
	return p.dynamic
}

func (p *fakeClients) Tekton() versioned.Interface {
	return p.tekton
}
func (p *fakeClients) Namespace() string {
	return p.namespace
}

func FakeClientSet(tekton versioned.Interface, dynamic dynamic.Interface, namespace string) kube.ClientSet {
	if tekton != nil {
		if err := gvr.InitializeAPIGroupRes(tekton.Discovery()); err != nil {
			return nil
		}
	}

	return &fakeClients{
		tekton:    tekton,
		dynamic:   dynamic,
		namespace: namespace,
	}
}
