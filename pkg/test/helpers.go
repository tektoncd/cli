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
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jonboulle/clockwork"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	pipelinetest "github.com/tektoncd/pipeline/test"
	triggerstest "github.com/tektoncd/triggers/test"
)

func SeedTestData(t *testing.T, d pipelinetest.Data) (pipelinetest.Clients, pipelinetest.Informers) {
	ctx, _ := ttesting.SetupFakeContext(t)
	return pipelinetest.SeedTestData(t, ctx, d)
}

func SeedV1beta1TestData(t *testing.T, d Data) (Clients, Informers) {
	ctx, _ := ttesting.SetupFakeContext(t)
	return seedTestData(t, ctx, d)
}

func SeedTestResources(t *testing.T, d triggerstest.Resources) triggerstest.Clients {
	ctx, _ := triggerstest.SetupFakeContext(t)
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

// Contains is a test utility to check if obj is in the contents of the container.
func Contains(t *testing.T, container interface{}, obj interface{}) {
	if reflect.TypeOf(container).Kind() != reflect.Slice {
		t.Errorf("obj cannot exist in non-slice %s", reflect.TypeOf(container).Kind().String())
	}

	diffs := []string{}
	val := reflect.ValueOf(container)
	for i := 0; i < val.Len(); i++ {
		diff := cmp.Diff(val.Index(i).Interface(), obj)
		if diff == "" {
			return
		}
		diffs = append(diffs, diff)
	}
	t.Errorf("no matches found in: %s", strings.Join(diffs, "\n"))
}

func FakeClock() *clockwork.FakeClock {
	return clockwork.NewFakeClockAt(time.Date(1984, time.April, 4, 0, 0, 0, 0, time.UTC))
}
