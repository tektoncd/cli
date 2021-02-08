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

package test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	triggerstest "github.com/tektoncd/triggers/test"
)

// SetupFakeContext aliases tektoncd/pipeline so we don't need to import ttesting everywhere
var SetupFakeContext = ttesting.SetupFakeContext

func SeedTestData(t *testing.T, d pipelinetest.Data) (pipelinetest.Clients, pipelinetest.Informers) {
	ctx, _ := ttesting.SetupFakeContext(t)
	return pipelinetest.SeedTestData(t, ctx, d)
}

func SeedV1beta1TestData(t *testing.T, d pipelinev1beta1test.Data) (pipelinev1beta1test.Clients, pipelinev1beta1test.Informers) {
	ctx, _ := ttesting.SetupFakeContext(t)
	return pipelinev1beta1test.SeedTestData(t, ctx, d)
}

func SeedTestResources(t *testing.T, d triggerstest.Resources) triggerstest.Clients {
	ctx, _ := ttesting.SetupFakeContext(t)
	return triggerstest.SeedResources(t, ctx, d)
}

func AssertOutput(t *testing.T, expected, actual interface{}) {
	t.Helper()
	diff := cmp.Diff(actual, expected)
	if diff == "" {
		return
	}

	t.Errorf(`
Unexpected output:
%s

Expected
%s

Actual
%s
`, diff, expected, actual)
}

func AssertOutputPrefix(t *testing.T, expected, actual string) {
	t.Helper()

	if !strings.HasPrefix(actual, expected) {
		t.Errorf(`
Unexpected output:

Expected prefix
%s

Actual
%s
`, expected, actual)
	}
}
