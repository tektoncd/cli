// Copyright © 2020 The Tekton Authors.
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

package suggestion

import (
	"fmt"
	"testing"

	"github.com/tektoncd/cli/pkg/cmd/task"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSuggestion(t *testing.T) {
	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
	}
	cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"task"})
	tdc := testDynamic.Options{}
	dynamic, err := tdc.Client()
	if err != nil {
		fmt.Println(err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dynamic}
	task := task.Command(p)
	args := []string{"l"}
	err = SubcommandsRequiredWithSuggestions(task, args)
	test.AssertOutput(t, "unknown command \"l\" for \"task\"\n\nDid you mean this?\n\tlist\n\tlogs\n", err.Error())
}
