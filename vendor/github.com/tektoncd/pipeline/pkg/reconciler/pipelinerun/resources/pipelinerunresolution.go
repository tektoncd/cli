/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	"fmt"
	"reflect"

	"go.uber.org/zap"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"knative.dev/pkg/apis"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/list"
	"github.com/tektoncd/pipeline/pkg/names"
	"github.com/tektoncd/pipeline/pkg/reconciler/taskrun/resources"
)

const (
	// ReasonRunning indicates that the reason for the inprogress status is that the TaskRun
	// is just starting to be reconciled
	ReasonRunning = "Running"

	// ReasonFailed indicates that the reason for the failure status is that one of the TaskRuns failed
	ReasonFailed = "Failed"

	// ReasonSucceeded indicates that the reason for the finished status is that all of the TaskRuns
	// completed successfully
	ReasonSucceeded = "Succeeded"

	// ReasonTimedOut indicates that the PipelineRun has taken longer than its configured
	// timeout
	ReasonTimedOut = "PipelineRunTimeout"

	// ReasonConditionCheckFailed indicates that the reason for the failure status is that the
	// condition check associated to the pipeline task evaluated to false
	ReasonConditionCheckFailed = "ConditionCheckFailed"
)

// ResolvedPipelineRunTask contains a Task and its associated TaskRun, if it
// exists. TaskRun can be nil to represent there being no TaskRun.
type ResolvedPipelineRunTask struct {
	TaskRunName           string
	TaskRun               *v1alpha1.TaskRun
	PipelineTask          *v1alpha1.PipelineTask
	ResolvedTaskResources *resources.ResolvedTaskResources
	// ConditionChecks ~~TaskRuns but for evaling conditions
	ResolvedConditionChecks TaskConditionCheckState // Could also be a TaskRun or maybe just a Pod?
}

// PipelineRunState is a slice of ResolvedPipelineRunTasks the represents the current execution
// state of the PipelineRun.
type PipelineRunState []*ResolvedPipelineRunTask

func (t ResolvedPipelineRunTask) IsDone() (isDone bool) {
	if t.TaskRun == nil || t.PipelineTask == nil {
		return
	}

	status := t.TaskRun.Status.GetCondition(apis.ConditionSucceeded)
	retriesDone := len(t.TaskRun.Status.RetriesStatus)
	retries := t.PipelineTask.Retries
	isDone = status.IsTrue() || status.IsFalse() && retriesDone >= retries
	return
}

// IsSuccessful returns true only if the taskrun itself has completed successfully
func (t ResolvedPipelineRunTask) IsSuccessful() bool {
	if t.TaskRun == nil {
		return false
	}
	c := t.TaskRun.Status.GetCondition(apis.ConditionSucceeded)
	if c == nil {
		return false
	}

	if c.Status == corev1.ConditionTrue {
		return true
	}
	return false
}

// IsFailed returns true only if the taskrun itself has failed
func (t ResolvedPipelineRunTask) IsFailure() bool {
	if t.TaskRun == nil {
		return false
	}
	c := t.TaskRun.Status.GetCondition(apis.ConditionSucceeded)
	retriesDone := len(t.TaskRun.Status.RetriesStatus)
	retries := t.PipelineTask.Retries
	return c.IsFalse() && retriesDone >= retries
}

func (state PipelineRunState) toMap() map[string]*ResolvedPipelineRunTask {
	m := make(map[string]*ResolvedPipelineRunTask)
	for _, rprt := range state {
		m[rprt.PipelineTask.Name] = rprt
	}
	return m
}

func (state PipelineRunState) IsDone() (isDone bool) {
	isDone = true
	for _, t := range state {
		if t.TaskRun == nil || t.PipelineTask == nil {
			return false
		}
		isDone = isDone && t.IsDone()
		if !isDone {
			return
		}
	}
	return
}

