// Copyright © 2026 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package bundle

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
)

// ValidateTektonObjects decodes every Tekton object in contents, applies the
// defaults and runs the same validation the Tekton webhook runs on create, so
// a bundle can be checked before it is pushed. All problems are returned
// together instead of stopping at the first one.
func ValidateTektonObjects(ctx context.Context, contents []string) error {
	ctx = apis.WithinCreate(ctx)
	var errs []error
	for _, content := range contents {
		if err := decodeObjects(content, func(gvk *schema.GroupVersionKind, obj runtime.Object, _ []byte) error {
			name, err := getObjectName(obj)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", gvk.Kind, err))
				return nil
			}
			if d, ok := obj.(apis.Defaultable); ok {
				d.SetDefaults(ctx)
			}
			if v, ok := obj.(apis.Validatable); ok {
				if fe := v.Validate(ctx); fe != nil {
					errs = append(errs, fmt.Errorf("%s %s is invalid: %w", gvk.Kind, name, fe))
				}
			}
			return nil
		}); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
