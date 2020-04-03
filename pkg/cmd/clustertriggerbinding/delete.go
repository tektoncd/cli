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

package clustertriggerbinding

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/deleter"
	"github.com/tektoncd/cli/pkg/options"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "clustertriggerbinding", ForceDelete: false, DeleteAll: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete ClusterTriggerBindings with names 'foo' and 'bar'

    tkn clustertriggerbinding delete foo bar

or

    tkn ctb rm foo bar
`

	c := &cobra.Command{
		Use:          "delete",
		Aliases:      []string{"rm"},
		Short:        "Delete clustertriggerbindings",
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

			if err := opts.CheckOptions(s, args, p.Namespace()); err != nil {
				return err
			}

			return deleteClusterTriggerBindings(s, p, args, opts.DeleteAll)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().BoolVarP(&opts.DeleteAll, "all", "", false, "Delete all ClusterTriggerBindings (default: false)")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_clustertriggerbinding")
	return c
}

func deleteClusterTriggerBindings(s *cli.Stream, p cli.Params, ctbNames []string, deleteAll bool) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}
	d := deleter.New("ClusterTriggerBinding", func(bindingName string) error {
		return cs.Triggers.TriggersV1alpha1().ClusterTriggerBindings().Delete(bindingName, &metav1.DeleteOptions{})
	})

	if deleteAll {
		ctbNames, err = allClusterTriggerBindingNames(p, cs)
		if err != nil {
			return err
		}
	}
	d.Delete(s, ctbNames)

	if !deleteAll {
		d.PrintSuccesses(s)
	} else if deleteAll {
		if d.Errors() == nil {
			fmt.Fprint(s.Out, "All ClusterTriggerBindings deleted\n")
		}
	}
	return d.Errors()
}

func allClusterTriggerBindingNames(p cli.Params, cs *cli.Clients) ([]string, error) {
	ctbs, err := cs.Triggers.TriggersV1alpha1().ClusterTriggerBindings().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, ctb := range ctbs.Items {
		names = append(names, ctb.Name)
	}
	return names, nil
}
