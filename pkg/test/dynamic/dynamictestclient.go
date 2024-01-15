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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	k8stest "k8s.io/client-go/testing"
)

type Options struct {
	WatchResource    string
	Watcher          watch.Interface
	WatchReactionFun k8stest.WatchReactionFunc
	PrependReactors  []PrependOpt
	AddReactorRes    string
	AddReactorVerb   string
	AddReactorFun    k8stest.ReactionFunc
}

type PrependOpt struct {
	Resource string
	Verb     string
	Action   k8stest.ReactionFunc
}

func (opt *Options) Client(objects ...runtime.Object) (dynamic.Interface, error) {
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			{Group: "tekton.dev", Version: "v1beta1", Resource: "clustertasks"}:                    "ClusterTaskList",
			{Group: "tekton.dev", Version: "v1beta1", Resource: "tasks"}:                           "TaskList",
			{Group: "tekton.dev", Version: "v1beta1", Resource: "taskruns"}:                        "TaskRunList",
			{Group: "tekton.dev", Version: "v1beta1", Resource: "customruns"}:                      "CustomRunList",
			{Group: "tekton.dev", Version: "v1beta1", Resource: "pipelines"}:                       "PipelineList",
			{Group: "tekton.dev", Version: "v1beta1", Resource: "pipelineruns"}:                    "PipelineRunList",
			{Group: "tekton.dev", Version: "v1", Resource: "tasks"}:                                "TaskList",
			{Group: "tekton.dev", Version: "v1", Resource: "taskruns"}:                             "TaskRunList",
			{Group: "tekton.dev", Version: "v1", Resource: "pipelines"}:                            "PipelineList",
			{Group: "tekton.dev", Version: "v1", Resource: "pipelineruns"}:                         "PipelineRunList",
			{Group: "triggers.tekton.dev", Version: "v1beta1", Resource: "triggertemplates"}:       "TriggerTemplateList",
			{Group: "triggers.tekton.dev", Version: "v1beta1", Resource: "triggerbindings"}:        "TriggerBindingList",
			{Group: "triggers.tekton.dev", Version: "v1beta1", Resource: "clustertriggerbindings"}: "ClusterTriggerBindingList",
			{Group: "triggers.tekton.dev", Version: "v1beta1", Resource: "eventlisteners"}:         "EventListenerList",
		},
		objects...,
	)
	if opt.Watcher != nil {
		dynamicClient.PrependWatchReactor(opt.WatchResource, k8stest.DefaultWatchReactor(opt.Watcher, nil))
	}
	if opt.WatchReactionFun != nil {
		dynamicClient.PrependWatchReactor(opt.WatchResource, opt.WatchReactionFun)
	}
	if len(opt.PrependReactors) != 0 {
		for _, res := range opt.PrependReactors {
			dynamicClient.PrependReactor(res.Verb, res.Resource, res.Action)
		}
	}
	if opt.AddReactorFun != nil {
		dynamicClient.AddReactor(opt.AddReactorVerb, opt.AddReactorRes, opt.AddReactorFun)
	}
	return clientset.New(clientset.WithClient(dynamicClient)), nil
}
