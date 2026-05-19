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

package kube

import (
	"fmt"

	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Config struct {
	Path      string
	Context   string
	Namespace string
}

type clients struct {
	tekton         versioned.Interface
	dynamic        dynamic.Interface
	kubeConfigPath string
	kubeContext    string
	namespace      string
}

type ClientSet interface {
	Dynamic() dynamic.Interface
	Tekton() versioned.Interface
	Namespace() string
}

var _ ClientSet = (*clients)(nil)

func (p *clients) Dynamic() dynamic.Interface {
	return p.dynamic
}

func (p *clients) Tekton() versioned.Interface {
	return p.tekton
}
func (p *clients) Namespace() string {
	return p.namespace
}

func NewClientSet(p Config) (*clients, error) {

	client := &clients{
		kubeConfigPath: p.Path,
		kubeContext:    p.Context,
		namespace:      p.Namespace,
	}

	config, err := client.config()
	if err != nil {
		return nil, err
	}

	client.tekton, err = tektonClient(config)
	if err != nil {
		return nil, err
	}

	client.dynamic, err = dynamicClient(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (p *clients) config() (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if p.kubeConfigPath != "" {
		loadingRules.ExplicitPath = p.kubeConfigPath
	}
	configOverrides := &clientcmd.ConfigOverrides{}
	if p.kubeContext != "" {
		configOverrides.CurrentContext = p.kubeContext
	}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	if p.namespace == "" {
		namespace, _, err := kubeConfig.Namespace()
		if err != nil {
			return nil, fmt.Errorf("couldn't get kubeConfiguration namespace: %v", err)
		}
		p.namespace = namespace
	}
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed parsing kubeconfig: %v", err)
	}
	return config, nil
}

func dynamicClient(config *rest.Config) (dynamic.Interface, error) {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client from config: %v", err)
	}
	return dynamicClient, err
}

func tektonClient(config *rest.Config) (versioned.Interface, error) {
	cs, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return cs, nil
}