// GetNextTasks will return the next ResolvedPipelineRunTasks to execute, which are the ones in the
// list of candidateTasks which aren't yet indicated in state to be running.
func (state PipelineRunState) GetNextTasks(candidateTasks map[string]v1alpha1.PipelineTask) []*ResolvedPipelineRunTask {
	tasks := []*ResolvedPipelineRunTask{}
	for _, t := range state {
		if _, ok := candidateTasks[t.PipelineTask.Name]; ok && t.TaskRun == nil {
			tasks = append(tasks, t)
		}
		if _, ok := candidateTasks[t.PipelineTask.Name]; ok && t.TaskRun != nil {
			status := t.TaskRun.Status.GetCondition(apis.ConditionSucceeded)
			if status != nil && status.IsFalse() {
				if !(t.TaskRun.IsCancelled() || status.Reason == v1alpha1.TaskRunSpecStatusCancelled || status.Reason == ReasonConditionCheckFailed) {
					if len(t.TaskRun.Status.RetriesStatus) < t.PipelineTask.Retries {
						tasks = append(tasks, t)
					}
				}
			}
		}
	}
	return tasks
}

// SuccessfulPipelineTaskNames returns a list of the names of all of the PipelineTasks in state
// which have successfully completed.
func (state PipelineRunState) SuccessfulPipelineTaskNames() []string {
	done := []string{}
	for _, t := range state {
		if t.TaskRun != nil {
			c := t.TaskRun.Status.GetCondition(apis.ConditionSucceeded)
			if c.IsTrue() {
				done = append(done, t.PipelineTask.Name)
			}
		}
	}
	return done
}

// GetTaskRun is a function that will retrieve the TaskRun name.
type GetTaskRun func(name string) (*v1alpha1.TaskRun, error)

// GetResourcesFromBindings will validate that all PipelineResources declared in Pipeline p are bound in PipelineRun pr
// and if so, will return a map from the declared name of the PipelineResource (which is how the PipelineResource will
// be referred to in the PipelineRun) to the ResourceRef.
func GetResourcesFromBindings(p *v1alpha1.Pipeline, pr *v1alpha1.PipelineRun) (map[string]v1alpha1.PipelineResourceRef, error) {
	resources := map[string]v1alpha1.PipelineResourceRef{}

	required := make([]string, 0, len(p.Spec.Resources))
	for _, resource := range p.Spec.Resources {
		required = append(required, resource.Name)
	}
	provided := make([]string, 0, len(pr.Spec.Resources))
	for _, resource := range pr.Spec.Resources {
		provided = append(provided, resource.Name)
	}
	err := list.IsSame(required, provided)
	if err != nil {
		return resources, xerrors.Errorf("PipelineRun bound resources didn't match Pipeline: %w", err)
	}

	for _, resource := range pr.Spec.Resources {
		resources[resource.Name] = resource.ResourceRef
	}
	return resources, nil
}

func getPipelineRunTaskResources(pt v1alpha1.PipelineTask, providedResources map[string]v1alpha1.PipelineResourceRef) ([]v1alpha1.TaskResourceBinding, []v1alpha1.TaskResourceBinding, error) {
	inputs, outputs := []v1alpha1.TaskResourceBinding{}, []v1alpha1.TaskResourceBinding{}
	if pt.Resources != nil {
		for _, taskInput := range pt.Resources.Inputs {
			resource, ok := providedResources[taskInput.Resource]
			if !ok {
				return inputs, outputs, xerrors.Errorf("pipelineTask tried to use input resource %s not present in declared resources", taskInput.Resource)
			}
			inputs = append(inputs, v1alpha1.TaskResourceBinding{
				Name:        taskInput.Name,
				ResourceRef: resource,
			})
		}
		for _, taskOutput := range pt.Resources.Outputs {
			resource, ok := providedResources[taskOutput.Resource]
			if !ok {
				return outputs, outputs, xerrors.Errorf("pipelineTask tried to use output resource %s not present in declared resources", taskOutput.Resource)
			}
			outputs = append(outputs, v1alpha1.TaskResourceBinding{
				Name:        taskOutput.Name,
				ResourceRef: resource,
			})
		}
	}
	return inputs, outputs, nil
}

// TaskNotFoundError indicates that the resolution failed because a referenced Task couldn't be retrieved
type TaskNotFoundError struct {
	Name string
	Msg  string
}

func (e *TaskNotFoundError) Error() string {
	return fmt.Sprintf("Couldn't retrieve Task %q: %s", e.Name, e.Msg)
}

// ResourceNotFoundError indicates that the resolution failed because a referenced PipelineResource couldn't be retrieved
type ResourceNotFoundError struct {
	Msg string
}

