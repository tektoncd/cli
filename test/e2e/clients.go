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

package e2e

import (
	"log"

	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1alpha1"
	resourceversioned "github.com/tektoncd/pipeline/pkg/client/resource/clientset/versioned"
	resourcev1alpha1 "github.com/tektoncd/pipeline/pkg/client/resource/clientset/versioned/typed/resource/v1alpha1"
	knativetest "knative.dev/pkg/test"
)

// clients holds instances of interfaces for making requests to the Pipeline controllers.
type Clients struct {
	KubeClient *knativetest.KubeClient

	PipelineClient         v1alpha1.PipelineInterface
	TaskClient             v1alpha1.TaskInterface
	TaskRunClient          v1alpha1.TaskRunInterface
	PipelineRunClient      v1alpha1.PipelineRunInterface
	PipelineResourceClient resourcev1alpha1.PipelineResourceInterface
	ConditionClient        v1alpha1.ConditionInterface
}

// newClients instantiates and returns several clientsets required for making requests to the
// Pipeline cluster specified by the combination of clusterName and configPath. Clients can
// make requests within namespace.
func NewClients(configPath, clusterName, namespace string) *Clients {

	var err error
	c := &Clients{}

	c.KubeClient, err = knativetest.NewKubeClient(configPath, clusterName)
	if err != nil {
		log.Fatalf("failed to create kubeclient from config file at %s: %s", configPath, err)
	}

	cfg, err := knativetest.BuildClientConfig(configPath, clusterName)
	if err != nil {
		log.Fatalf("failed to create configuration obj from %s for cluster %s: %s", configPath, clusterName, err)
	}

	cs, err := versioned.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create pipeline clientset from config file at %s: %s", configPath, err)
	}

	rcs, err := resourceversioned.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create pipeline clientset from config file at %s: %s", configPath, err)
	}
	c.PipelineClient = cs.TektonV1alpha1().Pipelines(namespace)
	c.TaskClient = cs.TektonV1alpha1().Tasks(namespace)
	c.TaskRunClient = cs.TektonV1alpha1().TaskRuns(namespace)
	c.PipelineRunClient = cs.TektonV1alpha1().PipelineRuns(namespace)
	c.PipelineResourceClient = rcs.TektonV1alpha1().PipelineResources(namespace)
	c.ConditionClient = cs.TektonV1alpha1().Conditions(namespace)
	return c
}

type Test struct {
	Cmd      string
	Expected map[int]interface{}
}

func NewTestData(cmd string, expected map[int]interface{}) *Test {
	t := &Test{}
	t.Cmd = cmd
	t.Expected = expected
	return t
}
