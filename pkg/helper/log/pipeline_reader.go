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
	"sync"
	"sync/atomic"
	"time"

	"github.com/tektoncd/cli/pkg/helper/pipelinerun"
	trh "github.com/tektoncd/cli/pkg/helper/taskrun"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

func (r *Reader) readPipelineLog() (<-chan Log, <-chan error, error) {
	tkn := r.clients.Tekton
	pr, err := tkn.TektonV1alpha1().PipelineRuns(r.ns).Get(r.run, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	if r.follow {
		return r.readLivePipelineLogs(pr)
	}
	return r.readAvailablePipelineLogs(pr)
}

func (r *Reader) readLivePipelineLogs(pr *v1alpha1.PipelineRun) (<-chan Log, <-chan error, error) {
	logC := make(chan Log)
	errC := make(chan error)

	go func() {
		defer close(logC)
		defer close(errC)

		prTracker := pipelinerun.NewTracker(pr.Name, r.ns, r.clients.Tekton)
		trC := prTracker.Monitor(r.tasks)

		wg := sync.WaitGroup{}
		taskIndex := int32(1)

		for trs := range trC {
			wg.Add(len(trs))

			for _, run := range trs {
				// NOTE: passing tr, taskIdx to avoid data race
				go func(tr trh.Run, taskNum int32) {
					defer wg.Done()

					// clone the object to keep task number and name separately
					c := r.clone()
					c.setUpTask(int(taskNum), tr)
					c.pipeLogs(logC, errC)
				}(run, atomic.AddInt32(&taskIndex, 1))
			}
		}

		wg.Wait()

		if !empty(pr.Status) && pr.Status.Conditions[0].Status == corev1.ConditionFalse {
			errC <- fmt.Errorf(pr.Status.Conditions[0].Message)
		}
	}()

	return logC, errC, nil
}

func (r *Reader) readAvailablePipelineLogs(pr *v1alpha1.PipelineRun) (<-chan Log, <-chan error, error) {
	if err := r.waitUntilAvailable(10); err != nil {
		return nil, nil, err
	}

	tkn := r.clients.Tekton
	pl, err := tkn.TektonV1alpha1().Pipelines(r.ns).Get(pr.Spec.PipelineRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	// Sort taskruns, to display the taskrun logs as per pipeline tasks order
	ordered := trh.SortTasksBySpecOrder(pl.Spec.Tasks, pr.Status.TaskRuns)
	taskRuns := trh.Filter(ordered, r.tasks)

	logC := make(chan Log)
	errC := make(chan error)

	go func() {
		defer close(logC)
		defer close(errC)

		for i, tr := range taskRuns {
			// clone the object to keep task number and name separately
			c := r.clone()
			c.setUpTask(i+1, tr)
			c.pipeLogs(logC, errC)
		}

		if !empty(pr.Status) && pr.Status.Conditions[0].Status == corev1.ConditionFalse {
			errC <- fmt.Errorf(pr.Status.Conditions[0].Message)
		}
	}()

	return logC, errC, nil
}

// reading of logs should wait till the status of run is unknown
// only if run status is unknown, open a watch channel on run
// and keep checking the status until it changes to true|false
// or the reach timeout
func (r *Reader) waitUntilAvailable(timeout time.Duration) error {
	var first = true
	opts := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", r.run).String(),
	}
	tkn := r.clients.Tekton
	run, err := tkn.TektonV1alpha1().PipelineRuns(r.ns).Get(r.run, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if empty(run.Status) {
		return nil
	}
	if run.Status.Conditions[0].Status != corev1.ConditionUnknown {
		return nil
	}

	watchRun, err := tkn.TektonV1alpha1().PipelineRuns(r.ns).Watch(opts)
	if err != nil {
		return err
	}
	for {
		select {
		case event := <-watchRun.ResultChan():
			if event.Object.(*v1alpha1.PipelineRun).IsDone() {
				watchRun.Stop()
				return nil
			}
			if first {
				first = false
				fmt.Fprintln(r.stream.Out, "Pipeline still running ...")
			}
		case <-time.After(timeout * time.Second):
			watchRun.Stop()
			fmt.Fprintln(r.stream.Err, "No logs found")
			return nil
		}
	}
}

func (r *Reader) pipeLogs(logC chan<- Log, errC chan<- error) {
	tlogC, terrC, err := r.readTaskLog()
	if err != nil {
		errC <- err
		return
	}

	for tlogC != nil || terrC != nil {
		select {
		case l, ok := <-tlogC:
			if !ok {
				tlogC = nil
				continue
			}
			logC <- Log{Task: l.Task, Step: l.Step, Log: l.Log}

		case e, ok := <-terrC:
			if !ok {
				terrC = nil
				continue
			}
			errC <- fmt.Errorf("failed to get logs for task %s : %s", r.task, e)
		}
	}
}

func (r *Reader) setUpTask(taskNumber int, tr trh.Run) {
	r.setNumber(taskNumber)
	r.setRun(tr.Name)
	r.setTask(tr.Task)
}

func empty(status v1alpha1.PipelineRunStatus) bool {
	if status.Conditions == nil {
		return true
	}
	return len(status.Conditions) == 0
}
