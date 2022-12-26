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

package pipelinerun

import (
	"context"
	"time"

	"github.com/tektoncd/cli/pkg/actions"
	trh "github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	informers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions"
	"github.com/tektoncd/pipeline/pkg/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// Tracker tracks the progress of a PipelineRun
type Tracker struct {
	Name         string
	Ns           string
	Tekton       versioned.Interface
	ongoingTasks map[string]bool
}

// NewTracker returns a new instance of Tracker
func NewTracker(name string, ns string, tekton versioned.Interface) *Tracker {
	return &Tracker{
		Name:         name,
		Ns:           ns,
		Tekton:       tekton,
		ongoingTasks: map[string]bool{},
	}
}

// Monitor to observe the progress of PipelineRun. It emits
// an event upon starting of a new Pipeline's Task.
// allowed containers the name of the Pipeline tasks, which used as filter
// limit the events to only those tasks
func (t *Tracker) Monitor(allowed []string) <-chan []trh.Run {

	factory := informers.NewSharedInformerFactoryWithOptions(
		t.Tekton,
		time.Second*10,
		informers.WithNamespace(t.Ns),
		informers.WithTweakListOptions(pipelinerunOpts(t.Name)))

	gvr, _ := actions.GetGroupVersionResource(
		schema.GroupVersionResource{Group: "tekton.dev", Resource: "pipelineruns"},
		t.Tekton.Discovery(),
	)

	genericInformer, _ := factory.ForResource(*gvr)
	informer := genericInformer.Informer()

	stopC := make(chan struct{})
	trC := make(chan []trh.Run)

	go func() {
		<-stopC
		close(trC)
	}()

	eventHandler := func(obj interface{}) {
		var pipelinerunConverted v1beta1.PipelineRun
		pr, ok := obj.(*v1beta1.PipelineRun)
		if !ok || pr == nil {
			return
		}

		trMap, runMap, err := status.GetFullPipelineTaskStatuses(context.Background(), t.Tekton, t.Ns, pr)
		if err != nil {
			return
		}
		pr.DeepCopyInto(&pipelinerunConverted)
		pipelinerunConverted.Status.TaskRuns = trMap
		pipelinerunConverted.Status.Runs = runMap

		trC <- t.findNewTaskruns(&pipelinerunConverted, allowed)

		if hasCompleted(&pipelinerunConverted) {
			close(stopC) // should close trC
		}
	}

	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    eventHandler,
			UpdateFunc: func(_, newObj interface{}) { eventHandler(newObj) },
		},
	)

	factory.Start(stopC)
	factory.WaitForCacheSync(stopC)

	return trC
}

func pipelinerunOpts(name string) func(opts *metav1.ListOptions) {
	return func(opts *metav1.ListOptions) {
		opts.FieldSelector = fields.OneTermEqualSelector("metadata.name", name).String()
	}

}

// handles changes to pipelinerun and pushes the Run information to the
// channel if the task is new and is in the allowed list of tasks
// returns true if the pipelinerun has finished
func (t *Tracker) findNewTaskruns(pr *v1beta1.PipelineRun, allowed []string) []trh.Run {
	ret := []trh.Run{}
	for tr, trs := range pr.Status.TaskRuns {
		retries := 0
		if pr.Status.PipelineSpec != nil {
			for _, pipelineTask := range pr.Status.PipelineSpec.Tasks {
				if trs.PipelineTaskName == pipelineTask.Name {
					retries = pipelineTask.Retries
				}
			}
		}
		run := trh.Run{Name: tr, Task: trs.PipelineTaskName, Retries: retries}

		if t.loggingInProgress(tr) ||
			!trh.HasScheduled(trs) ||
			trh.IsFiltered(run, allowed) {
			continue
		}

		t.ongoingTasks[tr] = true
		ret = append(ret, run)
	}

	return ret
}

func hasCompleted(pr *v1beta1.PipelineRun) bool {
	if len(pr.Status.Conditions) == 0 {
		return false
	}

	return pr.Status.Conditions[0].Status != corev1.ConditionUnknown
}

func (t *Tracker) loggingInProgress(tr string) bool {
	_, ok := t.ongoingTasks[tr]
	return ok
}
