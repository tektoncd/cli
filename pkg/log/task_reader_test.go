// Copyright © 2026 The Tekton Authors.
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
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const skippedMessage = `[{"key":"StartedAt","value":"2026-09-16T07:50:38.332Z","type":3},{"key":"Reason","value":"Skipped","type":3}]`

type containerSpec struct {
	name     string
	exitCode int32
	message  string
	waiting  bool
}

func podWithContainers(cs ...containerSpec) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "ns"},
	}

	for _, c := range cs {
		pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: "step-" + c.name})

		state := corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: c.exitCode,
				Message:  c.message,
			},
		}
		if c.waiting {
			state = corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{}}
		}

		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, corev1.ContainerStatus{
			Name:  "step-" + c.name,
			State: state,
		})
	}

	return pod
}

func stepNames(steps []*step) []string {
	names := []string{}
	for _, s := range steps {
		names = append(names, s.name)
	}
	return names
}

func TestFilterSteps(t *testing.T) {
	pod := podWithContainers(
		containerSpec{name: "fetch-repo"},
		containerSpec{name: "codespell"},
		containerSpec{name: "markdownlint", exitCode: 1},
		containerSpec{name: "vale", exitCode: 1, message: skippedMessage},
		containerSpec{name: "goreleaser-check", exitCode: 1, message: skippedMessage},
	)

	for _, tc := range []struct {
		name       string
		pod        *corev1.Pod
		allSteps   bool
		stepsGiven []string
		failedOnly bool
		want       []string
	}{{
		name: "no filtering returns every step",
		pod:  pod,
		want: []string{"fetch-repo", "codespell", "markdownlint", "vale", "goreleaser-check"},
	}, {
		name:       "failed only keeps the step that actually failed",
		pod:        pod,
		failedOnly: true,
		want:       []string{"markdownlint"},
	}, {
		name:       "named steps are returned as given",
		pod:        pod,
		stepsGiven: []string{"codespell", "markdownlint"},
		want:       []string{"codespell", "markdownlint"},
	}, {
		name:       "failed only intersects with named steps",
		pod:        pod,
		stepsGiven: []string{"codespell", "markdownlint"},
		failedOnly: true,
		want:       []string{"markdownlint"},
	}, {
		name:       "failed only drops named steps that passed",
		pod:        pod,
		stepsGiven: []string{"fetch-repo", "codespell"},
		failedOnly: true,
		want:       []string{},
	}, {
		name:       "failed only on a task with no failed step",
		pod:        podWithContainers(containerSpec{name: "fetch-repo"}, containerSpec{name: "codespell"}),
		failedOnly: true,
		want:       []string{},
	}, {
		name:       "failed only ignores steps that never terminated",
		pod:        podWithContainers(containerSpec{name: "fetch-repo", waiting: true}),
		failedOnly: true,
		want:       []string{},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got := stepNames(filterSteps(tc.pod, tc.allSteps, tc.stepsGiven, tc.failedOnly))
			if d := cmp.Diff(tc.want, got); d != "" {
				t.Errorf("unexpected steps: %s", d)
			}
		})
	}
}

func TestIsSkippedStep(t *testing.T) {
	for _, tc := range []struct {
		name    string
		message string
		want    bool
	}{{
		name:    "skipped after an earlier failure",
		message: skippedMessage,
		want:    true,
	}, {
		name:    "genuine failure",
		message: `[{"key":"StartedAt","value":"2026-09-16T07:50:35.810Z","type":3}]`,
	}, {
		name:    "empty termination message",
		message: "",
	}, {
		name:    "non json termination message",
		message: "container terminated",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got := isSkippedStep(&corev1.ContainerStateTerminated{Message: tc.message})
			if got != tc.want {
				t.Errorf("isSkippedStep() = %v, want %v", got, tc.want)
			}
		})
	}
}
