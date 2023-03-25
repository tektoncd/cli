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

package log

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/pods"
	taskrunpkg "github.com/tektoncd/cli/pkg/taskrun"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	MsgTRNotFoundErr = "Unable to get TaskRun"
)

type step struct {
	name      string
	container string
	state     corev1.ContainerState
}

func (s *step) hasStarted() bool {
	return s.state.Waiting == nil
}

func (r *Reader) readTaskLog() (<-chan Log, <-chan error, error) {
	tr, err := taskrunpkg.GetTaskRun(taskrunGroupResource, r.clients, r.run, r.ns)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %s", MsgTRNotFoundErr, err)
	}

	r.formTaskName(tr)

	if !isDone(tr, r.retries) && r.follow {
		return r.readLiveTaskLogs(tr)
	}
	return r.readAvailableTaskLogs(tr)
}

func (r *Reader) formTaskName(tr *v1.TaskRun) {
	if r.task != "" {
		return
	}

	if name, ok := tr.Labels["tekton.dev/pipelineTask"]; ok {
		r.task = name
		return
	}

	if tr.Spec.TaskRef != nil {
		r.task = tr.Spec.TaskRef.Name
		return
	}

	r.task = fmt.Sprintf("Task %d", r.number)
}

func (r *Reader) readLiveTaskLogs(tr *v1.TaskRun) (<-chan Log, <-chan error, error) {
	podC, podErrC, err := r.getTaskRunPodNames(tr)
	if err != nil {
		return nil, nil, err
	}
	logC, errC := r.readPodLogs(podC, podErrC, r.follow, r.timestamps)
	return logC, errC, nil
}

func (r *Reader) readAvailableTaskLogs(tr *v1.TaskRun) (<-chan Log, <-chan error, error) {
	if !tr.HasStarted() {
		return nil, nil, fmt.Errorf("task %s has not started yet", r.task)
	}

	// Check if taskrun failed on start up
	if err := hasTaskRunFailed(tr, r.task, r.retries); err != nil {
		if r.stream != nil {
			fmt.Fprintf(r.stream.Err, "%s\n", err.Error())
		} else {
			return nil, nil, err
		}
	}

	if tr.Status.PodName == "" {
		return nil, nil, fmt.Errorf("pod for taskrun %s not available yet", tr.Name)
	}

	podC := make(chan string)
	go func() {
		defer close(podC)
		if tr.Status.PodName != "" {
			if len(tr.Status.RetriesStatus) != 0 {
				for _, retryStatus := range tr.Status.RetriesStatus {
					podC <- retryStatus.PodName
				}
			}
			podC <- tr.Status.PodName
		}
	}()

	logC, errC := r.readPodLogs(podC, nil, false, r.timestamps)
	return logC, errC, nil
}

func (r *Reader) readStepsLogs(logC chan<- Log, errC chan<- error, steps []*step, pod *pods.Pod, follow, timestamps bool) {
	for _, step := range steps {
		if !follow && !step.hasStarted() {
			continue
		}

		container := pod.Container(step.container)
		containerLogC, containerLogErrC, err := container.LogReader(follow, timestamps).Read()
		if err != nil {
			errC <- fmt.Errorf("error in getting logs for step %s: %s", step.name, err)
			continue
		}

		for containerLogC != nil || containerLogErrC != nil {
			select {
			case l, ok := <-containerLogC:
				if !ok {
					containerLogC = nil
					logC <- Log{Task: r.task, Step: step.name, Log: "EOFLOG"}
					continue
				}
				logC <- Log{Task: r.task, Step: step.name, Log: l.Log}

			case e, ok := <-containerLogErrC:
				if !ok {
					containerLogErrC = nil
					continue
				}

				errC <- fmt.Errorf("failed to get logs for %s: %s", step.name, e)
			}
		}

		if err := container.Status(); err != nil {
			errC <- err
			return
		}
	}
}

func (r *Reader) readPodLogs(podC <-chan string, podErrC <-chan error, follow, timestamps bool) (<-chan Log, <-chan error) {
	logC := make(chan Log)
	errC := make(chan error)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		// forward pod error to error stream
		if podErrC != nil {
			for podErr := range podErrC {
				errC <- podErr
			}
		}
		wg.Done()

		// wait for all goroutines to close before closing errC channel
		wg.Wait()
		close(errC)
	}()

	wg.Add(1)
	go func() {
		defer func() {
			close(logC)
			wg.Done()
		}()

		for podName := range podC {
			p := pods.New(podName, r.ns, r.clients.Kube, r.streamer)
			var pod *corev1.Pod
			var err error

			if follow {
				pod, err = p.Wait()
			} else {
				pod, err = p.Get()
			}
			if err != nil {
				errC <- fmt.Errorf("task %s failed: %s. Run tkn tr desc %s for more details", r.task, strings.TrimSpace(err.Error()), r.run)
			}
			steps := filterSteps(pod, r.allSteps, r.steps)
			r.readStepsLogs(logC, errC, steps, p, follow, timestamps)
		}
	}()

	return logC, errC
}

