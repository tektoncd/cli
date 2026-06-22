package log

import (
	"testing"

	"github.com/tektoncd/cli/pkg/cli"
	podsfake "github.com/tektoncd/cli/pkg/pods/fake"
	"github.com/tektoncd/cli/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReadPodLogs_stopsAfterFailedStep(t *testing.T) {
	const (
		ns      = "ns"
		podName = "pod"
	)

	pods := []*corev1.Pod{{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "step-first"},
				{Name: "step-second"},
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "step-first",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 137,
							Reason:   "OOMKilled",
						},
					},
				},
				{
					Name: "step-second",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 0,
						},
					},
				},
			},
		},
	}}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Pods: pods})
	reader := &Reader{
		ns:      ns,
		clients: &cli.Clients{Kube: cs.Kube},
		streamer: podsfake.Streamer(podsfake.Logs(
			podsfake.Task(podName,
				podsfake.Step("step-first", "first-log"),
				podsfake.Step("step-second", "second-log"),
			),
		)),
		task: "task",
	}

	podC := make(chan string, 1)
	podC <- podName
	close(podC)

	logC, errC := reader.readPodLogs(podC, nil, false, false)

	var logs []Log
	var errs []error
	for logC != nil || errC != nil {
		select {
		case l, ok := <-logC:
			if !ok {
				logC = nil
				continue
			}
			logs = append(logs, l)
		case err, ok := <-errC:
			if !ok {
				errC = nil
				continue
			}
			errs = append(errs, err)
		}
	}

	if len(logs) != 2 {
		t.Fatalf("expected first step log and EOF only, got %#v", logs)
	}
	if logs[0].Step != "first" || logs[0].Log != "first-log" {
		t.Fatalf("unexpected first log: %#v", logs[0])
	}
	if logs[1].Step != "first" || logs[1].Log != "EOFLOG" {
		t.Fatalf("unexpected EOF log: %#v", logs[1])
	}
	if len(errs) != 1 {
		t.Fatalf("expected one error, got %#v", errs)
	}
	if errs[0] == nil || errs[0].Error() == "" {
		t.Fatalf("expected non-empty error, got %#v", errs[0])
	}
}
