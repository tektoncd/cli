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
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/helper/deleter"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/validate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "triggertemplate", ForceDelete: false, DeleteAllNs: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete TriggerTemplates with names 'foo' and 'bar' in namespace 'quux'

    tkn triggertemplate delete foo bar -n quux

or

    tkn tt rm foo bar -n quux
`

	c := &cobra.Command{
		Use:          "delete",
		Aliases:      []string{"rm"},
		Short:        "Delete triggertemplates in a namespace",
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

			if err := validate.NamespaceExists(p); err != nil {
				return err
			}

			if err := opts.CheckOptions(s, args, p.Namespace()); err != nil {
				return err
			}

			return deleteTriggerTemplates(s, p, args, opts.DeleteAllNs)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().BoolVarP(&opts.DeleteAllNs, "all", "", false, "Delete all TriggerTemplates in a namespace (default: false)")

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_triggertemplate")
	return c
}

func deleteTriggerTemplates(s *cli.Stream, p cli.Params, ttNames []string, deleteAll bool) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}
	d := deleter.New("TriggerTemplate", func(templateName string) error {
		return cs.Triggers.TektonV1alpha1().TriggerTemplates(p.Namespace()).Delete(templateName, &metav1.DeleteOptions{})
	})

	if deleteAll {
		ttNames, err = allTriggerTemplateNames(p, cs)
		if err != nil {
			return err
		}
	}
	d.Delete(s, ttNames)

	if !deleteAll {
		d.PrintSuccesses(s)
	} else if deleteAll {
		if d.Errors() == nil {
			fmt.Fprintf(s.Out, "All TriggerTemplates deleted in namespace %q\n", p.Namespace())
		}
	}
	return d.Errors()
}

func allTriggerTemplateNames(p cli.Params, cs *cli.Clients) ([]string, error) {
	tts, err := cs.Triggers.TektonV1alpha1().TriggerTemplates(p.Namespace()).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, tt := range tts.Items {
		names = append(names, tt.Name)
	}
	return names, nil
}
