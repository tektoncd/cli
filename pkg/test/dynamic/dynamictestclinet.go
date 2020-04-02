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

package dynamic

import (
	"github.com/tektoncd/cli/pkg/test/dynamic/clientset"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	k8stest "k8s.io/client-go/testing"
)

type Options struct {
	WatchResource string
	Watcher       *watch.RaceFreeFakeWatcher
}

func (opt *Options) Client(objects ...runtime.Object) (dynamic.Interface, error) {
	dynamicClient := fake.NewSimpleDynamicClient(runtime.NewScheme(), objects...)
	if opt.Watcher != nil {
		dynamicClient.PrependWatchReactor(opt.WatchResource, k8stest.DefaultWatchReactor(opt.Watcher, nil))
	}
	return clientset.New(clientset.WithClient(dynamicClient)), nil
}
