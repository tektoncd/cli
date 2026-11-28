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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"
	"knative.dev/pkg/apis"
)

// generatedNameSuffix stands in for the random suffix Kubernetes appends to metadata.generateName.
const generatedNameSuffix = "xxxxx"

// ValidateTektonObjects decodes every Tekton object in contents, applies the
// defaults and runs the same validation the Tekton webhook runs on create, so
// a bundle can be checked before it is pushed. Decoding is strict, so unknown
// fields and duplicate keys are reported, and manifests that only set
// metadata.generateName are accepted. All problems are returned together
// instead of stopping at the first one.
func ValidateTektonObjects(ctx context.Context, contents []string) error {
	ctx = apis.WithinCreate(ctx)
	decoder := serializer.NewCodecFactory(scheme.Scheme, serializer.EnableStrict).UniversalDeserializer()
	var errs []error
	for _, content := range contents {
		docs, splitErr := splitDocuments(content)
		for i, raw := range docs {
			doc := i + 1
			if jsonDoc, err := yaml.ToJSON(raw); err != nil {
				errs = append(errs, fmt.Errorf("found a spec that isn't YAML or JSON parseable (document %d): %w", doc, err))
				continue
			} else if bytes.Equal(bytes.TrimSpace(jsonDoc), []byte("null")) {
				continue
			}

			obj, gvk, err := decoder.Decode(raw, nil, nil)
			if err != nil {
				if !runtime.IsStrictDecodingError(err) || obj == nil || gvk == nil {
					errs = append(errs, fmt.Errorf("failed to parse string as a Tekton object (document %d): %w: %q", doc, sanitizeDecodeError(err), truncate(string(bytes.TrimSpace(raw)))))
					continue
				}
				// the object is usable, so report the unknown or duplicate fields and keep validating it
				errs = append(errs, fmt.Errorf("%s (document %d): %w", gvk.Kind, doc, err))
			}
			if gvk == nil {
				errs = append(errs, fmt.Errorf("failed to parse raw tekton object with no kind or apiVersion (document %d): %s", doc, truncate(string(bytes.TrimSpace(raw)))))
				continue
			}

			name, err := validationName(obj)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", gvk.Kind, err))
				continue
			}
			if d, ok := obj.(apis.Defaultable); ok {
				d.SetDefaults(ctx)
			}
			if v, ok := obj.(apis.Validatable); ok {
				if fe := v.Validate(ctx); fe != nil {
					errs = append(errs, fmt.Errorf("%s %s is invalid: %w", gvk.Kind, name, fe))
				}
			}
		}
		if splitErr != nil {
			errs = append(errs, splitErr)
		}
	}
	return errors.Join(errs...)
}

// validationName returns the name used to report on obj. An object that only has
// metadata.generateName gets a placeholder suffix, like the API server would add,
// so it can be validated before it exists.
func validationName(obj runtime.Object) (string, error) {
	if name, err := getObjectName(obj); err == nil {
		return name, nil
	}
	metaObj, ok := obj.(metav1.Object)
	if !ok || metaObj.GetGenerateName() == "" {
		return getObjectName(obj)
	}
	generated := metaObj.GetGenerateName() + generatedNameSuffix
	metaObj.SetName(generated)
	return metaObj.GetGenerateName() + "*", nil
}

// splitDocuments cuts contents into its raw YAML or JSON documents without converting them,
// so strict decoding sees the original keys. Documents read before a split error are still returned.
func splitDocuments(contents string) ([][]byte, error) {
	var docs [][]byte
	if yaml.IsJSONBuffer([]byte(strings.TrimSpace(contents))) {
		dec := json.NewDecoder(strings.NewReader(contents))
		for {
			var raw json.RawMessage
			if err := dec.Decode(&raw); err != nil {
				if err == io.EOF {
					return docs, nil
				}
				return docs, fmt.Errorf("found a spec that isn't YAML or JSON parseable (document %d): %w", len(docs)+1, err)
			}
			docs = append(docs, raw)
		}
	}
	reader := yaml.NewYAMLReader(bufio.NewReader(strings.NewReader(contents)))
	for {
		raw, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				return docs, nil
			}
			return docs, fmt.Errorf("found a spec that isn't YAML or JSON parseable (document %d): %w", len(docs)+1, err)
		}
		docs = append(docs, raw)
	}
}
