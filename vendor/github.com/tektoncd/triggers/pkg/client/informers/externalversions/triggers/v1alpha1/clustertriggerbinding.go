/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	time "time"

	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	versioned "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	internalinterfaces "github.com/tektoncd/triggers/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/tektoncd/triggers/pkg/client/listers/triggers/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ClusterTriggerBindingInformer provides access to a shared informer and lister for
// ClusterTriggerBindings.
type ClusterTriggerBindingInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ClusterTriggerBindingLister
}

type clusterTriggerBindingInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewClusterTriggerBindingInformer constructs a new informer for ClusterTriggerBinding type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewClusterTriggerBindingInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredClusterTriggerBindingInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredClusterTriggerBindingInformer constructs a new informer for ClusterTriggerBinding type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredClusterTriggerBindingInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.TektonV1alpha1().ClusterTriggerBindings().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.TektonV1alpha1().ClusterTriggerBindings().Watch(options)
			},
		},
		&triggersv1alpha1.ClusterTriggerBinding{},
		resyncPeriod,
		indexers,
	)
}

func (f *clusterTriggerBindingInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredClusterTriggerBindingInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *clusterTriggerBindingInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&triggersv1alpha1.ClusterTriggerBinding{}, f.defaultInformer)
}

func (f *clusterTriggerBindingInformer) Lister() v1alpha1.ClusterTriggerBindingLister {
	return v1alpha1.NewClusterTriggerBindingLister(f.Informer().GetIndexer())
}
