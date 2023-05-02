//go:build e2e
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

package eventlistener

import (
	"context"
	"os"
	"testing"

	"github.com/tektoncd/cli/test/cli"
	"github.com/tektoncd/cli/test/framework"
	"github.com/tektoncd/cli/test/helper"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativetest "knative.dev/pkg/test"
)

func TestEventListenerE2E(t *testing.T) {
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer cleanupResources(t, c, namespace)

	kubectl := cli.NewKubectl(namespace)
	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)
	elName := "github-listener-interceptor"

	t.Logf("Creating EventListener %s in namespace %s", elName, namespace)
	createResources(t, c, namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("eventlistener/eventlistener.yaml"))
	// Wait for pods to become available for next test
	kubectl.MustSucceed(t, "wait", "--for=condition=Ready", "pod", "-n", namespace, "--timeout=2m", "--all")

	t.Run("Assert if EventListener AVAILABLE status is true", func(t *testing.T) {
		res := tkn.MustSucceed(t, "eventlistener", "list")
		stdout := res.Stdout()
		assert.Check(t, helper.ContainsAll(stdout, elName, "AVAILABLE", "True"))
	})

	t.Logf("Scaling EventListener %s to 3 replicas in namespace %s", elName, namespace)
	kubectl.MustSucceed(t, "apply", "-f", helper.GetResourcePath("eventlistener/eventlistener-multi-replica.yaml"))
	// Wait for pods to become available for next test
	kubectl.MustSucceed(t, "wait", "--for=condition=Ready", "pod", "-n", namespace, "--timeout=2m", "--all")
}

func TestEventListenerLogsE2E(t *testing.T) {
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer cleanupResources(t, c, namespace)

	kubectl := cli.NewKubectl(namespace)
	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)
	elName := "github-listener-interceptor"

	t.Logf("Creating EventListener %s in namespace %s", elName, namespace)
	createResources(t, c, namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("eventlistener/eventlistener_log.yaml"))
	// Wait for pods to run and crash for next test
	kubectl.MustSucceed(t, "wait", "--for=jsonpath=.status.phase=Running", "pod", "-n", namespace, "--timeout=2m", "--all")

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

}

func TestEventListener_v1beta1LogsE2E(t *testing.T) {
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer cleanupResources(t, c, namespace)

	kubectl := cli.NewKubectl(namespace)
	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)
	elName := "github-listener-interceptor"

	t.Logf("Creating EventListener %s in namespace %s", elName, namespace)
	createResources(t, c, namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("eventlistener/eventlistener_v1beta1_log.yaml"))
	// Wait for pods to run and crash for next test
	kubectl.MustSucceed(t, "wait", "--for=jsonpath=.status.phase=Running", "pod", "-n", namespace, "--timeout=2m", "--all")

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

}

func TestEventListener_v1beta1E2E(t *testing.T) {
	c, namespace := framework.Setup(t)
	knativetest.CleanupOnInterrupt(func() { framework.TearDown(t, c, namespace) }, t.Logf)
	defer cleanupResources(t, c, namespace)

	kubectl := cli.NewKubectl(namespace)
	tkn, err := cli.NewTknRunner(namespace)
	assert.NilError(t, err)
	elName := "github-listener-interceptor"

	t.Logf("Creating EventListener %s in namespace %s", elName, namespace)
	createResources(t, c, namespace)
	kubectl.MustSucceed(t, "create", "-f", helper.GetResourcePath("eventlistener/eventlistener_v1beta1.yaml"))
	// Wait for pods to become available for next test
	kubectl.MustSucceed(t, "wait", "--for=condition=Ready", "pod", "-n", namespace, "--timeout=2m", "--all")

	t.Run("Assert if EventListener AVAILABLE status is true", func(t *testing.T) {
		res := tkn.MustSucceed(t, "eventlistener", "list")
		stdout := res.Stdout()
		assert.Check(t, helper.ContainsAll(stdout, elName, "AVAILABLE", "True"))
	})

	t.Logf("Scaling EventListener %s to 3 replicas in namespace %s", elName, namespace)
	kubectl.MustSucceed(t, "apply", "-f", helper.GetResourcePath("eventlistener/eventlistener_v1beta1-multi-replica.yaml"))
	// Wait for pods to become available for next test
	kubectl.MustSucceed(t, "wait", "--for=condition=Ready", "pod", "-n", namespace, "--timeout=2m", "--all")
}

func createResources(t *testing.T, c *framework.Clients, namespace string) {
	t.Helper()

	// Create SA and secret
	_, err := c.KubeClient.CoreV1().Secrets(namespace).Create(context.Background(),
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "github-secret"},
			Type:       corev1.SecretTypeOpaque,
			StringData: map[string]string{"secretToken": "1234567"},
		}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating secret: %s", err)
	}

	_, err = c.KubeClient.CoreV1().ServiceAccounts(namespace).Create(context.Background(),
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "tekton-triggers-github-sa"},
			Secrets: []corev1.ObjectReference{{
				Namespace: namespace,
				Name:      "github-secret",
			}},
		}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating SA: %s", err)
	}

	// Create ClusterRole required by triggers
	_, err = c.KubeClient.RbacV1().ClusterRoles().Create(context.Background(),
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "sa-clusterrole"},
			Rules: []rbacv1.PolicyRule{{
				APIGroups: []string{"triggers.tekton.dev"},
				Resources: []string{"clustertriggerbindings", "clusterinterceptors"},
				Verbs:     []string{"get", "list", "watch"},
			}},
		}, metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating ClusterRole: %s", err)
	}
	_, err = c.KubeClient.RbacV1().ClusterRoleBindings().Create(context.Background(),
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "sa-clusterrolebinding"},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      "tekton-triggers-github-sa",
				Namespace: namespace,
			}},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "sa-clusterrole",
			},
		}, metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Error creating ClusterRoleBinding: %s", err)
	}
}

func cleanupResources(t *testing.T, c *framework.Clients, namespace string) {
	t.Helper()
	framework.TearDown(t, c, namespace)

	if os.Getenv("TEST_KEEP_NAMESPACES") == "" && !t.Failed() {
		// Cleanup cluster-scoped resources
		t.Logf("Deleting cluster-scoped resources")
		if err := c.KubeClient.RbacV1().ClusterRoles().Delete(context.Background(), "sa-clusterrole", metav1.DeleteOptions{}); err != nil {
			t.Errorf("Failed to delete clusterrole sa-clusterrole: %s", err)
		}
		if err := c.KubeClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "sa-clusterrolebinding", metav1.DeleteOptions{}); err != nil {
			t.Errorf("Failed to delete clusterrolebinding sa-clusterrolebinding: %s", err)
		}
	}
}
