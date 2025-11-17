/*
Copyright 2022 The Tekton Authors
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

package objects

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/ptr"
)

// Label added to TaskRuns identifying the associated pipeline Task
const PipelineTaskLabel = "tekton.dev/pipelineTask"

// patchOptions contains the default patch options
var patchOptions = metav1.PatchOptions{
	FieldManager: "tekton-chains-controller",
	Force:        ptr.Bool(false),
}

// Object is used as a base object of all Kubernetes objects
// ref: https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.4/pkg/client#Object
type Object interface {
	// Metadata associated to all Kubernetes objects
	metav1.Object
	// Runtime identifying data
	runtime.Object
}

// Result is a generic key value store containing the results
// of Tekton operations. (eg. PipelineRun and TaskRun results)
type Result struct {
	Name  string
	Type  v1.ResultsType
	Value v1.ParamValue
}

// Tekton object is an extended Kubernetes object with operations specific
// to Tekton objects.
type TektonObject interface {
	Object
	GetGVK() string
	GetKindName() string
	GetObject() interface{}
	GetLatestAnnotations(ctx context.Context, clientSet versioned.Interface) (map[string]string, error)
	Patch(ctx context.Context, clientSet versioned.Interface, patchBytes []byte) error
	GetResults() []Result
	GetProvenance() *v1.Provenance
	GetServiceAccountName() string
	IsDone() bool
	IsSuccessful() bool
	SupportsTaskRunArtifact() bool
	SupportsPipelineRunArtifact() bool
	SupportsOCIArtifact() bool
	GetRemoteProvenance() *v1.Provenance
	IsRemote() bool
	GetStartTime() *time.Time
	GetCompletitionTime() *time.Time
}

func NewTektonObject(i interface{}) (TektonObject, error) {
	switch o := i.(type) {
	case *v1.PipelineRun:
		return NewPipelineRunObjectV1(o), nil
	case *v1.TaskRun:
		return NewTaskRunObjectV1(o), nil
	default:
		return nil, errors.New("unrecognized type when attempting to create tekton object")
	}
}

// TaskRunObjectV1 extends v1.TaskRun with additional functions.
type TaskRunObjectV1 struct {
	*v1.TaskRun
}

var _ TektonObject = &TaskRunObjectV1{}

func NewTaskRunObjectV1(tr *v1.TaskRun) *TaskRunObjectV1 {
	return &TaskRunObjectV1{
		tr,
	}
}

// Get the TaskRun GroupVersionKind
func (tro *TaskRunObjectV1) GetGVK() string {
	return fmt.Sprintf("%s/%s", tro.GetGroupVersionKind().GroupVersion().String(), tro.GetGroupVersionKind().Kind)
}

func (tro *TaskRunObjectV1) GetKindName() string {
	return strings.ToLower(tro.GetGroupVersionKind().Kind)
}

func (tro *TaskRunObjectV1) GetProvenance() *v1.Provenance {
	return tro.Status.Provenance
}

// Get the latest annotations on the TaskRun
func (tro *TaskRunObjectV1) GetLatestAnnotations(ctx context.Context, clientSet versioned.Interface) (map[string]string, error) {
	tr, err := clientSet.TektonV1().TaskRuns(tro.Namespace).Get(ctx, tro.Name, metav1.GetOptions{})
	return tr.Annotations, err
}

// Get the base TaskRun object
func (tro *TaskRunObjectV1) GetObject() interface{} {
	return tro.TaskRun
}

// Patch the original TaskRun object
func (tro *TaskRunObjectV1) Patch(ctx context.Context, clientSet versioned.Interface, patchBytes []byte) error {
	logger := logging.FromContext(ctx)
	_, err := clientSet.TektonV1().TaskRuns(tro.Namespace).Patch(
		ctx, tro.Name, types.ApplyPatchType, patchBytes, patchOptions)
	if apierrors.IsConflict(err) {
		// Since we only update the list of annotations we manage, there shouldn't be any conflicts unless
		// another controller/client is updating our annotations. We log the issue and force patch.
		logger.Warnf("failed to patch object %s/%s due to Server-Side Apply patch conflict, using force patch.", tro.Namespace, tro.Name)
		// use a copy to avoid changing the global var
		patchOptionsForce := patchOptions
		patchOptionsForce.Force = ptr.Bool(true)
		_, err = clientSet.TektonV1().TaskRuns(tro.Namespace).Patch(
			ctx, tro.Name, types.ApplyPatchType, patchBytes, patchOptionsForce)
	}
	return err
}

// Get the TaskRun results
func (tro *TaskRunObjectV1) GetResults() []Result {
	res := []Result{}
	for _, key := range tro.Status.Results {
		res = append(res, Result{
			Name:  key.Name,
			Value: key.Value,
		})
	}
	return res
}

// GetStepResults returns all the results from associated StepActions.
func (tro *TaskRunObjectV1) GetStepResults() []Result {
	res := []Result{}
	for _, s := range tro.Status.Steps {
		for _, r := range s.Results {
			res = append(res, Result{
				Name:  r.Name,
				Value: r.Value,
			})
		}
	}
	return res
}

func (tro *TaskRunObjectV1) GetStepImages() []string {
	images := []string{}
	for _, stepState := range tro.Status.Steps {
		images = append(images, stepState.ImageID)
	}
	return images
}

func (tro *TaskRunObjectV1) GetSidecarImages() []string {
	images := []string{}
	for _, sidecarState := range tro.Status.Sidecars {
		images = append(images, sidecarState.ImageID)
	}
	return images
}

// Get the ServiceAccount declared in the TaskRun
func (tro *TaskRunObjectV1) GetServiceAccountName() string {
	return tro.Spec.ServiceAccountName
}

func (tro *TaskRunObjectV1) SupportsTaskRunArtifact() bool {
	return true
}

func (tro *TaskRunObjectV1) SupportsPipelineRunArtifact() bool {
	return false
}

func (tro *TaskRunObjectV1) SupportsOCIArtifact() bool {
	return true
}

func (tro *TaskRunObjectV1) GetRemoteProvenance() *v1.Provenance {
	if t := tro.Status.Provenance; t != nil && t.RefSource != nil && tro.IsRemote() {
		return tro.Status.Provenance
	}
	return nil
}

func (tro *TaskRunObjectV1) IsRemote() bool {
	isRemoteTask := false
	if tro.Spec.TaskRef != nil {
		if tro.Spec.TaskRef.Resolver != "" && tro.Spec.TaskRef.Resolver != "Cluster" {
			isRemoteTask = true
		}
	}
	return isRemoteTask
}

// GetStartTime returns the time when the TaskRun started.
func (tro *TaskRunObjectV1) GetStartTime() *time.Time {
	var utc *time.Time
	if tro.Status.StartTime != nil {
		val := tro.Status.StartTime.Time.UTC()
		utc = &val
	}
	return utc
}

// GetCompletitionTime returns the time when the TaskRun finished.
func (tro *TaskRunObjectV1) GetCompletitionTime() *time.Time {
	var utc *time.Time
	if tro.Status.CompletionTime != nil {
		val := tro.Status.CompletionTime.Time.UTC()
		utc = &val
	}
	return utc
}

// PipelineRunObjectV1 extends v1.PipelineRun with additional functions.
type PipelineRunObjectV1 struct {
	// The base PipelineRun
	*v1.PipelineRun
	// taskRuns that were apart of this PipelineRun
	taskRuns []*v1.TaskRun
}

var _ TektonObject = &PipelineRunObjectV1{}

func NewPipelineRunObjectV1(pr *v1.PipelineRun) *PipelineRunObjectV1 {
	return &PipelineRunObjectV1{
		PipelineRun: pr,
	}
}

// Get the PipelineRun GroupVersionKind
func (pro *PipelineRunObjectV1) GetGVK() string {
	return fmt.Sprintf("%s/%s", pro.GetGroupVersionKind().GroupVersion().String(), pro.GetGroupVersionKind().Kind)
}

func (pro *PipelineRunObjectV1) GetKindName() string {
	return strings.ToLower(pro.GetGroupVersionKind().Kind)
}

// Request the current annotations on the PipelineRun object
func (pro *PipelineRunObjectV1) GetLatestAnnotations(ctx context.Context, clientSet versioned.Interface) (map[string]string, error) {
	pr, err := clientSet.TektonV1().PipelineRuns(pro.Namespace).Get(ctx, pro.Name, metav1.GetOptions{})
	return pr.Annotations, err
}

// Get the base PipelineRun
func (pro *PipelineRunObjectV1) GetObject() interface{} {
	return pro.PipelineRun
}

// Patch the original PipelineRun object
func (pro *PipelineRunObjectV1) Patch(ctx context.Context, clientSet versioned.Interface, patchBytes []byte) error {
	logger := logging.FromContext(ctx)
	_, err := clientSet.TektonV1().PipelineRuns(pro.Namespace).Patch(
		ctx, pro.Name, types.ApplyPatchType, patchBytes, patchOptions)
	if apierrors.IsConflict(err) {
		// Since we only update the list of annotations we manage, there shouldn't be any conflicts unless
		// another controller/client is updating our annotations. We log the issue and force patch.
		logger.Warnf("failed to patch object %s/%s due to Server-Side Apply patch conflict, using force patch.", pro.Namespace, pro.Name)
		// use a copy to avoid changing the global var
		patchOptionsForce := patchOptions
		patchOptionsForce.Force = ptr.Bool(true)
		_, err = clientSet.TektonV1().PipelineRuns(pro.Namespace).Patch(
			ctx, pro.Name, types.ApplyPatchType, patchBytes, patchOptionsForce)
	}
	return err
}

func (pro *PipelineRunObjectV1) GetProvenance() *v1.Provenance {
	return pro.Status.Provenance
}

// Get the resolved Pipelinerun results
func (pro *PipelineRunObjectV1) GetResults() []Result {
	res := []Result{}
	for _, key := range pro.Status.Results {
		res = append(res, Result{
			Name:  key.Name,
			Value: key.Value,
		})
	}
	return res
}

// Get the ServiceAccount declared in the PipelineRun
func (pro *PipelineRunObjectV1) GetServiceAccountName() string {
	return pro.Spec.TaskRunTemplate.ServiceAccountName
}

// Get the ServiceAccount declared in the PipelineRun
func (pro *PipelineRunObjectV1) IsSuccessful() bool {
	return pro.Status.GetCondition(apis.ConditionSucceeded).IsTrue()
}

// Append TaskRuns to this PipelineRun
func (pro *PipelineRunObjectV1) AppendTaskRun(tr *v1.TaskRun) {
	pro.taskRuns = append(pro.taskRuns, tr)
}

// Append TaskRuns to this PipelineRun
func (pro *PipelineRunObjectV1) GetTaskRuns() []*v1.TaskRun {
	return pro.taskRuns
}

// Get the associated TaskRun via the Task name
func (pro *PipelineRunObjectV1) GetTaskRunsFromTask(taskName string) []*TaskRunObjectV1 {
	var taskRuns []*TaskRunObjectV1
	for _, tr := range pro.taskRuns {
		val, ok := tr.Labels[PipelineTaskLabel]
		if ok && val == taskName {
			taskRuns = append(taskRuns, NewTaskRunObjectV1(tr))
		}
	}
	return taskRuns
}

func (pro *PipelineRunObjectV1) SupportsTaskRunArtifact() bool {
	return false
}

func (pro *PipelineRunObjectV1) SupportsPipelineRunArtifact() bool {
	return true
}

func (pro *PipelineRunObjectV1) SupportsOCIArtifact() bool {
	return false
}

func (pro *PipelineRunObjectV1) GetRemoteProvenance() *v1.Provenance {
	if p := pro.Status.Provenance; p != nil && p.RefSource != nil && pro.IsRemote() {
		return pro.Status.Provenance
	}
	return nil
}

func (pro *PipelineRunObjectV1) IsRemote() bool {
	isRemotePipeline := false
	if pro.Spec.PipelineRef != nil {
		if pro.Spec.PipelineRef.Resolver != "" && pro.Spec.PipelineRef.Resolver != "Cluster" {
			isRemotePipeline = true
		}
	}
	return isRemotePipeline
}

// GetStartTime returns the time when the PipelineRun started.
func (pro *PipelineRunObjectV1) GetStartTime() *time.Time {
	var utc *time.Time
	if pro.Status.StartTime != nil {
		val := pro.Status.StartTime.Time.UTC()
		utc = &val
	}
	return utc
}

// GetCompletitionTime returns the time when the PipelineRun finished.
func (pro *PipelineRunObjectV1) GetCompletitionTime() *time.Time {
	var utc *time.Time
	if pro.Status.CompletionTime != nil {
		val := pro.Status.CompletionTime.Time.UTC()
		utc = &val
	}
	return utc
}

// GetExecutedTasks returns the tasks that were executed during the pipeline run.
func (pro *PipelineRunObjectV1) GetExecutedTasks() (tro []*TaskRunObjectV1) {
	pSpec := pro.Status.PipelineSpec
	if pSpec == nil {
		return
	}
	tasks := pSpec.Tasks
	tasks = append(tasks, pSpec.Finally...)
	for _, task := range tasks {
		taskRuns := pro.GetTaskRunsFromTask(task.Name)
		if len(taskRuns) == 0 {
			continue
		}
		for _, tr := range taskRuns {
			if tr == nil || tr.Status.CompletionTime == nil {
				continue
			}

			tro = append(tro, tr)
		}
	}

	return
}
