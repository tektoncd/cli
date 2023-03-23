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
	"sync"

	"github.com/tektoncd/cli/test/framework"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

// ForTaskRunToComplete Wait For Task Run Resource to be completed
func ForTaskRunToComplete(c *framework.Clients, trname string, namespace string) {
	log.Printf("Waiting for TaskRun %s in namespace %s to complete", trname, namespace)
	if err := ForTaskRunState(c, trname, func(tr *v1.TaskRun) (bool, error) {
		cond := tr.Status.GetCondition(apis.ConditionSucceeded)
		if cond != nil {
			if cond.Status == corev1.ConditionTrue || cond.Status == corev1.ConditionFalse {
				return true, nil
			} else if cond.Status != corev1.ConditionUnknown {
				return false, fmt.Errorf("taskRun %s failed ", trname)
			}
		}
		return false, nil
	}, "TaskRunSuccess"); err != nil {
		log.Fatalf("Error waiting for TaskRun %s to finish: %s", trname, err)
	}
}

// ForTaskRunToBeStarted Wait For Task Run Resource to be completed
func ForTaskRunToBeStarted(c *framework.Clients, trname string, namespace string) {
	log.Printf("Waiting for TaskRun %s in namespace %s to be started", trname, namespace)
	if err := ForTaskRunState(c, trname, func(tr *v1.TaskRun) (bool, error) {
		cond := tr.Status.GetCondition(apis.ConditionSucceeded)
		if cond != nil {
			if cond.Status == corev1.ConditionTrue || cond.Status == corev1.ConditionFalse {
				return true, fmt.Errorf("TaskRun %s already finished", trname)
			} else if cond.Status == corev1.ConditionUnknown && cond.Reason == "Running" || cond.Reason != "Pending" {
				return true, nil
			}
		}
		return false, nil
	}, "TaskRunStartedSuccessfully"); err != nil {
		log.Fatalf("Error waiting for TaskRun %s to start: %s", trname, err)
	}

}

// ForPipelineRunToStart Waits for PipelineRun to be started
func ForPipelineRunToStart(c *framework.Clients, prname string, namespace string) {

	log.Printf("Waiting for Pipelinerun %s in namespace %s to be started", prname, namespace)
	if err := ForPipelineRunState(c, prname, framework.Apitimeout, func(pr *v1.PipelineRun) (bool, error) {
		c := pr.Status.GetCondition(apis.ConditionSucceeded)
		if c != nil {
			if c.Status == corev1.ConditionTrue || c.Status == corev1.ConditionFalse {
				return true, fmt.Errorf("PipelineRun %s already finished", prname)
			} else if c.Status == corev1.ConditionUnknown && (c.Reason == "Running" || c.Reason != "Pending") {
				return true, nil
			}
		}
		return false, nil
	}, "PipelineRunRunning"); err != nil {
		log.Fatalf("Error waiting for PipelineRun %s to be running: %s", prname, err)
	}
}

// WaitForPipelineRunToComplete Wait for Pipeline Run to complete
func ForPipelineRunToComplete(c *framework.Clients, prname string, namespace string) {
	log.Printf("Waiting for Pipelinerun %s in namespace %s to be started", prname, namespace)
	if err := ForPipelineRunState(c, prname, framework.Apitimeout, func(pr *v1.PipelineRun) (bool, error) {
		c := pr.Status.GetCondition(apis.ConditionSucceeded)
		if c != nil {
			if c.Status == corev1.ConditionTrue || c.Status == corev1.ConditionFalse {
				return true, fmt.Errorf("PipelineRun %s already finished", prname)
			} else if c.Status == corev1.ConditionUnknown && (c.Reason == "Running" || c.Reason == "Pending") {
				return true, nil
			}
		}
		return false, nil
	}, "PipelineRunRunning"); err != nil {
		log.Fatalf("Error waiting for PipelineRun %s to be running: %s", prname, err)
	}

	taskrunList, err := c.TaskRunClient.List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("tekton.dev/pipelineRun=%s", prname)})

	if err != nil {
		log.Fatalf("Error listing TaskRuns for PipelineRun %s: %s", prname, err)
	}

	log.Printf("Waiting for TaskRuns from PipelineRun %s in namespace %s to be running", prname, namespace)
	errChan := make(chan error, len(taskrunList.Items))
	defer close(errChan)

	for _, taskrunItem := range taskrunList.Items {
		go func(name string) {
			err := ForTaskRunState(c, name, func(tr *v1.TaskRun) (bool, error) {
				c := tr.Status.GetCondition(apis.ConditionSucceeded)
				if c != nil {
					if c.Status == corev1.ConditionTrue || c.Status == corev1.ConditionFalse {
						return true, fmt.Errorf("TaskRun %s already finished", name)
					} else if c.Status == corev1.ConditionUnknown && (c.Reason == "Running" || c.Reason == "Pending") {
						return true, nil
					}
				}
				return false, nil
			}, "TaskRunRunning")
			errChan <- err
		}(taskrunItem.Name)
	}

	for i := 1; i <= len(taskrunList.Items); i++ {
		if <-errChan != nil {
			log.Panicf("Error waiting for TaskRun %s to be running: %s", taskrunList.Items[i-1].Name, err)
		}
	}

	if _, err := c.PipelineRunClient.Get(context.Background(), prname, metav1.GetOptions{}); err != nil {
		log.Panicf("Failed to get PipelineRun `%s`: %s", prname, err)
	}

	log.Printf("Waiting for PipelineRun %s in namespace %s to be Completed", prname, namespace)
	if err := ForPipelineRunState(c, prname, framework.Apitimeout, func(pr *v1.PipelineRun) (bool, error) {
		c := pr.Status.GetCondition(apis.ConditionSucceeded)
		if c != nil {
			if c.Status == corev1.ConditionTrue || c.Status == corev1.ConditionFalse {
				return true, nil
			} else if c.Status != corev1.ConditionUnknown {
				return false, fmt.Errorf("pipeline Run %s failed ", prname)
			}
		}
		return false, nil
	}, "PipelineRunSuccess"); err != nil {
		log.Panicf("Error waiting for PipelineRun %s to finish: %s", prname, err)
	}

	log.Printf("Waiting for TaskRuns from PipelineRun %s in namespace %s to be completed", prname, namespace)
	var wg sync.WaitGroup
	for _, taskrunItem := range taskrunList.Items {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			err := ForTaskRunState(c, name, func(tr *v1.TaskRun) (bool, error) {
				cond := tr.Status.GetCondition(apis.ConditionSucceeded)
				if cond != nil {
					if cond.Status == corev1.ConditionTrue || cond.Status == corev1.ConditionFalse {
						return true, nil
					} else if cond.Status != corev1.ConditionUnknown {
						return false, fmt.Errorf("Task Run %s failed ", name)
					}
				}
				return false, nil
			}, "TaskRunSuccess")
			if err != nil {
				log.Fatalf("Error waiting for TaskRun %s to err: %s", name, err)
			}
		}(taskrunItem.Name)
	}
	wg.Wait()

	if _, err := c.PipelineRunClient.Get(context.Background(), prname, metav1.GetOptions{}); err != nil {
		log.Panicf("Failed to get PipelineRun `%s`: %s", prname, err)
	}
}
