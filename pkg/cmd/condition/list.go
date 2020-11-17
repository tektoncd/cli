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

package condition

import (
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/printer"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	emptyMsg = "No conditions found"
)

type listOptions struct {
	AllNamespaces bool
	NoHeaders     bool
}

func listCommand(p cli.Params) *cobra.Command {
	opts := &listOptions{}
	f := cliopts.NewPrintFlags("list")

	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists Conditions in a namespace",
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			stream := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			output, err := cmd.LocalFlags().GetString("output")
			if err != nil {
				return fmt.Errorf("output option not set properly: %v", err)
			}

			if output != "" {
				return printConditionListObj(stream, p, f, opts.AllNamespaces)
			}
			return printConditionDetails(stream, p, opts.AllNamespaces, opts.NoHeaders)
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.AllNamespaces, "all-namespaces", "A", opts.AllNamespaces, "list Conditions from all namespaces")
	c.Flags().BoolVar(&opts.NoHeaders, "no-headers", opts.NoHeaders, "do not print column headers with output (default print column headers with output)")
	return c
}

func printConditionDetails(s *cli.Stream, p cli.Params, allNamespaces bool, noHeaders bool) error {

	cs, err := p.Clients()
	if err != nil {
		return err
	}

	namespace := p.Namespace()

	if allNamespaces {
		namespace = ""
	}

	conditions, err := listAllConditions(cs.Tekton, namespace)
	if err != nil {
		if allNamespaces {
			return fmt.Errorf("failed to list Conditions from all namespaces: %v", err)
		}
		return fmt.Errorf("failed to list Conditions from %s namespace: %v", p.Namespace(), err)
	}

	if len(conditions.Items) == 0 {
		fmt.Fprintln(s.Err, emptyMsg)
		return nil
	}

	w := tabwriter.NewWriter(s.Out, 0, 5, 3, ' ', tabwriter.TabIndent)
	headers := "NAME\tDESCRIPTION\tAGE"
	body := "%s\t%s\t%s\n"
	if allNamespaces {
		headers = "NAMESPACE\t" + headers
		body = "%s\t" + body
	}
	if !noHeaders {
		fmt.Fprintln(w, headers)
	}

	for _, condition := range conditions.Items {
		if allNamespaces {
			fmt.Fprintf(w, body,
				condition.Namespace,
				condition.Name,
				formatted.FormatDesc(condition.Spec.Description),
				formatted.Age(&condition.CreationTimestamp, p.Time()),
			)
		} else {
			fmt.Fprintf(w, body,
				condition.Name,
				formatted.FormatDesc(condition.Spec.Description),
				formatted.Age(&condition.CreationTimestamp, p.Time()),
			)
		}
	}
	return w.Flush()
}

func printConditionListObj(s *cli.Stream, p cli.Params, f *cliopts.PrintFlags, allNamespaces bool) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	namespace := p.Namespace()
	if allNamespaces {
		namespace = ""
	}

	conditions, err := listAllConditions(cs.Tekton, namespace)
	if err != nil {
		if allNamespaces {
			return fmt.Errorf("failed to list Conditions from all namespaces: %v", err)
		}
		return fmt.Errorf("failed to list Conditions from %s namespace: %v", p.Namespace(), err)
	}
	return printer.PrintObject(s.Out, conditions, f)
}

func listAllConditions(cs versioned.Interface, ns string) (*v1alpha1.ConditionList, error) {
	c := cs.TektonV1alpha1().Conditions(ns)

	conditions, err := c.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// NOTE: this is required for -o json|yaml to work properly since
	// tektoncd go client fails to set these; probably a bug
	conditions.GetObjectKind().SetGroupVersionKind(
		schema.GroupVersionKind{
			Version: "tekton.dev/v1alpha1",
			Kind:    "ConditionList",
		})
	return conditions, nil
}
