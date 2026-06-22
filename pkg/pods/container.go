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

package pods

import (
	"bufio"
	"fmt"
	"io"

	"github.com/tektoncd/cli/pkg/pods/stream"
	corev1 "k8s.io/api/core/v1"
)

type Container struct {
	name        string
	NewStreamer stream.NewStreamerFunc
	pod         *Pod
}

func (c *Container) Status() error {
	return c.pod.CheckFailedContainers([]string{c.name})
}

// Log represents one log message from a pod
type Log struct {
	PodName       string
	ContainerName string
	Log           string
}
type LogReader struct {
	containerName string
	pod           *Pod
	follow        bool
	timestamps    bool
}

func (c *Container) LogReader(follow, timestamps bool) *LogReader {
	return &LogReader{c.name, c.pod, follow, timestamps}
}

func (lr *LogReader) Read() (<-chan Log, <-chan error, error) {
	pod := lr.pod
	opts := &corev1.PodLogOptions{
		Follow:     lr.follow,
		Container:  lr.containerName,
		Timestamps: lr.timestamps,
	}

	stream, err := pod.Stream(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting logs for pod %s(%s) : %s", pod.Name, lr.containerName, err)
	}

	logC := make(chan Log)
	errC := make(chan error)

	go func() {
		defer close(logC)
		defer close(errC)
		defer stream.Close()

		r := bufio.NewReader(stream)
		for {
			line, _, err := r.ReadLine()

			if err != nil {
				if err != io.EOF {
					errC <- err
				}
				return
			}

			logC <- Log{
				PodName:       pod.Name,
				ContainerName: lr.containerName,
				Log:           string(line),
			}
		}
	}()

	return logC, errC, nil
}

func (p *Pod) CheckFailedContainers(containerNames []string) error {
	pod, err := p.Get()
	if err != nil {
		return err
	}

	return CheckFailedContainers(pod, containerNames)
}

func CheckFailedContainers(pod *corev1.Pod, containerNames []string) error {
	containerSet := map[string]struct{}{}
	for _, containerName := range containerNames {
		containerSet[containerName] = struct{}{}
	}

	for _, cs := range pod.Status.ContainerStatuses {
		if _, ok := containerSet[cs.Name]; !ok {
			continue
		}

		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
			msg := ""

			if cs.State.Terminated.Reason != "" && cs.State.Terminated.Reason != "Error" {
				msg += " : " + cs.State.Terminated.Reason
			}

			if cs.State.Terminated.Message != "" && cs.State.Terminated.Message != "Error" {
				msg += " : " + cs.State.Terminated.Message
			}

			return fmt.Errorf("container %s has failed %s", cs.Name, msg)
		}
	}

	for _, cs := range pod.Status.InitContainerStatuses {
		if _, ok := containerSet[cs.Name]; !ok {
			continue
		}

		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
			return fmt.Errorf("container %s has failed: %s", cs.Name, cs.State.Terminated.Reason)
		}
	}

	return nil
}
