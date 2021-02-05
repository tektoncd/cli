// +build e2e
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

package pipeline

import (
	"context"
	"testing"

	"github.com/tektoncd/cli/test/cli"
	"github.com/tektoncd/cli/test/framework"
	"github.com/tektoncd/cli/test/helper"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativetest "knative.dev/pkg/test"
)

func TestEventListenerE2E(t *testing.T) {
	t.Parallel()
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer framework.TearDown(t, c, namespace)

	kubectl := cli.NewKubectl(namespace)
	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)
	elName := "github-listener-interceptor"

	t.Logf("Creating EventListener %s in namespace %s", elName, namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("eventlistener/eventlistener.yaml"))
	// Wait for pods to become available for next test
	kubectl.MustSucceed(t, "wait", "--for=condition=Ready", "pod", "-n", namespace, "--timeout=2m", "--all")

	t.Run("Assert if EventListener AVAILABLE status is true", func(t *testing.T) {
		res := tkn.MustSucceed(t, "eventlistener", "list")
		stdout := res.Stdout()
		assert.Check(t, helper.ContainsAll(stdout, elName, "AVAILABLE", "True"))
	})

	t.Run("Get logs of EventListener", func(t *testing.T) {
		res := tkn.MustSucceed(t, "eventlistener", "logs", elName, "-t", "1")

		elPods, err := c.KubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "eventlistener=" + elName})
		if err != nil {
			t.Fatalf("Error getting pods for EventListener %s: %v", elName, err)
		}

		podNum := len(elPods.Items)
		if podNum != 1 {
			t.Fatalf("Should be one replica for EventListener but had %d replicas", podNum)
		}

		stdout := res.Stdout()
		assert.Check(t, helper.ContainsAll(stdout, "github-listener-interceptor-el-github-listener-interceptor-", elPods.Items[0].Name))
	})

	t.Logf("Scaling EventListener %s to 3 replicas in namespace %s", elName, namespace)
	kubectl.MustSucceed(t, "apply", "-f", helper.GetResourcePath("eventlistener/eventlistener-multi-replica.yaml"))
	// Wait for pods to become available for next test
	kubectl.MustSucceed(t, "wait", "--for=condition=Ready", "pod", "-n", namespace, "--timeout=2m", "--all")

	t.Run("Get logs of EventListener with multiple pods", func(t *testing.T) {
		elPods, err := c.KubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "eventlistener=" + elName})
		if err != nil {
			t.Fatalf("Error getting pods for EventListener %s: %v", elName, err)
		}

		podNum := len(elPods.Items)
		if podNum != 3 {
			t.Fatalf("Should be three replicas for EventListener but had %d replicas", podNum)
		}

		res := tkn.MustSucceed(t, "eventlistener", "logs", elName, "-t", "1")
		stdout := res.Stdout()

		assert.Check(t, helper.ContainsAll(stdout, "github-listener-interceptor-el-github-listener-interceptor-", elPods.Items[0].Name, elPods.Items[1].Name, elPods.Items[2].Name))
	})
}
