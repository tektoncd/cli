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

package wait

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tektoncd/cli/test/framework"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"go.opencensus.io/trace"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/apis"
)

type TaskStateFn func(r *v1.Task) (bool, error)

// TaskRunStateFn is a condition function on TaskRun used polling functions
type TaskRunStateFn func(r *v1.TaskRun) (bool, error)

// PipelineRunStateFn is a condition function on TaskRun used polling functions
type PipelineRunStateFn func(pr *v1.PipelineRun) (bool, error)

type PodRunStateFn func(r *corev1.Pod) (bool, error)

// ForTaskRunState polls the status of the TaskRun called name from client every
// interval until inState returns `true` indicating it is done, returns an
// error or timeout. desc will be used to name the metric that is emitted to
// track how long it took for name to get into the state checked by inState.
func ForTaskRunState(c *framework.Clients, name string, inState TaskRunStateFn, desc string) error {
	metricName := fmt.Sprintf("WaitForTaskRunState/%s/%s", name, desc)
	_, span := trace.StartSpan(context.Background(), metricName)
	defer span.End()

	return wait.PollImmediate(framework.Interval, framework.Apitimeout, func() (bool, error) {
		r, err := c.TaskRunClient.Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return inState(r)
	})
}

// ForPodState polls the status of the Pod called name from client every
// interval until inState returns `true` indicating it is done, returns an
// error or timeout. desc will be used to name the metric that is emitted to
// track how long it took for name to get into the state checked by inState.
func ForPodState(c *framework.Clients, name string, namespace string, inState func(r *corev1.Pod) (bool, error), desc string) error {
	metricName := fmt.Sprintf("WaitForPodState/%s/%s", name, desc)
	_, span := trace.StartSpan(context.Background(), metricName)
	defer span.End()

	return wait.PollImmediate(framework.Interval, framework.Apitimeout, func() (bool, error) {
		r, err := c.KubeClient.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		return inState(r)
	})
}

func ForPodStateKube(c kubernetes.Interface, namespace string, inState PodRunStateFn, desc string) error {
	podlist, err := c.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, v := range podlist.Items {
		metricName := fmt.Sprintf("WaitForPodState/%s/%s", v.Name, desc)
		_, span := trace.StartSpan(context.Background(), metricName)
		defer span.End()

		err1 := wait.PollImmediate(framework.Interval, framework.Apitimeout, func() (bool, error) {
			fmt.Println("v.Name", v.Name)
			r, err := c.CoreV1().Pods(namespace).Get(context.Background(), v.Name, metav1.GetOptions{})
			if r.Status.Phase == "Running" || r.Status.Phase == "Succeeded" && err != nil {
				fmt.Printf("Pods are Running !! in namespace %s podName %s \n", namespace, v.Name)
				return true, err
			}
			return inState(r)
		})
		if err1 != nil {
			log.Panic(err1.Error())
		}
	}

	return err

}

func ForPodStatus(kubeClient kubernetes.Interface, namespace string) {
	watch, err := kubeClient.CoreV1().Pods(namespace).Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatal(err.Error())
	}
	respond := make(chan string)

	go PodStatus(respond, watch)
	defer close(respond)
	select {
	case queryResp := <-respond:
		if queryResp == "Done" {
			log.Println("Status of resources are up and running ")
		}
	case <-time.After(30 * time.Second):
		log.Panicln("Status of resources are not up and running ")
	}

}

func PodStatus(respond chan<- string, watch watch.Interface) {
	count := 0
	for event := range watch.ResultChan() {
		p, ok := event.Object.(*corev1.Pod)
		if !ok {
			log.Fatal("unexpected type")
		}
		fmt.Printf(" %s (status) -> %s \n", p.GenerateName, p.Status.Phase)
		if p.Status.Phase == "Running" || p.Status.Phase == "Succeeded" {
			count++
			if count == 2 {
				break
			}
		} else {
			continue
		}
	}
	respond <- "Done"

}

