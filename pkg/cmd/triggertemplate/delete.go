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

package triggertemplate

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/deleter"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/triggertemplate"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

var triggertemplateGroupResource = schema.GroupVersionResource{Group: "triggers.tekton.dev", Resource: "triggertemplates"}

// triggerTemplateExists validates that the arguments are valid TriggerTemplate names
func triggerTemplateExists(args []string, p cli.Params) ([]string, error) {

	availableTts := make([]string, 0)
	c, err := p.Clients()
	if err != nil {
		return availableTts, err
	}
	var errorList error
	ns := p.Namespace()
	for _, name := range args {
		_, err := triggertemplate.Get(c, name, metav1.GetOptions{}, ns)
		if err != nil {
			errorList = multierr.Append(errorList, err)
			continue
		}
		availableTts = append(availableTts, name)
	}
	return availableTts, errorList
}

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "triggertemplate", ForceDelete: false, DeleteAllNs: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete TriggerTemplates with names 'foo' and 'bar' in namespace 'quux'

    tkn triggertemplate delete foo bar -n quux

or

    tkn tt rm foo bar -n quux
`

	c := &cobra.Command{
		Use:               "delete",
		Aliases:           []string{"rm"},
		Short:             "Delete TriggerTemplates in a namespace",
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

			availableTts, errs := triggerTemplateExists(args, p)
			if len(availableTts) == 0 && errs != nil {
				return errs
			}

			if err := opts.CheckOptions(s, availableTts, p.Namespace()); err != nil {
				return err
			}

			if err := deleteTriggerTemplates(s, p, availableTts, opts.DeleteAllNs); err != nil {
				return err
			}
			return errs
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().BoolVarP(&opts.DeleteAllNs, "all", "", false, "Delete all TriggerTemplates in a namespace (default: false)")

	return c
}

func deleteTriggerTemplates(s *cli.Stream, p cli.Params, ttNames []string, deleteAll bool) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}
	d := deleter.New("TriggerTemplate", func(templateName string) error {
		return actions.Delete(triggertemplateGroupResource, cs.Dynamic, cs.Triggers.Discovery(), templateName, p.Namespace(), metav1.DeleteOptions{})
	})

	if deleteAll {
		ttNames, err = allTriggerTemplateNames(p)
		if err != nil {
			return err
		}
	}
	d.Delete(ttNames)

	if !deleteAll {
		d.PrintSuccesses(s)
	} else if deleteAll {
		if d.Errors() == nil {
			fmt.Fprintf(s.Out, "All TriggerTemplates deleted in namespace %q\n", p.Namespace())
		}
	}
	return d.Errors()
}

func allTriggerTemplateNames(p cli.Params) ([]string, error) {
	cs, err := p.Clients()
	if err != nil {
		return nil, err
	}
	tts, err := triggertemplate.List(cs, metav1.ListOptions{}, p.Namespace())
	if err != nil {
		return nil, err
	}
	var names []string
	for _, tt := range tts.Items {
		names = append(names, tt.Name)
	}
	return names, nil
}