// Reading of logs should wait until the name of the pod is
// updated in the status. Open a watch channel on the task run
// and keep checking the status until the taskrun completes
// or the timeout is reached.
func (r *Reader) getTaskRunPodNames(run *v1.TaskRun) (<-chan string, <-chan error, error) {
	opts := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", r.run).String(),
	}

	watchRun, err := actions.Watch(taskrunGroupResource, r.clients, r.ns, opts)
	if err != nil {
		return nil, nil, err
	}

	podC := make(chan string)
	errC := make(chan error)

	go func() {
		defer func() {
			close(podC)
			close(errC)
			watchRun.Stop()
		}()

		podMap := make(map[string]bool)
		addPod := func(name string) {
			if _, ok := podMap[name]; !ok {
				podMap[name] = true
				podC <- name
			}
		}

		if len(run.Status.RetriesStatus) != 0 {
			for _, retryStatus := range run.Status.RetriesStatus {
				addPod(retryStatus.PodName)
			}
		}
		if run.Status.PodName != "" {
			addPod(run.Status.PodName)
		}

		timeout := time.After(r.activityTimeout)

		for {
			select {
			case event := <-watchRun.ResultChan():
				var err error
				run, err = cast2taskrun(event.Object)
				if err != nil {
					errC <- err
					return
				}
				if run.Status.PodName != "" {
					addPod(run.Status.PodName)
					if areRetriesScheduled(run, r.retries) {
						return
					}
				}
			case <-timeout:
				// Check if taskrun failed on start up
				if err := hasTaskRunFailed(run, r.task, r.retries); err != nil {
					errC <- err
					return
				}
				// check if pod has been started and has a name
				if run.HasStarted() && run.Status.PodName != "" {
					if !areRetriesScheduled(run, r.retries) {
						continue
					}
					return
				}
				errC <- fmt.Errorf("task %s create has not started yet or pod for task not yet available", r.task)
				return
			}
		}
	}()

	return podC, errC, nil
}

func filterSteps(pod *corev1.Pod, allSteps bool, stepsGiven []string) []*step {
	steps := []*step{}
	stepsInPod := getSteps(pod)

	if allSteps {
		steps = append(steps, getInitSteps(pod)...)
	}

	if len(stepsGiven) == 0 {
		steps = append(steps, stepsInPod...)
		return steps
	}

	stepsToAdd := map[string]bool{}
	for _, s := range stepsGiven {
		stepsToAdd[s] = true
	}

	for _, sp := range stepsInPod {
		if stepsToAdd[sp.name] {
			steps = append(steps, sp)
		}
	}

	return steps
}

func getInitSteps(pod *corev1.Pod) []*step {
	status := map[string]corev1.ContainerState{}
	for _, ics := range pod.Status.InitContainerStatuses {
		status[ics.Name] = ics.State
	}

	steps := []*step{}
	for _, ic := range pod.Spec.InitContainers {
		steps = append(steps, &step{
			name:      strings.TrimPrefix(ic.Name, "step-"),
			container: ic.Name,
			state:     status[ic.Name],
		})
	}

	return steps
}

func getSteps(pod *corev1.Pod) []*step {
	status := map[string]corev1.ContainerState{}
	for _, cs := range pod.Status.ContainerStatuses {
		status[cs.Name] = cs.State
	}

	steps := []*step{}
	for _, c := range pod.Spec.Containers {
		steps = append(steps, &step{
			name:      strings.TrimPrefix(c.Name, "step-"),
			container: c.Name,
			state:     status[c.Name],
		})
	}

	return steps
}

func hasTaskRunFailed(tr *v1.TaskRun, taskName string, retries int) error {
	if isFailure(tr, retries) {
		return fmt.Errorf("task %s has failed: %s", taskName, tr.Status.Conditions[0].Message)
	}
	return nil
}

func cast2taskrun(obj runtime.Object) (*v1.TaskRun, error) {
	var run *v1.TaskRun
	unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct, &run); err != nil {
		return nil, err
	}
	return run, nil
}

func isDone(tr *v1.TaskRun, retries int) bool {
	return tr.IsDone() || !areRetriesScheduled(tr, retries)
}

func isFailure(tr *v1.TaskRun, retries int) bool {
	conditions := tr.Status.Conditions
	return len(conditions) != 0 && conditions[0].Status == corev1.ConditionFalse && areRetriesScheduled(tr, retries)
}

func areRetriesScheduled(tr *v1.TaskRun, retries int) bool {
	retriesDone := len(tr.Status.RetriesStatus)
	return retriesDone >= retries
}