// ForPipelineRunState polls the status of the PipelineRun called name from client every
// interval until inState returns `true` indicating it is done, returns an
// error or timeout. desc will be used to name the metric that is emitted to
// track how long it took for name to get into the state checked by inState.
func ForPipelineRunState(c *framework.Clients, name string, polltimeout time.Duration, inState PipelineRunStateFn, desc string) error {
	metricName := fmt.Sprintf("WaitForPipelineRunState/%s/%s", name, desc)
	_, span := trace.StartSpan(context.Background(), metricName)
	defer span.End()

	return wait.PollImmediate(framework.Interval, polltimeout, func() (bool, error) {
		r, err := c.PipelineRunClient.Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return inState(r)
	})
}

// ForServiceExternalIPState polls the status of the a k8s Service called name from client every
// interval until an external ip is assigned indicating it is done, returns an
// error or timeout. desc will be used to name the metric that is emitted to
// track how long it took for name to get into the state checked by inState.
func ForServiceExternalIPState(c *framework.Clients, namespace, name string, inState func(s *corev1.Service) (bool, error), desc string) error {
	metricName := fmt.Sprintf("WaitForServiceExternalIPState/%s/%s", name, desc)
	_, span := trace.StartSpan(context.Background(), metricName)
	defer span.End()

	return wait.PollImmediate(framework.Interval, framework.Apitimeout, func() (bool, error) {
		r, err := c.KubeClient.CoreV1().Services(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return inState(r)
	})
}

// TaskRunSucceed provides a poll condition function that checks if the TaskRun
// has successfully completed.
func TaskRunSucceed(name string) TaskRunStateFn {
	return func(tr *v1.TaskRun) (bool, error) {
		c := tr.Status.GetCondition(apis.ConditionSucceeded)
		if c != nil {
			if c.Status == corev1.ConditionTrue {
				return true, nil
			} else if c.Status == corev1.ConditionFalse {
				return true, fmt.Errorf("Task run %s failed", name)
			}
		}
		return false, nil
	}
}

func PodRunSucceed(name string) PodRunStateFn {
	return func(r *corev1.Pod) (bool, error) {

		c := r.Status.Phase

		if c != "Pending" {
			if c == "Running" || c == "Succeeded" {
				fmt.Println("pods running !! ")
				return true, nil
			}
			fmt.Println("pods not running!!")
			return true, fmt.Errorf("Pod run in namespace  %s failed", name)
		}

		return false, nil
	}
}

// TaskRunFailed provides a poll condition function that checks if the TaskRun
// has failed.
func TaskRunFailed(name string) TaskRunStateFn {
	return func(tr *v1.TaskRun) (bool, error) {
		c := tr.Status.GetCondition(apis.ConditionSucceeded)
		if c != nil {
			if c.Status == corev1.ConditionTrue {
				return true, fmt.Errorf("Task run %s succeeded", name)
			} else if c.Status == corev1.ConditionFalse {
				return true, nil
			}
		}
		return false, nil
	}
}

// PipelineRunSucceed provides a poll condition function that checks if the PipelineRun
// has successfully completed.
func PipelineRunSucceed(name string) PipelineRunStateFn {
	return func(pr *v1.PipelineRun) (bool, error) {
		c := pr.Status.GetCondition(apis.ConditionSucceeded)
		if c != nil {
			if c.Status == corev1.ConditionTrue {
				return true, nil
			} else if c.Status == corev1.ConditionFalse {
				return true, fmt.Errorf("PipelineRun %s failed", name)
			}
		}
		return false, nil
	}
}

// PipelineRunFailed provides a poll condition function that checks if the PipelineRun
// has failed.
func PipelineRunFailed(name string) PipelineRunStateFn {
	return func(tr *v1.PipelineRun) (bool, error) {
		c := tr.Status.GetCondition(apis.ConditionSucceeded)
		if c != nil {
			if c.Status == corev1.ConditionTrue {
				return true, fmt.Errorf("Task run %s succeeded", name)
			} else if c.Status == corev1.ConditionFalse {
				return true, nil
			}
		}
		return false, nil
	}
}
