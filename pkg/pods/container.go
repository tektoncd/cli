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
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tektoncd/cli/pkg/pods/stream"
	corev1 "k8s.io/api/core/v1"
)

type Container struct {
	name        string
	NewStreamer stream.NewStreamerFunc
	pod         *Pod
}

func (c *Container) Status() error {
	pod, err := c.pod.Get()
	if err != nil {
		return err
	}

	container := c.name
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name != container {
			continue
		}

		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode == 1 {
			msg := ""

			if cs.State.Terminated.Reason != "" && cs.State.Terminated.Reason != "Error" {
				msg = msg + " : " + cs.State.Terminated.Reason
			}

			if cs.State.Terminated.Message != "" && cs.State.Terminated.Message != "Error" {
				msg = msg + " : " + cs.State.Terminated.Message
			}

			return fmt.Errorf("container %s has failed %s", container, msg)
		}
	}

	for _, cs := range pod.Status.InitContainerStatuses {
		if cs.Name != container {
			continue
		}

		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode == 1 {
			return fmt.Errorf("container %s has failed: %s", container, cs.State.Terminated.Reason)
		}
	}

	return nil
}

// Log represents one log message from a pod
type Log struct {
	PodName       string
	ContainerName string
	Log           string
	Timestamp     time.Time
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
	return lr.ReadContext(context.Background())
}

// ReadContext reads container logs until the context is cancelled.
func (lr *LogReader) ReadContext(ctx context.Context) (<-chan Log, <-chan error, error) {
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
		done := make(chan struct{})
		defer close(done)

		r := bufio.NewReader(stream)
		go func() {
			select {
			case <-ctx.Done():
				_ = stream.Close()
			case <-done:
			}
		}()

		for {
			line, err := r.ReadString('\n')

			if len(line) > 0 {
				line = strings.TrimSuffix(line, "\n")
				line = strings.TrimSuffix(line, "\r")

				logEntry := Log{
					PodName:       pod.Name,
					ContainerName: lr.containerName,
					Log:           line,
				}

				if lr.timestamps {
					parts := strings.SplitN(line, " ", 2)

					if len(parts) == 2 {
						if t, parseErr := time.Parse(time.RFC3339Nano, parts[0]); parseErr == nil {
							logEntry.Timestamp = t
							logEntry.Log = parts[1]
						}
					}
				}

				select {
				case logC <- logEntry:
				case <-ctx.Done():
					return
				}
			}

			if err != nil {
				if err != io.EOF {
					select {
					case errC <- err:
					case <-ctx.Done():
					}
				}
				return
			}
		}
	}()

	return logC, errC, nil
}
