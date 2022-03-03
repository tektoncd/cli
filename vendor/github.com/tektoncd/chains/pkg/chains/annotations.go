/*
Copyright 2020 The Tekton Authors
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

package chains

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/tektoncd/chains/pkg/patch"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	versioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// ChainsAnnotation is the standard annotation to indicate a TR has been signed.
	ChainsAnnotation             = "chains.tekton.dev/signed"
	RetryAnnotation              = "chains.tekton.dev/retries"
	ChainsTransparencyAnnotation = "chains.tekton.dev/transparency"
	MaxRetries                   = 3
)

// Reconciled determines whether a TaskRun has already passed through the reconcile loops, up to 3x
func Reconciled(tr *v1beta1.TaskRun) bool {
	val, ok := tr.ObjectMeta.Annotations[ChainsAnnotation]
	if !ok {
		return false
	}
	return val == "true" || val == "failed"
}

// MarkSigned marks a TaskRun as signed.
func MarkSigned(tr *v1beta1.TaskRun, ps versioned.Interface, annotations map[string]string) error {
	if _, ok := tr.Annotations[ChainsAnnotation]; ok {
		return nil
	}
	return AddAnnotation(tr, ps, ChainsAnnotation, "true", annotations)
}

func MarkFailed(tr *v1beta1.TaskRun, ps versioned.Interface, annotations map[string]string) error {
	return AddAnnotation(tr, ps, ChainsAnnotation, "failed", annotations)
}

func RetryAvailable(tr *v1beta1.TaskRun) bool {
	retries, ok := tr.Annotations[RetryAnnotation]
	if !ok {
		return true
	}
	val, err := strconv.Atoi(retries)
	if err != nil {
		return false
	}
	return val < MaxRetries
}

func AddRetry(tr *v1beta1.TaskRun, ps versioned.Interface, annotations map[string]string) error {
	retries := tr.Annotations[RetryAnnotation]
	if retries == "" {
		return AddAnnotation(tr, ps, RetryAnnotation, "0", annotations)
	}
	val, err := strconv.Atoi(retries)
	if err != nil {
		return errors.Wrap(err, "adding retry")
	}
	return AddAnnotation(tr, ps, RetryAnnotation, fmt.Sprintf("%d", val+1), annotations)
}

func AddAnnotation(tr *v1beta1.TaskRun, ps versioned.Interface, key, value string, annotations map[string]string) error {
	// Use patch instead of update to help prevent race conditions.
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[key] = value
	patchBytes, err := patch.GetAnnotationsPatch(annotations)
	if err != nil {
		return err
	}
	if _, err := ps.TektonV1beta1().TaskRuns(tr.Namespace).Patch(
		context.TODO(), tr.Name, types.MergePatchType, patchBytes, v1.PatchOptions{}); err != nil {
		return err
	}
	return nil
}
