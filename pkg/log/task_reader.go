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
	"time"

	"github.com/pkg/errors"
	"github.com/tektoncd/cli/pkg/pods"
	tr "github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

const (
	MsgTRNotFoundErr = "Unable to get Taskrun"
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
	tr, err := tr.GetV1beta1(r.clients, r.run, metav1.GetOptions{}, r.ns)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %s", MsgTRNotFoundErr, err)
	}

	r.formTaskName(tr)

	if !tr.IsDone() && r.follow {
		return r.readLiveTaskLogs()
	}
	return r.readAvailableTaskLogs(tr)
}

func (r *Reader) formTaskName(tr *v1beta1.TaskRun) {
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

func (r *Reader) readLiveTaskLogs() (<-chan Log, <-chan error, error) {
	tr, err := r.waitUntilTaskPodNameAvailable(10)
	if err != nil {
		return nil, nil, err
	}

	var (
		podName = tr.Status.PodName
		kube    = r.clients.Kube
	)

	p := pods.New(podName, r.ns, kube, r.streamer)
	pod, err := p.Wait()
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("task %s failed: %s. Run tkn tr desc %s for more details.", r.task, strings.TrimSpace(err.Error()), tr.Name))
	}

	steps := filterSteps(pod, r.allSteps, r.steps)
	logC, errC := r.readStepsLogs(steps, p, r.follow)
	return logC, errC, err
}

func (r *Reader) readAvailableTaskLogs(tr *v1beta1.TaskRun) (<-chan Log, <-chan error, error) {
	if !tr.HasStarted() {
		return nil, nil, fmt.Errorf("task %s has not started yet", r.task)
	}

	// Check if taskrun failed on start up
	if err := hasTaskRunFailed(tr.Status.Conditions, r.task); err != nil {
		if r.stream != nil {
			fmt.Fprintf(r.stream.Err, "%s\n", err.Error())
		} else {
			return nil, nil, err
		}
	}

	if tr.Status.PodName == "" {
		return nil, nil, fmt.Errorf("pod for taskrun %s not available yet", tr.Name)
	}

	var (
		kube    = r.clients.Kube
		podName = tr.Status.PodName
	)

	p := pods.New(podName, r.ns, kube, r.streamer)
	pod, err := p.Get()
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("task %s failed: %s. Run tkn tr desc %s for more details.", r.task, strings.TrimSpace(err.Error()), tr.Name))
	}

	steps := filterSteps(pod, r.allSteps, r.steps)
	logC, errC := r.readStepsLogs(steps, p, r.follow)
	return logC, errC, nil
}

func (r *Reader) readStepsLogs(steps []*step, pod *pods.Pod, follow bool) (<-chan Log, <-chan error) {
	logC := make(chan Log)
	errC := make(chan error)

	go func() {
		defer close(logC)
		defer close(errC)

		for _, step := range steps {
			if !follow && !step.hasStarted() {
				continue
			}

			container := pod.Container(step.container)
			podC, perrC, err := container.LogReader(follow).Read()
			if err != nil {
				errC <- fmt.Errorf("error in getting logs for step %s: %s", step.name, err)
				continue
			}

			for podC != nil || perrC != nil {
				select {
				case l, ok := <-podC:
					if !ok {
						podC = nil
						logC <- Log{Task: r.task, Step: step.name, Log: "EOFLOG"}
						continue
					}
					logC <- Log{Task: r.task, Step: step.name, Log: l.Log}

				case e, ok := <-perrC:
					if !ok {
						perrC = nil
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
	}()

	return logC, errC
}

// Reading of logs should wait until the name of the pod is
// updated in the status. Open a watch channel on the task run
// and keep checking the status until the pod name updates
// or the timeout is reached.
func (r *Reader) waitUntilTaskPodNameAvailable(timeout time.Duration) (*v1beta1.TaskRun, error) {
	var first = true
	opts := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", r.run).String(),
	}
	run, err := tr.GetV1beta1(r.clients, r.run, metav1.GetOptions{}, r.ns)
	if err != nil {
		return nil, err
	}

	if run.Status.PodName != "" {
		return run, nil
	}

	watchRun, err := tr.Watch(r.clients, opts, r.ns)

	if err != nil {
		return nil, err
	}
	for {
		select {
		case event := <-watchRun.ResultChan():
			run, err := cast2taskrun(event.Object)
			if err != nil {
				return nil, err
			}
			if run.Status.PodName != "" {
				watchRun.Stop()
				return run, nil
			}
			if first {
				first = false
			}
		case <-time.After(timeout * time.Second):
			watchRun.Stop()

			// Check if taskrun failed on start up
			if err = hasTaskRunFailed(run.Status.Conditions, r.task); err != nil {
				return nil, err
			}

			return nil, fmt.Errorf("task %s create has not started yet or pod for task not yet available", r.task)
		}
	}
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

func hasTaskRunFailed(trConditions duckv1beta1.Conditions, taskName string) error {
	if len(trConditions) != 0 && trConditions[0].Status == corev1.ConditionFalse {
		return fmt.Errorf("task %s has failed: %s", taskName, trConditions[0].Message)
	}
	return nil
}

func cast2taskrun(obj runtime.Object) (*v1beta1.TaskRun, error) {
	var run *v1beta1.TaskRun
	unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct, &run); err != nil {
		return nil, err
	}
	return run, nil
}
