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

package taskrun

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/helper/pods"
	"github.com/tektoncd/cli/pkg/helper/pods/stream"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/reconciler/taskrun/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"knative.dev/pkg/apis/duck/v1beta1"
)

type step struct {
	name      string
	container string
	state     corev1.ContainerState
}

func (s *step) hasStarted() bool {
	return s.state.Waiting == nil
}

//Log data to write on log channel
type Log struct {
	Task string
	Step string
	Log  string
}

type LogReader struct {
	Task     string
	Run      string
	Number   int
	Ns       string
	Clients  *cli.Clients
	Streamer stream.NewStreamerFunc
	Follow   bool
	AllSteps bool
	Stream   *cli.Stream
}

func (lr *LogReader) Read() (<-chan Log, <-chan error, error) {
	tkn := lr.Clients.Tekton
	tr, err := tkn.TektonV1alpha1().TaskRuns(lr.Ns).Get(lr.Run, metav1.GetOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("%s : %s", msgTRNotFoundErr, err)
	}

	lr.formTaskName(tr)

	return lr.readLogs(tr)
}

func (lr *LogReader) readLogs(tr *v1alpha1.TaskRun) (<-chan Log, <-chan error, error) {
	if lr.Follow {
		return lr.readLiveLogs()
	}
	return lr.readAvailableLogs(tr)
}

func (lr *LogReader) formTaskName(tr *v1alpha1.TaskRun) {
	if lr.Task != "" {
		return
	}

	if name, ok := tr.Labels["tekton.dev/pipelineTask"]; ok {
		lr.Task = name
		return
	}

	if tr.Spec.TaskRef != nil {
		lr.Task = tr.Spec.TaskRef.Name
		return
	}

	lr.Task = fmt.Sprintf("Task %d", lr.Number)
}

func (lr *LogReader) readLiveLogs() (<-chan Log, <-chan error, error) {
	tr, err := lr.waitUntilPodNameAvailable(10)
	if err != nil {
		return nil, nil, err
	}

	var (
		podName = tr.Status.PodName
		kube    = lr.Clients.Kube
	)

	p := pods.New(podName, lr.Ns, kube, lr.Streamer)
	pod, err := p.Wait()
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("task %s failed: %s. Run tkn tr desc %s for more details.", lr.Task, strings.TrimSpace(err.Error()), tr.Name))
	}

	steps := filterSteps(pod, lr.AllSteps)
	logC, errC := lr.readStepsLogs(steps, p, lr.Follow)
	return logC, errC, err
}

func (lr *LogReader) readAvailableLogs(tr *v1alpha1.TaskRun) (<-chan Log, <-chan error, error) {
	if !tr.HasStarted() {
		return nil, nil, fmt.Errorf("task %s has not started yet", lr.Task)
	}

	//Check if taskrun failed on start up
	if err := hasTaskRunFailed(tr.Status.Conditions, lr.Task); err != nil {
		return nil, nil, err
	}

	if tr.Status.PodName == "" {
		return nil, nil, fmt.Errorf("pod for taskrun %s not available yet", tr.Name)
	}

	var (
		kube    = lr.Clients.Kube
		podName = tr.Status.PodName
	)

	p := pods.New(podName, lr.Ns, kube, lr.Streamer)
	pod, err := p.Get()
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("task %s failed: %s. Run tkn tr desc %s for more details.", lr.Task, strings.TrimSpace(err.Error()), tr.Name))
	}

	steps := filterSteps(pod, lr.AllSteps)
	logC, errC := lr.readStepsLogs(steps, p, lr.Follow)
	return logC, errC, nil
}

func (lr *LogReader) readStepsLogs(steps []*step, pod *pods.Pod, follow bool) (<-chan Log, <-chan error) {
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
						logC <- Log{Task: lr.Task, Step: step.name, Log: "EOFLOG"}
						continue
					}
					logC <- Log{Task: lr.Task, Step: step.name, Log: l.Log}

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

func filterSteps(pod *corev1.Pod, allSteps bool) []*step {
	steps := []*step{}

	if allSteps {
		steps = append(steps, getInitSteps(pod)...)
	}

	steps = append(steps, getSteps(pod)...)

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
			name:      resources.TrimContainerNamePrefix(ic.Name),
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
			name:      resources.TrimContainerNamePrefix(c.Name),
			container: c.Name,
			state:     status[c.Name],
		})
	}

	return steps
}

// Reading of logs should wait until the name of the pod is
// updated in the status. Open a watch channel on the task run
// and keep checking the status until the pod name updates
// or the timeout is reached.
func (lr *LogReader) waitUntilPodNameAvailable(timeout time.Duration) (*v1alpha1.TaskRun, error) {
	var first = true
	opts := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", lr.Run).String(),
	}
	tkn := lr.Clients.Tekton
	run, err := tkn.TektonV1alpha1().TaskRuns(lr.Ns).Get(lr.Run, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if run.Status.PodName != "" {
		return run, nil
	}

	watchRun, err := tkn.TektonV1alpha1().TaskRuns(lr.Ns).Watch(opts)
	if err != nil {
		return nil, err
	}
	for {
		select {
		case event := <-watchRun.ResultChan():
			run := event.Object.(*v1alpha1.TaskRun)
			if run.Status.PodName != "" {
				watchRun.Stop()
				return run, nil
			}
			if first {
				first = false
			}
		case <-time.After(timeout * time.Second):
			watchRun.Stop()

			//Check if taskrun failed on start up
			if err = hasTaskRunFailed(run.Status.Conditions, lr.Task); err != nil {
				return nil, err
			}

			return nil, fmt.Errorf("task %s create has not started yet or pod for task not yet available", lr.Task)
		}
	}
}

func hasTaskRunFailed(trConditions v1beta1.Conditions, taskName string) error {
	if len(trConditions) != 0 && trConditions[0].Status == corev1.ConditionFalse {
		return fmt.Errorf("task %s has failed: %s", taskName, trConditions[0].Message)
	}

	return nil
}
