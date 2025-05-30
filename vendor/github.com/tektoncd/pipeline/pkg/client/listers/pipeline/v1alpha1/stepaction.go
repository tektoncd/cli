/*
Copyright 2020 The Tekton Authors

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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// StepActionLister helps list StepActions.
// All objects returned here must be treated as read-only.
type StepActionLister interface {
	// List lists all StepActions in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*pipelinev1alpha1.StepAction, err error)
	// StepActions returns an object that can list and get StepActions.
	StepActions(namespace string) StepActionNamespaceLister
	StepActionListerExpansion
}

// stepActionLister implements the StepActionLister interface.
type stepActionLister struct {
	listers.ResourceIndexer[*pipelinev1alpha1.StepAction]
}

// NewStepActionLister returns a new StepActionLister.
func NewStepActionLister(indexer cache.Indexer) StepActionLister {
	return &stepActionLister{listers.New[*pipelinev1alpha1.StepAction](indexer, pipelinev1alpha1.Resource("stepaction"))}
}

// StepActions returns an object that can list and get StepActions.
func (s *stepActionLister) StepActions(namespace string) StepActionNamespaceLister {
	return stepActionNamespaceLister{listers.NewNamespaced[*pipelinev1alpha1.StepAction](s.ResourceIndexer, namespace)}
}

// StepActionNamespaceLister helps list and get StepActions.
// All objects returned here must be treated as read-only.
type StepActionNamespaceLister interface {
	// List lists all StepActions in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*pipelinev1alpha1.StepAction, err error)
	// Get retrieves the StepAction from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*pipelinev1alpha1.StepAction, error)
	StepActionNamespaceListerExpansion
}

// stepActionNamespaceLister implements the StepActionNamespaceLister
// interface.
type stepActionNamespaceLister struct {
	listers.ResourceIndexer[*pipelinev1alpha1.StepAction]
}
