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

package taskrun

import (
	"fmt"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	trsort "github.com/tektoncd/cli/pkg/taskrun/sort"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var taskrunGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}

func GetAllTaskRuns(gr schema.GroupVersionResource, opts metav1.ListOptions, c *cli.Clients, ns string, limit int, time clockwork.Clock) ([]string, error) {
	var taskruns *v1.TaskRunList
	if err := actions.ListV1(gr, c, opts, ns, &taskruns); err != nil {
		return nil, fmt.Errorf("failed to list TaskRuns from namespace %s: %v", ns, err)
	}

	runslen := len(taskruns.Items)
	if limit > runslen {
		limit = runslen
	}

	trsort.SortByStartTime(taskruns.Items)
	ret := []string{}
	for i, run := range taskruns.Items {
		if i < limit {
			ret = append(ret, run.ObjectMeta.Name+" started "+formatted.Age(run.Status.StartTime, time))
		}
	}
	return ret, nil
}
