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

	// resolveRoot returns the root PipelineRun this tracker observes, fetching
	// it from the informer cache. It is used for TaskRun events, which do not
	// carry the PipelineRun object.
	resolveRoot := func() *v1.PipelineRun {
		obj, err := genericInformer.Lister().ByNamespace(t.Ns).Get(t.Name)
		if err != nil {
			return nil
		}
		return toPipelineRun(obj)
	}

	// eventHandler recomputes the full tree of task runs from the root
	// PipelineRun. It is driven by both the root PipelineRun informer and the
	// namespace-wide TaskRun informer, so child PipelineRun and TaskRun updates
	// (not just root PR updates) surface newly scheduled tasks while following
	// live logs. Emitting the same set repeatedly is harmless because
	// findNewTaskruns de-duplicates tasks already in progress.
	eventHandler := func(obj interface{}) {
		var pr *v1.PipelineRun
		if obj != nil {
			if typed, ok := obj.(*v1.PipelineRun); ok && typed != nil {
				pr = typed.DeepCopy()
			} else if typed, ok := obj.(*v1beta1.PipelineRun); ok && typed != nil {
				var prv1 v1.PipelineRun
				if err := typed.ConvertTo(context.Background(), &prv1); err != nil {
					return
				}
				pr = &prv1
			}
		}
		if pr == nil {
			pr = resolveRoot()
			if pr == nil {
				return
			}
		}

		trsMap, childPRs, err := GetTaskRunsWithStatus(pr, t.Client, t.Ns)
		if err != nil {
			return
		}
		trC <- t.findNewTaskruns(pr, allowed, trsMap, childPRs)

		if hasCompleted(pr) {
			close(stopC) // should close trC
		}
	}

	// guarded calls eventHandler once while holding the lock, unless a stop
	// signal has already been received.
	guarded := func(obj interface{}) {
		mu.Lock()
		defer mu.Unlock()
		select {
		case <-stopC:
			return
		default:
			eventHandler(obj)
		}
	}

	_, err := informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj interface{}) { guarded(obj) },
			UpdateFunc: func(_, newObj interface{}) { guarded(newObj) },
			DeleteFunc: func(obj interface{}) { guarded(obj) },
		},
	)
	if err != nil {
		return nil
	}

	// Watch TaskRuns in the namespace too. Child PipelineRuns/TaskRuns of a
	// Pipelines-in-Pipelines hierarchy do not emit events on the root PR's
	// field-selected informer, so without this, `logs -f` would stay silent
	// while a long-running child progresses.
	taskGVR, err := actions.GetGroupVersionResource(
		taskrunGroupResource,
		t.Client.Tekton.Discovery(),
	)
	if err != nil {
		return nil
	}
	taskInformer, _ := factory.ForResource(*taskGVR)
	_ = taskInformer.Informer().SetWatchErrorHandlerWithContext(watchErrorHandler)
	_, err = taskInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    func(interface{}) { guarded(nil) },
			UpdateFunc: func(_, _ interface{}) { guarded(nil) },
			DeleteFunc: func(interface{}) { guarded(nil) },
		},
	)
	if err != nil {
		return nil
	}

	factory.Start(stopC)
	// Wait for the root PipelineRun informer to sync before returning so the
	// initial task set is emitted promptly. The TaskRun informer is only used
	// to surface child updates during live tailing; it runs in the background
	// and is stopped with the factory once the run completes.
	if !cache.WaitForCacheSync(stopC, informer.HasSynced) {
		return nil
	}

	return trC
}

// toPipelineRun converts a cached informer object to a *v1.PipelineRun.
func toPipelineRun(obj interface{}) *v1.PipelineRun {
	pr, ok := obj.(*v1.PipelineRun)
	if !ok || pr == nil {
		prV1beta1, ok := obj.(*v1beta1.PipelineRun)
		if !ok || prV1beta1 == nil {
			return nil
		}
		var prv1 v1.PipelineRun
		if err := prV1beta1.ConvertTo(context.Background(), &prv1); err != nil {
			return nil
		}
		return &prv1
	}
	return pr.DeepCopy()
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
func (t *Tracker) findNewTaskruns(pr *v1.PipelineRun, allowed []string, trStatuses map[string]*v1.PipelineRunTaskRunStatus, childPRs map[string]*v1.PipelineRun) []taskrunpkg.Run {
	ret := []taskrunpkg.Run{}
	for tr, trs := range trStatuses {
		retries := 0
		if strings.Contains(trs.PipelineTaskName, taskrunpkg.ChildTaskSeparator) {
			segments := strings.Split(trs.PipelineTaskName, taskrunpkg.ChildTaskSeparator)
			currentPR := pr
		chainLoop:
			for i := 0; i < len(segments)-1; i++ {
				for _, cr := range currentPR.Status.ChildReferences {
					if cr.Kind == "PipelineRun" && cr.PipelineTaskName == segments[i] {
						// Resolve the chain from the PRs already fetched while
						// gathering the task run statuses, to avoid re-fetching
						// each child PipelineRun on every event.
						childPR, ok := childPRs[cr.Name]
						if !ok {
							// Can't resolve the chain; leave retries at 0 rather than guessing.
							currentPR = pr
							break chainLoop
						}
						currentPR = childPR
						break
					}
				}
			}
			leafTaskName := segments[len(segments)-1]
			if currentPR != pr && currentPR.Status.PipelineSpec != nil {
				for _, pt := range currentPR.Status.PipelineSpec.Tasks {
					if pt.Name == leafTaskName {
						retries = pt.Retries
						break
					}
				}
				if retries == 0 {
					for _, pt := range currentPR.Status.PipelineSpec.Finally {
						if pt.Name == leafTaskName {
							retries = pt.Retries
						}
					}
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

func GetTaskRunsWithStatus(pr *v1.PipelineRun, c *cli.Clients, ns string) (map[string]*v1.PipelineRunTaskRunStatus, map[string]*v1.PipelineRun, error) {
	childPRs := map[string]*v1.PipelineRun{}
	trStatuses, err := getTaskRunsWithStatusRecursive(pr, c, ns, "", childPRs)
	return trStatuses, childPRs, err
}

func getTaskRunsWithStatusRecursive(pr *v1.PipelineRun, c *cli.Clients, ns string, prefix string, childPRs map[string]*v1.PipelineRun) (map[string]*v1.PipelineRunTaskRunStatus, error) {
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
				if apierrors.IsNotFound(err) {
					continue
				}
				return nil, err
			}
			taskName := cr.PipelineTaskName
			if prefix != "" {
				taskName = prefix + taskrunpkg.ChildTaskSeparator + taskName
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
			childPRs[cr.Name] = childPR
			childPrefix := cr.PipelineTaskName
			if prefix != "" {
				childPrefix = prefix + taskrunpkg.ChildTaskSeparator + childPrefix
			}
			childTRs, err := getTaskRunsWithStatusRecursive(childPR, c, ns, childPrefix, childPRs)
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
