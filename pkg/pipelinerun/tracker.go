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
	"errors"
	"strings"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	taskrunpkg "github.com/tektoncd/cli/pkg/taskrun"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	informers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

// ChildTaskSeparator separates parent and child pipeline task names in Pipelines-in-Pipelines describe/logs output.
const ChildTaskSeparator = " > "

// Tracker tracks the progress of a PipelineRun
type Tracker struct {
	Name         string
	Ns           string
	Client       *cli.Clients
	ongoingTasks map[string]bool
}

// NewTracker returns a new instance of Tracker
func NewTracker(name string, ns string, client *cli.Clients) *Tracker {
	return &Tracker{
		Name:         name,
		Ns:           ns,
		Client:       client,
		ongoingTasks: map[string]bool{},
	}
}

// Monitor to observe the progress of PipelineRun. It emits
// an event upon starting of a new Pipeline's Task.
// allowed containers the name of the Pipeline tasks, which used as filter
// limit the events to only those tasks
func (t *Tracker) Monitor(allowed []string) <-chan []taskrunpkg.Run {

	factory := informers.NewSharedInformerFactoryWithOptions(
		t.Client.Tekton,
		time.Second*10,
		informers.WithNamespace(t.Ns),
		informers.WithTweakListOptions(pipelinerunOpts(t.Name)))

	gvr, _ := actions.GetGroupVersionResource(
		pipelineRunGroupResource,
		t.Client.Tekton.Discovery(),
	)

	genericInformer, _ := factory.ForResource(*gvr)
	informer := genericInformer.Informer()

	// Set a custom watch error handler that ignores context.Canceled errors
	// to prevent "Failed to watch" log messages when the informer is stopped intentionally
	_ = informer.SetWatchErrorHandlerWithContext(watchErrorHandler)

	mu := &sync.Mutex{}
	stopC := make(chan struct{})
	trC := make(chan []taskrunpkg.Run)
	go func() {
		<-stopC
		close(trC)
	}()

	eventHandler := func(obj interface{}) {
		var pipelinerunConverted v1.PipelineRun
		pr, ok := obj.(*v1.PipelineRun)
		if !ok || pr == nil {
			prV1beta1, ok := obj.(*v1beta1.PipelineRun)
			if !ok || prV1beta1 == nil {
				return
			}
			var prv1 v1.PipelineRun
			err := prV1beta1.ConvertTo(context.Background(), &prv1)
			if err != nil {
				return
			}
			pr = &prv1
		}

		trsMap, err := GetTaskRunsWithStatus(pr, t.Client, t.Ns)
		if err != nil {
			return
		}
		pr.DeepCopyInto(&pipelinerunConverted)
		trC <- t.findNewTaskruns(&pipelinerunConverted, allowed, trsMap)

		if hasCompleted(&pipelinerunConverted) {
			close(stopC) // should close trC
		}
	}

	_, err := informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				// To ensure synchonization and checks is the stopC channel has received a signal to stop
				// If it receives a signal then return and does nothing
				mu.Lock()
				defer mu.Unlock()
				select {
				case <-stopC:
					return
				default:
					eventHandler(obj)
				}
			},
			UpdateFunc: func(_, newObj interface{}) {
				mu.Lock()
				defer mu.Unlock()
				select {
				case <-stopC:
					return
				default:
					eventHandler(newObj)
				}
			},
			DeleteFunc: func(obj interface{}) {
				mu.Lock()
				defer mu.Unlock()
				select {
				case <-stopC:
					return
				default:
					eventHandler(obj)
				}
			},
		},
	)
	if err != nil {
		return nil
	}

	factory.Start(stopC)
	factory.WaitForCacheSync(stopC)

	return trC
}

