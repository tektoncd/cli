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

package task

import (
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/cli/prerun"
	"github.com/tektoncd/cli/pkg/flags"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var taskGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "tasks"}
var taskrunGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}

func Command(p cli.Params) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "task",
		Aliases: []string{"t", "tasks"},
		Short:   "Manage Tasks",
		Annotations: map[string]string{
			"commandType": "main",
		},
		PersistentPreRunE: prerun.PersistentPreRunE(p),
	}

	flags.AddTektonOptions(cmd)
	cmd.AddCommand(
		deleteCommand(p),
		describeCommand(p),
		listCommand(p),
		logCommand(p),
		startCommand(p),
		createCommand(p),
		signCommand(p),
		verifyCommand(p),
	)
	return cmd
}

// Example of an "marked experimental" sub command
// func fooCommand(p cli.Params) *cobra.Command {
// 	return &cobra.Command{
// 		Use:   "foo",
// 		Short: "Foo is bar and baz",
// 		Annotations: map[string]string{
// 			"experimental": "",
// 			"commandType":  "main",
// 		},
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			fmt.Fprintf(cmd.OutOrStdout(), "Foo is bar\n")
// 			return nil
// 		},
// 	}
// }
