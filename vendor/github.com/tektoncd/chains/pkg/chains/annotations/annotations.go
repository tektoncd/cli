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

package annotations

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/tektoncd/chains/pkg/chains/objects"
	versioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"knative.dev/pkg/logging"
)

//nolint:revive,exported
const (
	// ChainsAnnotationPrefix is the prefix for all Chains annotations
	ChainsAnnotationPrefix       = "chains.tekton.dev/"
	ChainsAnnotation             = ChainsAnnotationPrefix + "signed"
	RetryAnnotation              = ChainsAnnotationPrefix + "retries"
	ChainsTransparencyAnnotation = ChainsAnnotationPrefix + "transparency"
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

// mergeAnnotations creates a new map with existing annotations plus a new key-value pair
func mergeAnnotations(annotations map[string]string, key, value string) map[string]string {
	merged := make(map[string]string)
	for k, v := range annotations {
		merged[k] = v
	}
	merged[key] = value
	return merged
}

// MarkSigned marks a Tekton object as signed.
func MarkSigned(ctx context.Context, obj objects.TektonObject, ps versioned.Interface, annotations map[string]string) error {
	if _, ok := obj.GetAnnotations()[ChainsAnnotation]; ok {
		// Object is already signed, but we may still need to apply additional annotations
		if len(annotations) > 0 {
			return AddAnnotations(ctx, obj, ps, mergeAnnotations(annotations, ChainsAnnotation, "true"))
		}
		return nil
	}
	return AddAnnotations(ctx, obj, ps, mergeAnnotations(annotations, ChainsAnnotation, "true"))
}

func MarkFailed(ctx context.Context, obj objects.TektonObject, ps versioned.Interface, annotations map[string]string) error {
	return AddAnnotations(ctx, obj, ps, mergeAnnotations(annotations, ChainsAnnotation, "failed"))
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
		return AddAnnotations(ctx, obj, ps, mergeAnnotations(annotations, RetryAnnotation, "0"))
	}
	val, err := strconv.Atoi(ann)
	if err != nil {
		return errors.Wrap(err, "adding retry")
	}
	return AddAnnotations(ctx, obj, ps, mergeAnnotations(annotations, RetryAnnotation, fmt.Sprintf("%d", val+1)))
}

// AddAnnotations adds annotation to the k8s object
func AddAnnotations(ctx context.Context, obj objects.TektonObject, ps versioned.Interface, annotations map[string]string) error {
	// Get current annotations from API server to ensure we have the latest state
	currentAnnotations, err := obj.GetLatestAnnotations(ctx, ps)
	if err != nil {
		return err
	}

	// Start with existing chains annotations, ignore annotations from other controllers,
	// so we do not take ownership of them.
	mergedAnnotations := make(map[string]string)
	for k, v := range currentAnnotations {
		if strings.HasPrefix(k, ChainsAnnotationPrefix) {
			mergedAnnotations[k] = v
		}
	}

	// Add the new chains annotations, they all must be chains annotations
	for k, v := range annotations {
		if !strings.HasPrefix(k, ChainsAnnotationPrefix) {
			return fmt.Errorf("invalid annotation key %q: all annotations must have prefix %q", k, ChainsAnnotationPrefix)
		}
		mergedAnnotations[k] = v
	}

	patchBytes, err := CreateAnnotationsPatch(mergedAnnotations, obj)
	if err != nil {
		return err
	}
	err = obj.Patch(ctx, ps, patchBytes)
	if err != nil {
		return err
	}

	// Note: Ideally here we'll update the in-memory object to keep it consistent through
	// the reconciliation loop. It hasn't been done to preserve the existing controller behavior
	// and maintain compatibility with existing tests. This could be revisited in the future.

	return nil
}

// HandleRetry handles retries managed as annotation on the k8s object
func HandleRetry(ctx context.Context, obj objects.TektonObject, ps versioned.Interface, annotationsMap map[string]string) error {
	if RetryAvailable(obj) {
		return AddRetry(ctx, obj, ps, annotationsMap)
	}
	return MarkFailed(ctx, obj, ps, annotationsMap)
}

// CreateAnnotationsPatch returns patch bytes that can be used with kubectl patch
func CreateAnnotationsPatch(newAnnotations map[string]string, obj objects.TektonObject) ([]byte, error) {
	// Get GVK using the TektonObject interface method (more reliable than runtime.Object)
	gvkStr := obj.GetGVK()
	if gvkStr == "" {
		return nil, fmt.Errorf("unable to determine GroupVersionKind for object %s/%s", obj.GetNamespace(), obj.GetName())
	}

	// Parse the string format "group/version/kind"
	parts := strings.Split(gvkStr, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return nil, fmt.Errorf("invalid GVK format: %s", gvkStr)
	}
	apiVersion := parts[0] + "/" + parts[1]
	kind := parts[2]

	// For server-side apply, we need to create a structured patch with metadata
	p := serverSideApplyPatch{
		APIVersion: apiVersion,
		Kind:       kind,
		Metadata: serverSideApplyMetadata{
			Name:        obj.GetName(),
			Namespace:   obj.GetNamespace(),
			Annotations: newAnnotations,
		},
	}
	return json.Marshal(p)
}

// These are used to get proper json formatting for server-side apply
type serverSideApplyPatch struct {
	APIVersion string                  `json:"apiVersion"`
	Kind       string                  `json:"kind"`
	Metadata   serverSideApplyMetadata `json:"metadata"`
}

type serverSideApplyMetadata struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
