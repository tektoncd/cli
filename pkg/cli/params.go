// Copyright © 2019 The Tekton Authors.
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

package cli

import (
	"github.com/fatih/color"
	"github.com/jonboulle/clockwork"
	"github.com/pkg/errors"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	versionedResource "github.com/tektoncd/pipeline/pkg/client/resource/clientset/versioned"
	versionedTriggers "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"k8s.io/client-go/dynamic"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type TektonParams struct {
	clients        *Clients
	kubeConfigPath string
	kubeContext    string
	namespace      string
}

// ensure that TektonParams complies with cli.Params interface
var _ Params = (*TektonParams)(nil)

func (p *TektonParams) SetKubeConfigPath(path string) {
	p.kubeConfigPath = path
}

func (p *TektonParams) SetKubeContext(context string) {
	p.kubeContext = context
}

func (p *TektonParams) tektonClient(config *rest.Config) (versioned.Interface, error) {
	cs, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func (p *TektonParams) triggersClient(config *rest.Config) (versionedTriggers.Interface, error) {
	cs, err := versionedTriggers.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func (p *TektonParams) resourceClient(config *rest.Config) (versionedResource.Interface, error) {
	cs, err := versionedResource.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

// Set kube client based on config
func (p *TektonParams) kubeClient(config *rest.Config) (k8s.Interface, error) {
	k8scs, err := k8s.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create k8s client from config")
	}

	return k8scs, nil
}

func (p *TektonParams) dynamicClient(config *rest.Config) (dynamic.Interface, error) {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create dynamic client from config")

	}
	return dynamicClient, err
}

// Only returns kube client, not tekton client
func (p *TektonParams) KubeClient() (k8s.Interface, error) {
	config, err := p.config()
	if err != nil {
		return nil, err
	}

	kube, err := p.kubeClient(config)
	if err != nil {
		return nil, err
	}

	return kube, nil
}

func (p *TektonParams) Clients(cfg ...*rest.Config) (*Clients, error) {
	if p.clients != nil {
		return p.clients, nil
	}
	var config *rest.Config

	if len(cfg) != 0 && cfg[0] != nil {
		config = cfg[0]
	} else {
		defaultConfig, err := p.config()
		if err != nil {
			return nil, err
		}
		config = defaultConfig
	}

	tekton, err := p.tektonClient(config)
	if err != nil {
		return nil, err
	}

	resource, err := p.resourceClient(config)
	if err != nil {
		return nil, err
	}

	triggers, err := p.triggersClient(config)
	if err != nil {
		return nil, err
	}

	kube, err := p.kubeClient(config)
	if err != nil {
		return nil, err
	}

	dynamic, err := p.dynamicClient(config)
	if err != nil {
		return nil, err
	}

	p.clients = &Clients{
		Tekton:   tekton,
		Kube:     kube,
		Resource: resource,
		Triggers: triggers,
		Dynamic:  dynamic,
	}

	return p.clients, nil
}

func (p *TektonParams) config() (*rest.Config, error) {
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
			return nil, errors.Wrap(err, "Couldn't get kubeConfiguration namespace")
		}
		p.namespace = namespace
	}
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Parsing kubeconfig failed")
	}

	// set values as done in kubectl
	config.QPS = 50.0
	config.Burst = 300

	return config, nil
}

func (p *TektonParams) SetNoColour(b bool) {
	color.NoColor = b
}

func (p *TektonParams) SetNamespace(ns string) {
	p.namespace = ns
}

func (p *TektonParams) Namespace() string {
	return p.namespace
}

func (p *TektonParams) Time() clockwork.Clock {
	return clockwork.NewRealClock()
}
