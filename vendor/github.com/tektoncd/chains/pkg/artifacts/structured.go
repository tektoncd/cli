/*
Copyright 2023 The Tekton Authors
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

package artifacts

import (
	"context"
	"strings"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"knative.dev/pkg/logging"
)

// StructuredSignable contains info for signable targets to become either subjects or materials in
// intoto Statements.
// URI is the resource uri for the target needed iff the target is a material.
// Digest is the target's SHA digest.
type StructuredSignable struct {
	URI    string
	Digest string
}

type structuredSignableExtractor struct {
	uriSuffix    string
	digestSuffix string
	isValid      func(StructuredSignable) bool
}

func (b *structuredSignableExtractor) extract(ctx context.Context, results []objects.Result) []StructuredSignable {
	logger := logging.FromContext(ctx)
	partials := map[string]StructuredSignable{}

	suffixes := map[string]func(StructuredSignable, string) StructuredSignable{
		b.uriSuffix: func(s StructuredSignable, value string) StructuredSignable {
			s.URI = value
			return s
		},
		b.digestSuffix: func(s StructuredSignable, value string) StructuredSignable {
			s.Digest = value
			return s
		},
	}

	for _, res := range results {
		for suffix, setFn := range suffixes {
			if suffix == "" {
				continue
			}
			if !strings.HasSuffix(res.Name, suffix) {
				continue
			}
			value := strings.TrimSpace(res.Value.StringVal)
			if value == "" {
				logger.Debugf("error getting string value for %s", res.Name)
				continue
			}
			marker := strings.TrimSuffix(res.Name, suffix)
			if _, ok := partials[marker]; !ok {
				partials[marker] = StructuredSignable{}
			}
			partials[marker] = setFn(partials[marker], value)
		}
	}

	var signables []StructuredSignable
	for _, s := range partials {
		if !b.isValid(s) {
			continue
		}
		signables = append(signables, s)
	}

	return signables
}
