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

	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
)

// Label added to TaskRuns identifying the associated pipeline Task
const PipelineTaskLabel = "tekton.dev/pipelineTask"

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
	GetPullSecrets() []string
	IsDone() bool
	IsSuccessful() bool
	SupportsTaskRunArtifact() bool
	SupportsPipelineRunArtifact() bool
	SupportsOCIArtifact() bool
	GetRemoteProvenance() *v1.Provenance
	IsRemote() bool
}

func NewTektonObject(i interface{}) (TektonObject, error) {
	switch o := i.(type) {
	case *v1.PipelineRun:
		return NewPipelineRunObjectV1(o), nil
	case *v1.TaskRun:
		return NewTaskRunObjectV1(o), nil
	case *v1beta1.PipelineRun: //nolint:staticcheck
		return NewPipelineRunObjectV1Beta1(o), nil
	case *v1beta1.TaskRun: //nolint:staticcheck
		return NewTaskRunObjectV1Beta1(o), nil
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
	_, err := clientSet.TektonV1().TaskRuns(tro.Namespace).Patch(
		ctx, tro.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
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

// Get the imgPullSecrets from the pod template
func (tro *TaskRunObjectV1) GetPullSecrets() []string {
	return getPodPullSecrets(tro.Spec.PodTemplate)
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
	_, err := clientSet.TektonV1().PipelineRuns(pro.Namespace).Patch(
		ctx, pro.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
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
func (pro *PipelineRunObjectV1) GetTaskRuns() []*v1.TaskRun { //nolint:staticcheck
	return pro.taskRuns
}

// Get the associated TaskRun via the Task name
func (pro *PipelineRunObjectV1) GetTaskRunFromTask(taskName string) *TaskRunObjectV1 {
	for _, tr := range pro.taskRuns {
		val, ok := tr.Labels[PipelineTaskLabel]
		if ok && val == taskName {
			return NewTaskRunObjectV1(tr)
		}
	}
	return nil
}

// Get the imgPullSecrets from the pod template
func (pro *PipelineRunObjectV1) GetPullSecrets() []string {
	return getPodPullSecrets(pro.Spec.TaskRunTemplate.PodTemplate)
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

// Get the imgPullSecrets from a pod template, if they exist
func getPodPullSecrets(podTemplate *pod.Template) []string {
	imgPullSecrets := []string{}
	if podTemplate != nil {
		for _, secret := range podTemplate.ImagePullSecrets {
			imgPullSecrets = append(imgPullSecrets, secret.Name)
		}
	}
	return imgPullSecrets
}

// PipelineRunObjectV1Beta1 extends v1.PipelineRun with additional functions.
type PipelineRunObjectV1Beta1 struct {
	// The base PipelineRun
	*v1beta1.PipelineRun
	// taskRuns that were apart of this PipelineRun
	taskRuns []*v1beta1.TaskRun //nolint:staticcheck
}

var _ TektonObject = &PipelineRunObjectV1Beta1{}

func NewPipelineRunObjectV1Beta1(pr *v1beta1.PipelineRun) *PipelineRunObjectV1Beta1 { //nolint:staticcheck
	return &PipelineRunObjectV1Beta1{
		PipelineRun: pr,
	}
}

// Get the PipelineRun GroupVersionKind
func (pro *PipelineRunObjectV1Beta1) GetGVK() string {
	return fmt.Sprintf("%s/%s", pro.GetGroupVersionKind().GroupVersion().String(), pro.GetGroupVersionKind().Kind)
}

func (pro *PipelineRunObjectV1Beta1) GetKindName() string {
	return strings.ToLower(pro.GetGroupVersionKind().Kind)
}

// Request the current annotations on the PipelineRun object
func (pro *PipelineRunObjectV1Beta1) GetLatestAnnotations(ctx context.Context, clientSet versioned.Interface) (map[string]string, error) {
	pr, err := clientSet.TektonV1beta1().PipelineRuns(pro.Namespace).Get(ctx, pro.Name, metav1.GetOptions{})
	return pr.Annotations, err
}

// Get the base PipelineRun
func (pro *PipelineRunObjectV1Beta1) GetObject() interface{} {
	return pro.PipelineRun
}

// Patch the original PipelineRun object
func (pro *PipelineRunObjectV1Beta1) Patch(ctx context.Context, clientSet versioned.Interface, patchBytes []byte) error {
	_, err := clientSet.TektonV1beta1().PipelineRuns(pro.Namespace).Patch(
		ctx, pro.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func (pro *PipelineRunObjectV1Beta1) GetProvenance() *v1.Provenance {
	var rs *v1.RefSource
	if pro.Status.Provenance != nil && pro.Status.Provenance.RefSource != nil {
		rs = &v1.RefSource{
			URI:        pro.Status.Provenance.RefSource.URI,
			Digest:     pro.Status.Provenance.RefSource.Digest,
			EntryPoint: pro.Status.Provenance.RefSource.EntryPoint,
		}
	} else if pro.Status.Provenance != nil && pro.Status.Provenance.ConfigSource != nil { //nolint:staticcheck
		rs = &v1.RefSource{
			URI:        pro.Status.Provenance.ConfigSource.URI,        //nolint:staticcheck
			Digest:     pro.Status.Provenance.ConfigSource.Digest,     //nolint:staticcheck
			EntryPoint: pro.Status.Provenance.ConfigSource.EntryPoint, //nolint:staticcheck
		}
	}

	var ff *config.FeatureFlags
	if pro.Status.Provenance != nil {
		ff = pro.Status.Provenance.FeatureFlags
	}

	return &v1.Provenance{
		RefSource:    rs,
		FeatureFlags: ff,
	}
}

// Get the resolved Pipelinerun results
func (pro *PipelineRunObjectV1Beta1) GetResults() []Result {
	res := []Result{}
	for _, key := range pro.Status.PipelineResults {
		res = append(res, Result{
			Name: key.Name,
			Value: v1.ParamValue{
				ArrayVal:  key.Value.ArrayVal,
				ObjectVal: key.Value.ObjectVal,
				StringVal: key.Value.StringVal,
				Type:      v1.ParamType(key.Value.Type),
			},
		})
	}
	return res
}

// Get the ServiceAccount declared in the PipelineRun
func (pro *PipelineRunObjectV1Beta1) GetServiceAccountName() string {
	return pro.Spec.ServiceAccountName
}

// Get the ServiceAccount declared in the PipelineRun
func (pro *PipelineRunObjectV1Beta1) IsSuccessful() bool {
	return pro.Status.GetCondition(apis.ConditionSucceeded).IsTrue()
}

// Append TaskRuns to this PipelineRun
func (pro *PipelineRunObjectV1Beta1) AppendTaskRun(tr *v1beta1.TaskRun) { //nolint:staticcheck
	pro.taskRuns = append(pro.taskRuns, tr)
}

// Get the associated TaskRun via the Task name
func (pro *PipelineRunObjectV1Beta1) GetTaskRunFromTask(taskName string) *TaskRunObjectV1Beta1 {
	for _, tr := range pro.taskRuns {
		val, ok := tr.Labels[PipelineTaskLabel]
		if ok && val == taskName {
			return NewTaskRunObjectV1Beta1(tr)
		}
	}
	return nil
}

// Get the imgPullSecrets from the pod template
func (pro *PipelineRunObjectV1Beta1) GetPullSecrets() []string {
	return getPodPullSecrets(pro.Spec.PodTemplate)
}

func (pro *PipelineRunObjectV1Beta1) SupportsTaskRunArtifact() bool {
	return false
}

func (pro *PipelineRunObjectV1Beta1) SupportsPipelineRunArtifact() bool {
	return true
}

func (pro *PipelineRunObjectV1Beta1) SupportsOCIArtifact() bool {
	return false
}

func (pro *PipelineRunObjectV1Beta1) GetRemoteProvenance() *v1.Provenance {
	if p := pro.Status.Provenance; p != nil && p.RefSource != nil && pro.IsRemote() {
		return &v1.Provenance{
			RefSource:    pro.GetProvenance().RefSource,
			FeatureFlags: pro.GetProvenance().FeatureFlags,
		}
	}
	return nil
}

func (pro *PipelineRunObjectV1Beta1) IsRemote() bool {
	isRemotePipeline := false
	if pro.Spec.PipelineRef != nil {
		if pro.Spec.PipelineRef.Resolver != "" && pro.Spec.PipelineRef.Resolver != "Cluster" {
			isRemotePipeline = true
		}
	}
	return isRemotePipeline
}

// TaskRunObjectV1Beta1 extends v1beta1.TaskRun with additional functions.
type TaskRunObjectV1Beta1 struct {
	*v1beta1.TaskRun
}

var _ TektonObject = &TaskRunObjectV1Beta1{}

func NewTaskRunObjectV1Beta1(tr *v1beta1.TaskRun) *TaskRunObjectV1Beta1 { //nolint:staticcheck
	return &TaskRunObjectV1Beta1{
		tr,
	}
}

// Get the TaskRun GroupVersionKind
func (tro *TaskRunObjectV1Beta1) GetGVK() string {
	return fmt.Sprintf("%s/%s", tro.GetGroupVersionKind().GroupVersion().String(), tro.GetGroupVersionKind().Kind)
}

func (tro *TaskRunObjectV1Beta1) GetKindName() string {
	return strings.ToLower(tro.GetGroupVersionKind().Kind)
}

func (tro *TaskRunObjectV1Beta1) GetProvenance() *v1.Provenance {
	var rs *v1.RefSource
	if tro.Status.Provenance != nil && tro.Status.Provenance.RefSource != nil {
		rs = &v1.RefSource{
			URI:        tro.Status.Provenance.RefSource.URI,
			Digest:     tro.Status.Provenance.RefSource.Digest,
			EntryPoint: tro.Status.Provenance.RefSource.EntryPoint,
		}
	} else if tro.Status.Provenance != nil && tro.Status.Provenance.ConfigSource != nil { //nolint:staticcheck
		rs = &v1.RefSource{
			URI:        tro.Status.Provenance.ConfigSource.URI,        //nolint:staticcheck
			Digest:     tro.Status.Provenance.ConfigSource.Digest,     //nolint:staticcheck
			EntryPoint: tro.Status.Provenance.ConfigSource.EntryPoint, //nolint:staticcheck
		}
	}

	var ff *config.FeatureFlags
	if tro.Status.Provenance != nil {
		ff = tro.Status.Provenance.FeatureFlags
	}

	return &v1.Provenance{
		RefSource:    rs,
		FeatureFlags: ff,
	}
}

// Get the latest annotations on the TaskRun
func (tro *TaskRunObjectV1Beta1) GetLatestAnnotations(ctx context.Context, clientSet versioned.Interface) (map[string]string, error) {
	tr, err := clientSet.TektonV1beta1().TaskRuns(tro.Namespace).Get(ctx, tro.Name, metav1.GetOptions{})
	return tr.Annotations, err
}

// Get the base TaskRun object
func (tro *TaskRunObjectV1Beta1) GetObject() interface{} {
	return tro.TaskRun
}

// Patch the original TaskRun object
func (tro *TaskRunObjectV1Beta1) Patch(ctx context.Context, clientSet versioned.Interface, patchBytes []byte) error {
	_, err := clientSet.TektonV1beta1().TaskRuns(tro.Namespace).Patch(
		ctx, tro.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	return err
}

// Get the TaskRun results
func (tro *TaskRunObjectV1Beta1) GetResults() []Result {
	res := []Result{}
	for _, key := range tro.Status.TaskRunResults {
		res = append(res, Result{
			Name: key.Name,
			Value: v1.ParamValue{
				ArrayVal:  key.Value.ArrayVal,
				ObjectVal: key.Value.ObjectVal,
				StringVal: key.Value.StringVal,
				Type:      v1.ParamType(key.Value.Type),
			},
		})
	}
	return res
}

func (tro *TaskRunObjectV1Beta1) GetStepImages() []string {
	images := []string{}
	for _, stepState := range tro.Status.Steps {
		images = append(images, stepState.ImageID)
	}
	return images
}

func (tro *TaskRunObjectV1Beta1) GetSidecarImages() []string {
	images := []string{}
	for _, sidecarState := range tro.Status.Sidecars {
		images = append(images, sidecarState.ImageID)
	}
	return images
}

// Get the ServiceAccount declared in the TaskRun
func (tro *TaskRunObjectV1Beta1) GetServiceAccountName() string {
	return tro.Spec.ServiceAccountName
}

// Get the imgPullSecrets from the pod template
func (tro *TaskRunObjectV1Beta1) GetPullSecrets() []string {
	return getPodPullSecrets(tro.Spec.PodTemplate)
}

func (tro *TaskRunObjectV1Beta1) SupportsTaskRunArtifact() bool {
	return true
}

func (tro *TaskRunObjectV1Beta1) SupportsPipelineRunArtifact() bool {
	return false
}

func (tro *TaskRunObjectV1Beta1) SupportsOCIArtifact() bool {
	return true
}

func (tro *TaskRunObjectV1Beta1) GetRemoteProvenance() *v1.Provenance {
	if t := tro.Status.Provenance; t != nil && t.RefSource != nil && tro.IsRemote() {
		return &v1.Provenance{
			RefSource:    tro.GetProvenance().RefSource,
			FeatureFlags: tro.GetProvenance().FeatureFlags,
		}
	}
	return nil
}

func (tro *TaskRunObjectV1Beta1) IsRemote() bool {
	isRemoteTask := false
	if tro.Spec.TaskRef != nil {
		if tro.Spec.TaskRef.Resolver != "" && tro.Spec.TaskRef.Resolver != "Cluster" {
			isRemoteTask = true
		}
	}
	return isRemoteTask
}
