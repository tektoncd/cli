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
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/patch"
	versioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"knative.dev/pkg/logging"
)

const (
	// ChainsAnnotation is the standard annotation to indicate a TR has been signed.
	ChainsAnnotation             = "chains.tekton.dev/signed"
	RetryAnnotation              = "chains.tekton.dev/retries"
	ChainsTransparencyAnnotation = "chains.tekton.dev/transparency"
	MaxRetries                   = 3
)

// Reconciled determines whether a Tekton object has already been reconciled.
// It first inspects the state of the given TektonObject. If that indicates it
// has not been reconciled, then Reconciled fetches the latest version of the
// TektonObject from the cluster and inspects that version as well. This aims
// to avoid creating multiple attestations due to a stale cached TektonObject.
func Reconciled(ctx context.Context, client versioned.Interface, obj objects.TektonObject) bool {
	if reconciledFromAnnotations(obj.GetAnnotations()) {
		return true
	}

	logger := logging.FromContext(ctx)
	annotations, err := obj.GetLatestAnnotations(ctx, client)
	if err != nil {
		logger.Warnf("Ignoring error when fetching latest annotations: %s", err)
		return false
	}
	return reconciledFromAnnotations(annotations)
}

func reconciledFromAnnotations(annotations map[string]string) bool {
	val, ok := annotations[ChainsAnnotation]
	if !ok {
		return false
	}
	return val == "true" || val == "failed"
}

// MarkSigned marks a Tekton object as signed.
func MarkSigned(ctx context.Context, obj objects.TektonObject, ps versioned.Interface, annotations map[string]string) error {
	if _, ok := obj.GetAnnotations()[ChainsAnnotation]; ok {
		return nil
	}
	return AddAnnotation(ctx, obj, ps, ChainsAnnotation, "true", annotations)
}

func MarkFailed(ctx context.Context, obj objects.TektonObject, ps versioned.Interface, annotations map[string]string) error {
	return AddAnnotation(ctx, obj, ps, ChainsAnnotation, "failed", annotations)
}

func RetryAvailable(obj objects.TektonObject) bool {
	ann, ok := obj.GetAnnotations()[RetryAnnotation]
	if !ok {
		return true
	}
	val, err := strconv.Atoi(ann)
	if err != nil {
		return false
	}
	return val < MaxRetries
}

func AddRetry(ctx context.Context, obj objects.TektonObject, ps versioned.Interface, annotations map[string]string) error {
	ann := obj.GetAnnotations()[RetryAnnotation]
	if ann == "" {
		return AddAnnotation(ctx, obj, ps, RetryAnnotation, "0", annotations)
	}
	val, err := strconv.Atoi(ann)
	if err != nil {
		return errors.Wrap(err, "adding retry")
	}
	return AddAnnotation(ctx, obj, ps, RetryAnnotation, fmt.Sprintf("%d", val+1), annotations)
}

func AddAnnotation(ctx context.Context, obj objects.TektonObject, ps versioned.Interface, key, value string, annotations map[string]string) error {
	// Use patch instead of update to help prevent race conditions.
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[key] = value
	patchBytes, err := patch.GetAnnotationsPatch(annotations)
	if err != nil {
		return err
	}
	err = obj.Patch(ctx, ps, patchBytes)
	if err != nil {
		return err
	}
	return nil
}