func (e *ResourceNotFoundError) Error() string {
	return fmt.Sprintf("Couldn't retrieve PipelineResource: %s", e.Msg)
}

type ConditionNotFoundError struct {
	Name string
	Msg  string
}

func (e *ConditionNotFoundError) Error() string {
	return fmt.Sprintf("Couldn't retrieve Condition %q: %s", e.Name, e.Msg)
}

// ResolvePipelineRun retrieves all Tasks instances which are reference by tasks, getting
// instances from getTask. If it is unable to retrieve an instance of a referenced Task, it
// will return an error, otherwise it returns a list of all of the Tasks retrieved.
// It will retrieve the Resources needed for the TaskRun as well using getResource and the mapping
// of providedResources.
func ResolvePipelineRun(
	pipelineRun v1alpha1.PipelineRun,
	getTask resources.GetTask,
	getTaskRun resources.GetTaskRun,
	getClusterTask resources.GetClusterTask,
	getResource resources.GetResource,
	getCondition GetCondition,
	tasks []v1alpha1.PipelineTask,
	providedResources map[string]v1alpha1.PipelineResourceRef,
) (PipelineRunState, error) {

	state := []*ResolvedPipelineRunTask{}
	for i := range tasks {
		pt := tasks[i]

		rprt := ResolvedPipelineRunTask{
			PipelineTask: &pt,
			TaskRunName:  getTaskRunName(pipelineRun.Status.TaskRuns, pt.Name, pipelineRun.Name),
		}

		// Find the Task that this PipelineTask is using
		var t v1alpha1.TaskInterface
		var err error
		if pt.TaskRef.Kind == v1alpha1.ClusterTaskKind {
			t, err = getClusterTask(pt.TaskRef.Name)
		} else {
			t, err = getTask(pt.TaskRef.Name)
		}
		if err != nil {
			return nil, &TaskNotFoundError{
				Name: pt.TaskRef.Name,
				Msg:  err.Error(),
			}
		}

		// Get all the resources that this task will be using, if any
		inputs, outputs, err := getPipelineRunTaskResources(pt, providedResources)
		if err != nil {
			return nil, xerrors.Errorf("unexpected error which should have been caught by Pipeline webhook: %w", err)
		}

		spec := t.TaskSpec()
		rtr, err := resources.ResolveTaskResources(&spec, t.TaskMetadata().Name, pt.TaskRef.Kind, inputs, outputs, getResource)
		if err != nil {
			return nil, &ResourceNotFoundError{Msg: err.Error()}
		}
		rprt.ResolvedTaskResources = rtr

		taskRun, err := getTaskRun(rprt.TaskRunName)
		if err != nil {
			if !errors.IsNotFound(err) {
				return nil, xerrors.Errorf("error retrieving TaskRun %s: %w", rprt.TaskRunName, err)
			}
		}
		if taskRun != nil {
			rprt.TaskRun = taskRun
		}

		// Get all conditions that this pipelineTask will be using, if any
		if len(pt.Conditions) > 0 {
			rcc, err := resolveConditionChecks(&pt, pipelineRun.Status.TaskRuns, rprt.TaskRunName, getTaskRun, getCondition, getResource, providedResources)
			if err != nil {
				return nil, err
			}
			rprt.ResolvedConditionChecks = rcc
		}

		// Add this task to the state of the PipelineRun
		state = append(state, &rprt)
	}
	return state, nil
}

// getConditionCheckName should return a unique name for a `ConditionCheck` if one has not already been defined, and the existing one otherwise.
func getConditionCheckName(taskRunStatus map[string]*v1alpha1.PipelineRunTaskRunStatus, trName, conditionName string) string {
	trStatus, ok := taskRunStatus[trName]
	if ok && trStatus.ConditionChecks != nil {
		for k, v := range trStatus.ConditionChecks {
			// TODO(1022): Should  we allow multiple conditions of the same type?
			if conditionName == v.ConditionName {
				return k
			}
		}
	}
	return names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(fmt.Sprintf("%s-%s", trName, conditionName))
}

