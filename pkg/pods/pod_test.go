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
	"testing"
	"time"

	"github.com/tektoncd/cli/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	k8s "k8s.io/client-go/kubernetes"
	k8stest "k8s.io/client-go/testing"
)

func Test_wait_pod_initialized(t *testing.T) {
	podname := "test"
	ns := "ns"

	initial := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podname,
			Namespace: ns,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}
	later := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podname,
			Namespace: ns,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}
	kc := simulateAddWatch(t, &initial, &later)

	pod := NewWithDefaults(podname, ns, kc)
	p, err := pod.Wait()

	if p == nil {
		t.Errorf("Unexpected p mismatch: \n%s\n", p)
	}

	if err != nil {
		t.Errorf("Unexpected error: \n%s\n", err)
	}
}

func Test_wait_pod_success(t *testing.T) {
	podname := "test"
	ns := "ns"

	initial := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podname,
			Namespace: ns,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}
	later := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podname,
			Namespace: ns,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodSucceeded,
		},
	}
	kc := simulateAddWatch(t, &initial, &later)

	pod := NewWithDefaults(podname, ns, kc)
	p, err := pod.Wait()

	if p == nil {
		t.Errorf("Unexpected output mismatch: \n%s\n", p)
	}

	if err != nil {
		t.Errorf("Unexpected error: \n%s\n", err)
	}
}

func Test_wait_pod_ready_without_watch(t *testing.T) {
	podname := "test"
	ns := "ns"

	clients, _ := test.SeedV1beta1TestData(t, test.Data{Pods: []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podname,
				Namespace: ns,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		},
	}})

	pod := NewWithDefaults(podname, ns, clients.Kube)
	p, err := pod.Wait()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil || p.Name != podname {
		t.Fatalf("unexpected pod result: %#v", p)
	}
}

func Test_wait_pod_retries_when_watch_closes(t *testing.T) {
	podname := "test"
	ns := "ns"

	initial := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            podname,
			Namespace:       ns,
			ResourceVersion: "1",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}
	running := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            podname,
			Namespace:       ns,
			ResourceVersion: "2",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	clients, _ := test.SeedV1beta1TestData(t, test.Data{Pods: []*corev1.Pod{initial}})
	firstWatch := watch.NewFake()
	secondWatch := watch.NewFake()
	watchCalls := 0
	clients.Kube.PrependWatchReactor("pods", func(_ k8stest.Action) (bool, watch.Interface, error) {
		watchCalls++
		if watchCalls == 1 {
			go func() {
				time.Sleep(50 * time.Millisecond)
				firstWatch.Stop()
			}()
			return true, firstWatch, nil
		}

		go func() {
			time.Sleep(50 * time.Millisecond)
			secondWatch.Modify(running)
		}()
		return true, secondWatch, nil
	})

	pod := NewWithDefaults(podname, ns, clients.Kube)
	p, err := pod.Wait()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil || p.Status.Phase != corev1.PodRunning {
		t.Fatalf("unexpected pod result: %#v", p)
	}
	if watchCalls < 2 {
		t.Fatalf("expected Wait to retry watch, got %d watch calls", watchCalls)
	}
}

func Test_wait_pod_fail(t *testing.T) {
	podname := "test"
	ns := "ns"

	initial := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podname,
			Namespace: ns,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}
	later := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podname,
			Namespace: ns,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodInitialized,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	kc := simulateAddWatch(t, &initial, &later)

	pod := NewWithDefaults(podname, ns, kc)
	p, err := pod.Wait()

	if p == nil {
		t.Errorf("Unexpected output mismatch: \n%s\n", p)
	}

	if err != nil {
		t.Errorf("Unexpected error: \n%s\n", err)
	}
}

func Test_wait_pod_imagepull_error(t *testing.T) {
	podname := "test"
	ns := "ns"

	initial := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podname,
			Namespace: ns,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}

	deletionTime := metav1.Now()
	later := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              podname,
			Namespace:         ns,
			DeletionTimestamp: &deletionTime,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
		},
	}

	kc := simulateDeleteWatch(t, &initial, &later)
	pod := NewWithDefaults(podname, ns, kc)
	p, err := pod.Wait()

	if p == nil {
		t.Errorf("Unexpected output mismatch: \n%s\n", p)
	}

	if err == nil {
		t.Errorf("Unexpected error type %v", err)
	}
}

func simulateAddWatch(t *testing.T, initial *corev1.Pod, later *corev1.Pod) k8s.Interface {
	ps := []*corev1.Pod{
		initial,
	}

	clients, _ := test.SeedV1beta1TestData(t, test.Data{Pods: ps})
	watcher := watch.NewFake()
	clients.Kube.PrependWatchReactor("pods", k8stest.DefaultWatchReactor(watcher, nil))

	go func() {
		time.Sleep(2 * time.Second)
		watcher.Add(later)
	}()

	return clients.Kube
}

func simulateDeleteWatch(t *testing.T, initial *corev1.Pod, later *corev1.Pod) k8s.Interface {
	ps := []*corev1.Pod{
		initial,
	}

	clients, _ := test.SeedV1beta1TestData(t, test.Data{Pods: ps})
	watcher := watch.NewFake()
	clients.Kube.PrependWatchReactor("pods", k8stest.DefaultWatchReactor(watcher, nil))

	go func() {
		time.Sleep(2 * time.Second)
		watcher.Delete(later)
	}()

	return clients.Kube
}

func Test_watchErrorHandler(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "context.Canceled should be filtered",
			err:  context.Canceled,
		},
		{
			name: "wrapped context.Canceled should be filtered",
			err:  errors.Join(errors.New("watch failed"), context.Canceled),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// Call watchErrorHandler with context.Canceled errors
			// These should be filtered (not passed to DefaultWatchErrorHandler)
			// so passing nil reflector is safe
			watchErrorHandler(context.Background(), nil, tt.err)
		})
	}
}
