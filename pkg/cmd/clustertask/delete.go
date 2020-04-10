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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/clustertask"
	"github.com/tektoncd/cli/pkg/deleter"
	"github.com/tektoncd/cli/pkg/options"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "clustertask", ForceDelete: false, DeleteAll: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete ClusterTasks with names 'foo' and 'bar':

    tkn clustertask delete foo bar

or

    tkn ct rm foo bar
`

	c := &cobra.Command{
		Use:          "delete",
		Aliases:      []string{"rm"},
		Short:        "Delete clustertask resources in a cluster",
		Example:      eg,
		Args:         cobra.MinimumNArgs(0),
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				In:  cmd.InOrStdin(),
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if err := opts.CheckOptions(s, args, ""); err != nil {
				return err
			}

			return deleteClusterTasks(s, p, args, opts.DeleteAll)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().BoolVarP(&opts.DeleteAll, "all", "", false, "Delete all clustertasks (default: false)")
	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_clustertasks")
	return c
}

func deleteClusterTasks(s *cli.Stream, p cli.Params, tNames []string, deleteAll bool) error {
	ctGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "clustertasks"}

	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("Failed to create tekton client")
	}
	d := deleter.New("ClusterTask", func(taskName string) error {
		return actions.Delete(ctGroupResource, cs, taskName, "", &metav1.DeleteOptions{})
	})
	if deleteAll {
		cts, err := allClusterTaskNames(cs)
		if err != nil {
			return err
		}
		d.Delete(s, cts)
	} else {
		d.Delete(s, tNames)
	}

	if !deleteAll {
		d.PrintSuccesses(s)
	} else if deleteAll {
		if d.Errors() == nil {
			fmt.Fprint(s.Out, "All ClusterTasks deleted\n")
		}
	}
	return d.Errors()
}

func allClusterTaskNames(cs *cli.Clients) ([]string, error) {
	clusterTasks, err := clustertask.List(cs, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, ct := range clusterTasks.Items {
		names = append(names, ct.Name)
	}
	return names, nil
}