// getTaskRunName should return a unique name for a `TaskRun` if one has not already been defined, and the existing one otherwise.
func getTaskRunName(taskRunsStatus map[string]*v1alpha1.PipelineRunTaskRunStatus, ptName, prName string) string {
	for k, v := range taskRunsStatus {
		if v.PipelineTaskName == ptName {
			return k
		}
	}

	return names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(fmt.Sprintf("%s-%s", prName, ptName))
}

// GetPipelineConditionStatus will return the Condition that the PipelineRun prName should be
// updated with, based on the status of the TaskRuns in state.
func GetPipelineConditionStatus(pr *v1alpha1.PipelineRun, state PipelineRunState, logger *zap.SugaredLogger, dag *v1alpha1.DAG) *apis.Condition {
	// We have 4 different states here:
	// 1. Timed out -> Failed
	// 2. Any one TaskRun has failed - >Failed. This should change with #1020 and #1023
	// 3. All tasks are done or are skipped (i.e. condition check failed).-> Success
	// 4. A Task or Condition is running right now  or there are things left to run -> Running

	if pr.IsTimedOut() {
		return &apis.Condition{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionFalse,
			Reason:  ReasonTimedOut,
			Message: fmt.Sprintf("PipelineRun %q failed to finish within %q", pr.Name, pr.Spec.Timeout.String()),
		}
	}

	// A single failed task mean we fail the pipeline
	for _, rprt := range state {
		if rprt.IsFailure() { //IsDone ensures we have crossed the retry limit
			logger.Infof("TaskRun %s has failed, so PipelineRun %s has failed, retries done: %b", rprt.TaskRunName, pr.Name, len(rprt.TaskRun.Status.RetriesStatus))
			return &apis.Condition{
				Type:    apis.ConditionSucceeded,
				Status:  corev1.ConditionFalse,
				Reason:  ReasonFailed,
				Message: fmt.Sprintf("TaskRun %s has failed", rprt.TaskRun.Name),
			}
		}
	}

	allTasks := []string{}
	successOrSkipTasks := []string{}

	// Check to see if all tasks are success or skipped
	for _, rprt := range state {
		allTasks = append(allTasks, rprt.PipelineTask.Name)
		if rprt.IsSuccessful() || isSkipped(rprt, state.toMap(), dag) {
			successOrSkipTasks = append(successOrSkipTasks, rprt.PipelineTask.Name)
		}
	}

	if reflect.DeepEqual(allTasks, successOrSkipTasks) {
		logger.Infof("All TaskRuns have finished for PipelineRun %s so it has finished", pr.Name)
		return &apis.Condition{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionTrue,
			Reason:  ReasonSucceeded,
			Message: "All Tasks have completed executing",
		}
	}

	// Hasn't timed out; no taskrun failed yet; and not all tasks have finished....
	// Must keep running then....
	return &apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionUnknown,
		Reason:  ReasonRunning,
		Message: "Not all Tasks in the Pipeline have finished executing",
	}
}

// isSkipped returns true if a Task in a TaskRun will not be run either because
//  its Condition Checks failed or because one of the parent tasks's conditions failed
// Note that this means isSkipped returns false if a conditionCheck is in progress
func isSkipped(rprt *ResolvedPipelineRunTask, stateMap map[string]*ResolvedPipelineRunTask, d *v1alpha1.DAG) bool {
	// Taskrun not skipped if it already exists
	if rprt.TaskRun != nil {
		return false
	}

	// Check if conditionChecks have failed, if so task is skipped
	if len(rprt.ResolvedConditionChecks) > 0 {
		// isSkipped is only true iof
		if rprt.ResolvedConditionChecks.IsDone() && !rprt.ResolvedConditionChecks.IsSuccess() {
			return true
		}
	}

	// Recursively look at parent tasks to see if they have been skipped,
	// if any of the parents have been skipped, skip as well
	node := d.Nodes[rprt.PipelineTask.Name]
	for _, p := range node.Prev {
		skip := isSkipped(stateMap[p.Task.Name], stateMap, d)
		if skip {
			return true
		}
	}
	return false
}

func findReferencedTask(pb string, state []*ResolvedPipelineRunTask) *ResolvedPipelineRunTask {
	for _, rprtRef := range state {
		if rprtRef.PipelineTask.Name == pb {
			return rprtRef
		}
	}
	return nil
}