func pipelinerunOpts(name string) func(opts *metav1.ListOptions) {
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

// handles changes to pipelinerun and pushes the Run information to the
// channel if the task is new and is in the allowed list of tasks
// returns true if the pipelinerun has finished
func (t *Tracker) findNewTaskruns(pr *v1.PipelineRun, allowed []string, trStatuses map[string]*v1.PipelineRunTaskRunStatus) []taskrunpkg.Run {
	ret := []taskrunpkg.Run{}
	for tr, trs := range trStatuses {
		retries := 0
		if strings.Contains(trs.PipelineTaskName, ChildTaskSeparator) {
			segments := strings.SplitN(trs.PipelineTaskName, ChildTaskSeparator, 2)
			parentTaskName := segments[0]
			childTaskName := segments[1]
			for _, cr := range pr.Status.ChildReferences {
				if cr.Kind == "PipelineRun" && cr.PipelineTaskName == parentTaskName {
					childPR, err := GetPipelineRun(pipelineRunGroupResource, t.Client, cr.Name, t.Ns)
					if err == nil && childPR.Status.PipelineSpec != nil {
						for _, pt := range childPR.Status.PipelineSpec.Tasks {
							if pt.Name == childTaskName {
								retries = pt.Retries
								break
							}
						}
					}
					break
				}
			}
		} else if pr.Status.PipelineSpec != nil {
			for _, pipelineTask := range pr.Status.PipelineSpec.Tasks {
				if trs.PipelineTaskName == pipelineTask.Name {
					retries = pipelineTask.Retries
				}
			}
		}
		run := taskrunpkg.Run{Name: tr, Task: trs.PipelineTaskName, Retries: retries}

		if t.loggingInProgress(tr) ||
			!taskrunpkg.HasScheduled(trs) ||
			taskrunpkg.IsFiltered(run, allowed) {
			continue
		}

		t.ongoingTasks[tr] = true
		ret = append(ret, run)
	}

	return ret
}

func hasCompleted(pr *v1.PipelineRun) bool {
	if len(pr.Status.Conditions) == 0 {
		return false
	}
	return pr.Status.Conditions[0].Status != corev1.ConditionUnknown
}

func (t *Tracker) loggingInProgress(tr string) bool {
	_, ok := t.ongoingTasks[tr]
	return ok
}

func GetTaskRunsWithStatus(pr *v1.PipelineRun, c *cli.Clients, ns string) (map[string]*v1.PipelineRunTaskRunStatus, error) {
	return getTaskRunsWithStatusRecursive(pr, c, ns, "")
}

func getTaskRunsWithStatusRecursive(pr *v1.PipelineRun, c *cli.Clients, ns string, prefix string) (map[string]*v1.PipelineRunTaskRunStatus, error) {
	if pr == nil {
		return nil, nil
	}
	if len(pr.Status.ChildReferences) == 0 {
		return map[string]*v1.PipelineRunTaskRunStatus{}, nil
	}
	trStatuses := make(map[string]*v1.PipelineRunTaskRunStatus)
	for _, cr := range pr.Status.ChildReferences {
		switch cr.Kind {
		case "TaskRun":
			tr, err := taskrunpkg.GetTaskRun(taskrunGroupResource, c, cr.Name, ns)
			if err != nil {
				return nil, err
			}
			taskName := cr.PipelineTaskName
			if prefix != "" {
				taskName = prefix + ChildTaskSeparator + taskName
			}
			trStatuses[cr.Name] = &v1.PipelineRunTaskRunStatus{
				PipelineTaskName: taskName,
				Status:           &tr.Status,
			}
		case "PipelineRun":
			childPR, err := GetPipelineRun(pipelineRunGroupResource, c, cr.Name, ns)
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return nil, err
			}
			childPrefix := cr.PipelineTaskName
			if prefix != "" {
				childPrefix = prefix + ChildTaskSeparator + childPrefix
			}
			childTRs, err := getTaskRunsWithStatusRecursive(childPR, c, ns, childPrefix)
			if err != nil {
				return nil, err
			}
			for k, v := range childTRs {
				trStatuses[k] = v
			}
		}
	}
	return trStatuses, nil
}
