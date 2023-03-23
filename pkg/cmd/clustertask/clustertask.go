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

package clustertask

import (
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cli/prerun"
	"github.com/tektoncd/cli/pkg/flags"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var taskrunGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}
var clustertaskGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "clustertasks"}

func Command(p cli.Params) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clustertask",
		Aliases: []string{"ct", "clustertasks"},
		Short:   "Manage ClusterTasks",
		Annotations: map[string]string{
			"commandType": "main",
		},
		PersistentPreRunE: prerun.PersistentPreRunE(p),
	}

	flags.AddTektonOptions(cmd)
	_ = cmd.PersistentFlags().MarkHidden("namespace")
	cmd.AddCommand(
		deleteCommand(p),
		describeCommand(p),
		listCommand(p),
		startCommand(p),
		logCommand(p),
		createCommand(p),
	)
	cmd.Deprecated = "ClusterTasks are deprecated, this command will be removed in future releases."
	return cmd
}
