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

package framework

import (
	"log"

	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	v1 "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1beta1"
	"k8s.io/client-go/kubernetes"
	knativetest "knative.dev/pkg/test"
)

// clients holds instances of interfaces for making requests to the Pipeline controllers.
type Clients struct {
	KubeClient        kubernetes.Interface
	PipelineClient    v1.PipelineInterface
	TaskClient        v1.TaskInterface
	ClusterTaskClient v1beta1.ClusterTaskInterface
	TaskRunClient     v1.TaskRunInterface
	PipelineRunClient v1.PipelineRunInterface
}

// newClients instantiates and returns several clientsets required for making requests to the
// Pipeline cluster specified by the combination of clusterName and configPath. Clients can
// make requests within namespace.
func NewClients(configPath, clusterName, namespace string) *Clients {

	var err error
	c := &Clients{}

	cfg, err := knativetest.BuildClientConfig(configPath, clusterName)
	if err != nil {
		log.Fatalf("failed to create configuration obj from %s for cluster %s: %s", configPath, clusterName, err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create kubeclient from config file at %s: %s", configPath, err)
	}
	c.KubeClient = kubeClient

	cs, err := versioned.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create pipeline clientset from config file at %s: %s", configPath, err)
	}

	c.PipelineClient = cs.TektonV1().Pipelines(namespace)
	c.TaskClient = cs.TektonV1().Tasks(namespace)
	c.ClusterTaskClient = cs.TektonV1beta1().ClusterTasks()
	c.TaskRunClient = cs.TektonV1().TaskRuns(namespace)
	c.PipelineRunClient = cs.TektonV1().PipelineRuns(namespace)
	return c
}

type Test struct {
	Cmd      string
	Expected map[int]interface{}
}
