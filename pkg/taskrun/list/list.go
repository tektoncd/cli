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

package list

import (
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	trsort "github.com/tektoncd/cli/pkg/taskrun/sort"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetAllTaskRuns(p cli.Params, opts metav1.ListOptions, limit int) ([]string, error) {
	cs, err := p.Clients()
	if err != nil {
		return nil, err
	}

	runs, err := TaskRuns(cs, opts, p.Namespace())
	if err != nil {
		return nil, err
	}

	trsort.SortByStartTime(runs.Items)
	runslen := len(runs.Items)
	if limit > runslen {
		limit = runslen
	}

	ret := []string{}
	for i, run := range runs.Items {
		if i < limit {
			ret = append(ret, run.ObjectMeta.Name+" started "+formatted.Age(run.Status.StartTime, p.Time()))
		}
	}
	return ret, nil
}

func TaskRuns(c *cli.Clients, opts metav1.ListOptions, ns string) (*v1beta1.TaskRunList, error) {
	trGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}
	unstructuredTRL, err := actions.List(trGroupResource, c.Dynamic, c.Tekton.Discovery(), ns, opts)
	if err != nil {
		return nil, err
	}

	var runs *v1beta1.TaskRunList
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredTRL.UnstructuredContent(), &runs); err != nil {
		return nil, err
	}

	return runs, nil
}
