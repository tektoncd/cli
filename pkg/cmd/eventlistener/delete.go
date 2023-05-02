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

package eventlistener

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/deleter"
	"github.com/tektoncd/cli/pkg/eventlistener"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

// eventListenerExists validates that the arguments are valid EventListener names
func eventListenerExists(args []string, p cli.Params) ([]string, error) {

	availableELs := make([]string, 0)
	c, err := p.Clients()
	if err != nil {
		return availableELs, err
	}
	var errorList error
	ns := p.Namespace()
	for _, name := range args {
		_, err := eventlistener.Get(c, name, metav1.GetOptions{}, ns)
		if err != nil {
			errorList = multierr.Append(errorList, err)
			continue
		}
		availableELs = append(availableELs, name)
	}
	return availableELs, errorList
}

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "eventlistener", ForceDelete: false, DeleteAllNs: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete EventListeners with names 'foo' and 'bar' in namespace 'bar'

    tkn eventlistener delete foo bar -n quux

or

    tkn el rm foo bar -n quux
`

	c := &cobra.Command{
		Use:               "delete",
		Aliases:           []string{"rm"},
		Short:             "Delete EventListeners in a namespace",
		Example:           eg,
		Args:              cobra.MinimumNArgs(0),
		SilenceUsage:      true,
		ValidArgsFunction: formatted.ParentCompletion,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				In:  cmd.InOrStdin(),
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			availableELs, errs := eventListenerExists(args, p)
			if len(availableELs) == 0 && errs != nil {
				return errs
			}

			if err := opts.CheckOptions(s, availableELs, p.Namespace()); err != nil {
				return err
			}

			if err := deleteEventListeners(s, p, availableELs, opts.DeleteAllNs); err != nil {
				return err
			}
			return errs
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().BoolVarP(&opts.DeleteAllNs, "all", "", false, "Delete all EventListeners in a namespace (default: false)")

	return c
}

func deleteEventListeners(s *cli.Stream, p cli.Params, elNames []string, deleteAll bool) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}
	d := deleter.New("EventListener", func(listenerName string) error {
		return actions.Delete(eventlistenerGroupResource, cs.Dynamic, cs.Triggers.Discovery(), listenerName, p.Namespace(), metav1.DeleteOptions{})
	})
	if deleteAll {
		elNames, err = eventlistener.GetAllEventListenerNames(cs, p.Namespace())
		if err != nil {
			return err
		}
	}
	d.Delete(elNames)

	if !deleteAll {
		d.PrintSuccesses(s)
	} else if deleteAll {
		if d.Errors() == nil {
			fmt.Fprintf(s.Out, "All EventListeners deleted in namespace %q\n", p.Namespace())
		}
	}
	return d.Errors()
}
