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
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/names"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/system"
	knativetest "knative.dev/pkg/test"
	"knative.dev/pkg/test/logging"
	"sigs.k8s.io/yaml"

	// Mysteriously by k8s libs, or they fail to create `KubeClient`s from config. Apparently just importing it is enough. @_@ side effects @_@. https://github.com/kubernetes/client-go/issues/242
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// Mysteriously by k8s libs, or they fail to create `KubeClient`s when using oidc authentication. Apparently just importing it is enough. @_@ side effects @_@. https://github.com/kubernetes/client-go/issues/345
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

var initMetrics sync.Once

const (
	Interval   = 1 * time.Second
	Apitimeout = 10 * time.Minute
)

func Setup(t *testing.T) (*Clients, string) {
	t.Helper()
	namespace := names.SimpleNameGenerator.RestrictLengthWithRandomSuffix("testcli")

	initializeLogsAndMetrics(t)
	c := NewClients(knativetest.Flags.Kubeconfig, knativetest.Flags.Cluster, namespace)
	CreateNamespace(namespace, c.KubeClient)
	VerifyServiceAccountExistence(namespace, c.KubeClient)
	return c, namespace
}

func Header(logf logging.FormatLogger, text string) {
	left := "### "
	right := " ###"
	txt := left + text + right
	bar := strings.Repeat("#", len(txt))
	logf(bar)
	logf(txt)
	logf(bar)
}

func TearDown(t *testing.T, cs *Clients, namespace string) {
	t.Helper()
	if cs.KubeClient == nil {
		return
	}
	if t.Failed() {
		Header(t.Logf, fmt.Sprintf("Dumping objects from %s", namespace))
		bs, err := getCRDYaml(cs, namespace)
		if err != nil {
			t.Error(err)
		} else {
			t.Log(string(bs))
		}
		Header(t.Logf, fmt.Sprintf("Dumping logs from Pods in the %s", namespace))
		taskruns, err := cs.TaskRunClient.List(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Errorf("Error getting TaskRun list %s", err)
		}
		for _, tr := range taskruns.Items {
			if tr.Status.PodName != "" {
				CollectPodLogs(cs, tr.Status.PodName, namespace, t.Logf)
			}
		}
	}

	if os.Getenv("TEST_KEEP_NAMESPACES") == "" && !t.Failed() {
		t.Logf("Deleting namespace %s", namespace)
		if err := cs.KubeClient.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{}); err != nil {
			t.Errorf("Failed to delete namespace %s: %s", namespace, err)
		}
	}
}

func initializeLogsAndMetrics(t *testing.T) {
	initMetrics.Do(func() {
		flag.Parse()
		flag.Set("alsologtostderr", "true")
		logging.InitializeLogger()

		logging.InitializeMetricExporter(t.Name())
	})
}

func CreateNamespace(namespace string, kubeClient kubernetes.Interface) {
	log.Printf("Create namespace %s to deploy to", namespace)
	if _, err := kubeClient.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{}); err != nil {
		log.Printf("Failed to create namespace %s for tests: %s", namespace, err)
	}
}

func DeleteNamespace(namespace string, cs *Clients) {
	log.Printf("Deleting namespace %s", namespace)
	if err := cs.KubeClient.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{}); err != nil {
		log.Printf("Failed to delete namespace %s: %s", namespace, err)
	}
}

func getDefaultSA(kubeClient kubernetes.Interface) string {
	configDefaultsCM, err := kubeClient.CoreV1().ConfigMaps(system.Namespace()).Get(context.Background(), config.GetDefaultsConfigName(), metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Failed to get ConfigMap `%s`: %s", config.GetDefaultsConfigName(), err)
	}
	actual, ok := configDefaultsCM.Data["default-service-account"]
	if !ok {
		return "default"
	}
	return actual
}

func VerifyServiceAccountExistence(namespace string, kubeClient kubernetes.Interface) {
	defaultSA := getDefaultSA(kubeClient)
	log.Printf("Verify SA %q is created in namespace %q", defaultSA, namespace)

	if err := wait.PollImmediate(Interval, Apitimeout, func() (bool, error) {
		_, err := kubeClient.CoreV1().ServiceAccounts(namespace).Get(context.Background(), defaultSA, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			return false, nil
		}
		return true, err
	}); err != nil {
		log.Printf("Failed to get SA %q in namespace %q for tests: %s", defaultSA, namespace, err)
	}
}

func VerifyServiceAccountExistenceForSecrets(namespace string, kubeClient kubernetes.Interface, sa string) {
	defaultSA := sa
	log.Printf("Verify SA %q is created in namespace %q", defaultSA, namespace)

	if err := wait.PollImmediate(Interval, Apitimeout, func() (bool, error) {
		_, err := kubeClient.CoreV1().ServiceAccounts(namespace).Get(context.Background(), defaultSA, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			return false, nil
		}
		return true, err
	}); err != nil {
		log.Printf("Failed to get SA %q in namespace %q for tests: %s", defaultSA, namespace, err)
	}
}

func getCRDYaml(cs *Clients, ns string) ([]byte, error) {
	var output []byte
	printOrAdd := func(_, _ string, i interface{}) {
		bs, err := yaml.Marshal(i)
		if err != nil {
			return
		}
		output = append(output, []byte("\n---\n")...)
		output = append(output, bs...)
	}

	ps, err := cs.PipelineClient.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get pipeline: %w", err)
	}
	for _, i := range ps.Items {
		printOrAdd("Pipeline", i.Name, i)
	}

	prrs, err := cs.PipelineRunClient.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get pipelinerun: %w", err)
	}
	for _, i := range prrs.Items {
		printOrAdd("PipelineRun", i.Name, i)
	}

	ts, err := cs.TaskClient.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get tasks: %w", err)
	}
	for _, i := range ts.Items {
		printOrAdd("Task", i.Name, i)
	}
	trs, err := cs.TaskRunClient.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get taskrun: %w", err)
	}
	for _, i := range trs.Items {
		printOrAdd("TaskRun", i.Name, i)
	}

	pods, err := cs.KubeClient.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get pods: %w", err)
	}
	for _, i := range pods.Items {
		printOrAdd("Pod", i.Name, i)
	}

	return output, nil
}
