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
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/tektoncd/cli/pkg/pods/stream"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	k8s "k8s.io/client-go/kubernetes"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
)

type Stream struct {
	name string
	pods typedv1.PodInterface
	opts *corev1.PodLogOptions
}

func NewStream(pods typedv1.PodInterface, name string, opts *corev1.PodLogOptions) stream.Streamer {
	return &Stream{name, pods, opts}
}

// Stream Creates a stream object which allows reading the logs
func (s *Stream) Stream() (io.ReadCloser, error) {
	return s.pods.GetLogs(s.name, s.opts).Stream(context.Background())
}

type Pod struct {
	Name     string
	Ns       string
	Kc       k8s.Interface
	Streamer stream.NewStreamerFunc
}

func New(name, ns string, client k8s.Interface, streamer stream.NewStreamerFunc) *Pod {
	return &Pod{
		Name: name, Ns: ns,
		Kc:       client,
		Streamer: streamer,
	}
}

func NewWithDefaults(name, ns string, client k8s.Interface) *Pod {
	return &Pod{
		Name: name, Ns: ns,
		Kc:       client,
		Streamer: NewStream,
	}
}

// Wait wait for the pod to get up and running
func (p *Pod) Wait() (*corev1.Pod, error) {
	for {
		pod, err := p.Get()
		if err != nil {
			return nil, err
		}

		if readyPod, err := checkPodStatus(pod); readyPod != nil || err != nil {
			return readyPod, err
		}

		watcher, err := p.Kc.CoreV1().Pods(p.Ns).Watch(context.Background(), metav1.ListOptions{
			FieldSelector:   fields.OneTermEqualSelector("metadata.name", p.Name).String(),
			ResourceVersion: pod.ResourceVersion,
		})
		if err != nil {
			return nil, err
		}

		retry := false
		for event := range watcher.ResultChan() {
			updatedPod, ok, err := podFromWatchEvent(event)
			if err != nil {
				watcher.Stop()
				return nil, err
			}
			if !ok {
				retry = true
				break
			}
			if readyPod, err := checkPodStatus(updatedPod); readyPod != nil || err != nil {
				watcher.Stop()
				return readyPod, err
			}
		}
		watcher.Stop()

		if retry {
			continue
		}
	}
}

func podOpts(name string) func(opts *metav1.ListOptions) {
	return func(opts *metav1.ListOptions) {
		opts.FieldSelector = fields.OneTermEqualSelector("metadata.name", name).String()
	}
}

// watchErrorHandler is a custom watch error handler that filters out context.Canceled errors
// to prevent "Failed to watch" log messages when the informer is stopped intentionally.
// Other errors are passed to the default handler.
func watchErrorHandler(ctx context.Context, r *cache.Reflector, err error) {
	if !errors.Is(err, context.Canceled) {
		cache.DefaultWatchErrorHandler(ctx, r, err)
	}
}

func checkPodStatus(obj interface{}) (*corev1.Pod, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("failed to cast to pod object")
	}

	if pod.DeletionTimestamp != nil {
		return pod, fmt.Errorf("failed to run the pod %s ", pod.Name)
	}

	if pod.Status.Phase == corev1.PodSucceeded ||
		pod.Status.Phase == corev1.PodRunning ||
		pod.Status.Phase == corev1.PodFailed {
		return pod, nil
	}

	// Handle any issues with pulling images that may fail
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodInitialized || c.Type == corev1.ContainersReady {
			if c.Status == corev1.ConditionUnknown {
				return pod, fmt.Errorf("%s", c.Message)
			}
		}
	}

	return nil, nil
}

func podFromWatchEvent(event watch.Event) (*corev1.Pod, bool, error) {
	if event.Type == watch.Error {
		return nil, false, nil
	}

	pod, ok := event.Object.(*corev1.Pod)
	if !ok {
		return nil, false, nil
	}
	return pod, true, nil
}

// Get gets the pod
func (p *Pod) Get() (*corev1.Pod, error) {
	return p.Kc.CoreV1().Pods(p.Ns).Get(context.Background(), p.Name, metav1.GetOptions{})
}

// Container returns the an instance of Container
func (p *Pod) Container(c string) *Container {
	return &Container{
		name:        c,
		pod:         p,
		NewStreamer: p.Streamer,
	}
}

// Stream returns the stream object for given container and mode
// in order to fetch the logs
func (p *Pod) Stream(opt *corev1.PodLogOptions) (io.ReadCloser, error) {
	pods := p.Kc.CoreV1().Pods(p.Ns)
	if pods == nil {
		return nil, fmt.Errorf("error getting pods")
	}

	return p.Streamer(pods, p.Name, opt).Stream()
}
