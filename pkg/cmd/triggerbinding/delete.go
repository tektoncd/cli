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

package triggerbinding

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/deleter"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/triggerbinding"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

// triggerBindingExists validates that the arguments are valid TriggerBinding names
func triggerBindingExists(args []string, p cli.Params) ([]string, error) {

	availableTBs := make([]string, 0)
	c, err := p.Clients()
	if err != nil {
		return availableTBs, err
	}
	var errorList error
	ns := p.Namespace()
	for _, name := range args {
		_, err := triggerbinding.Get(c, name, metav1.GetOptions{}, ns)
		if err != nil {
			errorList = multierr.Append(errorList, err)
			continue
		}
		availableTBs = append(availableTBs, name)
	}
	return availableTBs, errorList
}

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "triggerbinding", ForceDelete: false, DeleteAllNs: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete TriggerBindings with names 'foo' and 'bar' in namespace 'quux'

    tkn triggerbinding delete foo bar -n quux

or

    tkn tb rm foo bar -n quux
`

	c := &cobra.Command{
		Use:               "delete",
		Aliases:           []string{"rm"},
		Short:             "Delete TriggerBindings in a namespace",
		Example:           eg,
		ValidArgsFunction: formatted.ParentCompletion,
		Args:              cobra.MinimumNArgs(0),
		SilenceUsage:      true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				In:  cmd.InOrStdin(),
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			availableTbs, errs := triggerBindingExists(args, p)
			if len(availableTbs) == 0 && errs != nil {
				return errs
			}

			if err := opts.CheckOptions(s, availableTbs, p.Namespace()); err != nil {
				return err
			}

			if err := deleteTriggerBindings(s, p, availableTbs, opts.DeleteAllNs); err != nil {
				return err
			}
			return errs
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().BoolVarP(&opts.DeleteAllNs, "all", "", false, "Delete all TriggerBindings in a namespace (default: false)")

	return c
}

func deleteTriggerBindings(s *cli.Stream, p cli.Params, tbNames []string, deleteAll bool) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}
	d := deleter.New("TriggerBinding", func(bindingName string) error {
		return actions.Delete(triggerbindingGroupResource, cs.Dynamic, cs.Triggers.Discovery(), bindingName, p.Namespace(), metav1.DeleteOptions{})
	})

	if deleteAll {
		tbNames, err = triggerbinding.GetAllTriggerBindingNames(cs, p.Namespace())
		if err != nil {
			return err
		}
	}
	d.Delete(tbNames)

	if !deleteAll {
		d.PrintSuccesses(s)
	} else if deleteAll {
		if d.Errors() == nil {
			fmt.Fprintf(s.Out, "All TriggerBindings deleted in namespace %q\n", p.Namespace())
		}
	}
	return d.Errors()
}
