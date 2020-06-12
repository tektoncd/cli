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
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/options"
	trdesc "github.com/tektoncd/cli/pkg/taskrun/description"
	trlist "github.com/tektoncd/cli/pkg/taskrun/list"
	"github.com/tektoncd/cli/pkg/validate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	defaultTaskRunLimit = 5
)

func describeCommand(p cli.Params) *cobra.Command {
	opts := &options.DescribeOptions{Params: p}
	f := cliopts.NewPrintFlags("describe")
	eg := `Describe a TaskRun of name 'foo' in namespace 'bar':

    tkn taskrun describe foo -n bar

or

    tkn tr desc foo -n bar
`

	c := &cobra.Command{
		Use:          "describe",
		Aliases:      []string{"desc"},
		Short:        "Describe a TaskRun in a namespace",
		Example:      eg,
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}
			if err := validate.NamespaceExists(p); err != nil {
				return err
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}

			if !opts.Fzf {
				if _, ok := os.LookupEnv("TKN_USE_FZF"); ok {
					opts.Fzf = true
				}
			}

			if len(args) == 0 {
				if !opts.Last {
					err = askTaskRunName(opts, p)
					if err != nil {
						return err
					}
				} else {
					lOpts := metav1.ListOptions{}
					trs, err := trlist.GetAllTaskRuns(p, lOpts, 1)
					if err != nil {
						return err
					}
					opts.TaskrunName = strings.Fields(trs[0])[0]
				}
			} else {
				opts.TaskrunName = args[0]
			}

			if output != "" {
				taskRunGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}
				return actions.PrintObject(taskRunGroupResource, opts.TaskrunName, cmd.OutOrStdout(), p, f, p.Namespace())
			}

			return trdesc.PrintTaskRunDescription(s, opts.TaskrunName, p)
		},
	}

	c.Flags().BoolVarP(&opts.Last, "last", "L", false, "show description for last TaskRun")
	c.Flags().IntVarP(&opts.Limit, "limit", "", defaultTaskRunLimit, "lists number of TaskRuns when selecting a TaskRun to describe")
	c.Flags().BoolVarP(&opts.Fzf, "fzf", "F", false, "use fzf to select a taskrun to describe")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_taskrun")
	f.AddFlags(c)

	return c
}

func askTaskRunName(opts *options.DescribeOptions, p cli.Params) error {
	lOpts := metav1.ListOptions{}

	err := opts.ValidateOpts()
	if err != nil {
		return err
	}

	trs, err := trlist.GetAllTaskRuns(opts.Params, lOpts, opts.Limit)
	if err != nil {
		return err
	}
	if len(trs) == 0 {
		return fmt.Errorf("no TaskRuns found")
	}

	if opts.Fzf {
		err = opts.FuzzyAsk(options.ResourceNameTaskRun, trs)
	} else {
		err = opts.Ask(options.ResourceNameTaskRun, trs)
	}
	if err != nil {
		return err
	}

	return nil
}