// ValidateFrom will look at any `from` clauses in the resolved PipelineRun state
// and validate it: the `from` must specify an input of the current `Task`. The `PipelineTask`
// it corresponds to must actually exist in the `Pipeline`. The `PipelineResource` that is bound to the input
// must be the same `PipelineResource` that was bound to the output of the previous `Task`. If the state is
// not valid, it will return an error.
func ValidateFrom(state PipelineRunState) error {
	for _, rprt := range state {
		if rprt.PipelineTask.Resources != nil {
			for _, dep := range rprt.PipelineTask.Resources.Inputs {
				inputBinding := rprt.ResolvedTaskResources.Inputs[dep.Name]
				for _, pb := range dep.From {
					if pb == rprt.PipelineTask.Name {
						return xerrors.Errorf("PipelineTask %s is trying to depend on a PipelineResource from itself", pb)
					}
					depTask := findReferencedTask(pb, state)
					if depTask == nil {
						return xerrors.Errorf("pipelineTask %s is trying to depend on previous Task %q but it does not exist", rprt.PipelineTask.Name, pb)
					}

					sameBindingExists := false
					for _, output := range depTask.ResolvedTaskResources.Outputs {
						if output.Name == inputBinding.Name {
							sameBindingExists = true
						}
					}
					if !sameBindingExists {
						return xerrors.Errorf("from is ambiguous: input %q for PipelineTask %q is bound to %q but no outputs in PipelineTask %q are bound to same resource",
							dep.Name, rprt.PipelineTask.Name, inputBinding.Name, depTask.PipelineTask.Name)
					}
				}
			}
		}
	}

	return nil
}

func resolveConditionChecks(pt *v1alpha1.PipelineTask,
	taskRunStatus map[string]*v1alpha1.PipelineRunTaskRunStatus,
	taskRunName string, getTaskRun resources.GetTaskRun, getCondition GetCondition,
	getResource resources.GetResource, providedResources map[string]v1alpha1.PipelineResourceRef) ([]*ResolvedConditionCheck, error) {
	rccs := []*ResolvedConditionCheck{}
	for _, ptc := range pt.Conditions {
		cName := ptc.ConditionRef
		c, err := getCondition(cName)
		if err != nil {
			return nil, &ConditionNotFoundError{
				Name: cName,
				Msg:  err.Error(),
			}
		}
		conditionCheckName := getConditionCheckName(taskRunStatus, taskRunName, cName)
		cctr, err := getTaskRun(conditionCheckName)
		if err != nil {
			if !errors.IsNotFound(err) {
				return nil, xerrors.Errorf("error retrieving ConditionCheck %s for taskRun name %s : %w", conditionCheckName, taskRunName, err)
			}
		}

		rcc := ResolvedConditionCheck{
			Condition:             c,
			ConditionCheckName:    conditionCheckName,
			ConditionCheck:        v1alpha1.NewConditionCheck(cctr),
			PipelineTaskCondition: &ptc,
		}

		if len(ptc.Resources) > 0 {
			r, err := resolveConditionResources(ptc.Resources, getResource, providedResources)
			if err != nil {
				return nil, xerrors.Errorf("cloud not resolve resources for condition %s in pipeline task %s: %w", cName, pt.Name, err)
			}
			rcc.ResolvedResources = r
		}

		rccs = append(rccs, &rcc)
	}
	return rccs, nil
}

func resolveConditionResources(prc []v1alpha1.PipelineConditionResource,
	getResource resources.GetResource,
	providedResources map[string]v1alpha1.PipelineResourceRef,
) (map[string]*v1alpha1.PipelineResource, error) {
	rr := make(map[string]*v1alpha1.PipelineResource)
	for _, r := range prc {
		// First get a ref to actual resource name from its bound name
		resourceRef, ok := providedResources[r.Resource]
		if !ok {
			return nil, xerrors.Errorf("resource %s not present in declared resources", r.Resource)
		}

		// Next, fetch the actual resource definition
		gotResource, err := getResource(resourceRef.Name)
		if err != nil {
			return nil, xerrors.Errorf("could not retrieve resource %s: %w", r.Name, err)
		}

		// Finally add it to the resolved resources map
		rr[r.Name] = gotResource
	}
	return rr, nil
}
